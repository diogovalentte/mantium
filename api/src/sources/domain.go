package sources

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/diogovalentte/mantium/api/src/util"
)

// movingSource is a source whose domain changes over time, like KLManga, which
// moves every few weeks and redirects the old domains to the new one.
type movingSource interface {
	SiteURL() *util.SiteURL
}

// ForcedSourcesURLs forces the base URL of a source for every Mantium instance,
// skipping the discovery. It's for when a source changes its domain and the old
// one doesn't redirect to the new one, so the discovery can't follow it.
// Example: "klmanga": "https://klmanga.zone".
var ForcedSourcesURLs = map[string]string{}

var updateDomainsMu sync.Mutex

// UpdateSourcesDomains finds the current domain of every moving source and
// rewrites the URLs of its mangas in the DB to it.
//
// The domain comes from ForcedSourcesURLs when the source is there. Otherwise
// it's found by following the redirects of the last known domain and of the
// domains stored in the DB.
// A source whose domain can't be found keeps the current one.
func UpdateSourcesDomains(log *zerolog.Logger) error {
	updateDomainsMu.Lock()
	defer updateDomainsMu.Unlock()

	var errs []string
	for _, name := range slices.Sorted(maps.Keys(Sources)) {
		source, ok := Sources[name].(movingSource)
		if !ok {
			continue
		}
		if err := updateSourceDomain(name, source.SiteURL(), log); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("error updating the sources domains: %s", strings.Join(errs, "; "))
	}

	return nil
}

func updateSourceDomain(name string, siteURL *util.SiteURL, log *zerolog.Logger) error {
	current := siteURL.Get()

	newURL := ForcedSourcesURLs[name]
	if newURL == "" {
		stored, err := getSourceStoredBaseURLs(name)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not get the %s domains stored in the DB, trying only %s", name, current)
		}

		candidates := []string{current}
		for _, u := range stored {
			if !slices.Contains(candidates, u) {
				candidates = append(candidates, u)
			}
		}

		newURL = discoverSourceURL(name, candidates, log)
		if newURL == "" {
			log.Warn().Msgf("Could not find the current %s domain, keeping %s", name, current)
			newURL = current
		}
	}

	if newURL != current {
		log.Info().Msgf("The %s domain changed from %s to %s", name, current, newURL)
		siteURL.Set(newURL)
	}

	return changeSourceBaseURLInDB(name, newURL)
}

// discoverSourceURL returns where the first candidate that answers redirects
// to, or "" if none does.
func discoverSourceURL(name string, candidates []string, log *zerolog.Logger) string {
	for _, candidate := range candidates {
		final, err := util.FollowSiteRedirects(candidate)
		if err != nil {
			log.Debug().Err(err).Msgf("Domain candidate %s for %s didn't answer", candidate, name)
			continue
		}

		// The URLs are matched to a source by its name in the host, and a
		// redirect to an unrelated host is more likely an ad or parking page.
		parsed, _ := url.Parse(final)
		if !strings.Contains(parsed.Hostname(), name) {
			log.Warn().Msgf("Domain candidate %s for %s redirects to %s, which doesn't look like %s. Ignoring it", candidate, name, final, name)
			continue
		}

		return final
	}

	return ""
}

// getSourceStoredBaseURLs returns the base URLs of the source's mangas in the
// DB, the most used first.
func getSourceStoredBaseURLs(sourceName string) ([]string, error) {
	contextError := "error getting the base URLs of source '%s' from DB"

	_db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, sourceName), err)
	}

	const query = `
		SELECT base_url
		FROM (
			SELECT SUBSTRING(url FROM '^https?://[^/]+') AS base_url
			FROM mangas
			WHERE source = $1
		) urls
		WHERE base_url IS NOT NULL
		GROUP BY base_url
		ORDER BY COUNT(*) DESC
	`
	rows, err := _db.Query(query, sourceName)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, sourceName), err)
	}
	defer rows.Close()

	var baseURLs []string
	for rows.Next() {
		var baseURL string
		if err := rows.Scan(&baseURL); err != nil {
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, sourceName), err)
		}
		baseURLs = append(baseURLs, baseURL)
	}
	if err := rows.Err(); err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, sourceName), err)
	}

	return baseURLs, nil
}

// changeSourceBaseURLInDB points the URLs of the source's mangas and chapters
// to baseURL, keeping the path.
func changeSourceBaseURLInDB(sourceName, baseURL string) error {
	contextError := "error changing the base URL of source '%s' to '%s' in DB"

	_db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, sourceName, baseURL), err)
	}

	tx, err := _db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, sourceName, baseURL), err)
	}
	defer tx.Rollback()

	// Both run on every start and before every update, so they skip the rows
	// that are already right instead of rewriting the tables. url is part of
	// the primary key in both, so a row whose new URL is taken, by an existing
	// row or by another row moving to the same URL, is skipped.
	const mangasQuery = `
		WITH moving AS (
			SELECT DISTINCT ON (new_url) id, new_url
			FROM (
				SELECT id, REGEXP_REPLACE(url, '^https?://[^/]+', $1) AS new_url
				FROM mangas
				WHERE source = $2
					AND SUBSTRING(url FROM '^https?://[^/]+') IS DISTINCT FROM $1
			) candidates
			WHERE NOT EXISTS (SELECT 1 FROM mangas other WHERE other.url = candidates.new_url)
			ORDER BY new_url, id
		)
		UPDATE mangas
		SET url = moving.new_url
		FROM moving
		WHERE mangas.id = moving.id
	`
	if _, err = tx.Exec(mangasQuery, baseURL, sourceName); err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, sourceName, baseURL), err)
	}

	// The last released chapter belongs to a manga of the source. The last
	// read chapter belongs to a multimanga, which can mix sources, so it's
	// matched by the source name in the host, like urlToSource does.
	const chaptersQuery = `
		WITH moving AS (
			SELECT DISTINCT ON (new_url, type) id, new_url
			FROM (
				SELECT c.id, c.type, REGEXP_REPLACE(c.url, '^https?://[^/]+', $1) AS new_url
				FROM chapters c
				WHERE SUBSTRING(c.url FROM '^https?://[^/]+') IS DISTINCT FROM $1
					AND (
						c.manga_id IN (SELECT id FROM mangas WHERE source = $2)
						OR (c.multimanga_id IS NOT NULL AND SUBSTRING(c.url FROM '^https?://([^/:]+)') LIKE '%' || $2 || '%')
					)
			) candidates
			WHERE NOT EXISTS (
				SELECT 1 FROM chapters other
				WHERE other.url = candidates.new_url AND other.type = candidates.type
			)
			ORDER BY new_url, type, id
		)
		UPDATE chapters
		SET url = moving.new_url
		FROM moving
		WHERE chapters.id = moving.id
	`
	if _, err = tx.Exec(chaptersQuery, baseURL, sourceName); err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, sourceName, baseURL), err)
	}

	if err = tx.Commit(); err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, sourceName, baseURL), err)
	}

	return nil
}
