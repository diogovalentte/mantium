package manga

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/lib/pq"

	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/util"
)

// Preset is a set of custom manga last released chapter selectors that work for many sites,
// like sites using the same WordPress theme, so the user doesn't have to write them.
type Preset struct {
	NameSelector *HTMLSelector `json:"name_selector"`
	URLSelector  *HTMLSelector `json:"url_selector"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	// Domains are the sites the preset is selected for automatically, like "example.com".
	// Subdomains also match.
	Domains    []string `json:"domains"`
	ID         int      `json:"id"`
	UseBrowser bool     `json:"use_browser"`
	// Builtin presets come with Mantium and are recreated every time the API starts.
	Builtin bool `json:"builtin"`
}

// chapterNumberRegex gets the first number of the text, like "12.5" from "Chapter 12.5 - The end".
const chapterNumberRegex = `(\d+(?:\.\d+)?)`

// BuiltinPresets are the presets that come with Mantium.
var BuiltinPresets = []*Preset{
	{
		Name:         "Madara",
		Description:  "WordPress sites using the Madara theme, common among scanlation sites. The chapter list has the newest chapter first.",
		NameSelector: &HTMLSelector{Selector: "css:li.wp-manga-chapter > a", Regex: chapterNumberRegex, GetFirst: true},
		URLSelector:  &HTMLSelector{Selector: "css:li.wp-manga-chapter > a", Attribute: "href", GetFirst: true},
	},
	{
		Name:         "MangaReader (Themesia)",
		Description:  "WordPress sites using the MangaReader theme by Themesia. The chapter list has the newest chapter first.",
		NameSelector: &HTMLSelector{Selector: "css:#chapterlist li", Attribute: "data-num", GetFirst: true},
		URLSelector:  &HTMLSelector{Selector: "css:#chapterlist li .eph-num a", Attribute: "href", GetFirst: true},
	},
	{
		Name:         "RSS feed",
		Description:  "RSS feed of a single series, with one item per chapter and the newest first. The chapter is the first number in the item title.",
		NameSelector: &HTMLSelector{Selector: "xpath://item/title", Regex: chapterNumberRegex, GetFirst: true},
		URLSelector:  &HTMLSelector{Selector: "xpath://item/link", GetFirst: true},
	},
	{
		Name:         "RSS feed (titles only)",
		Description:  "Like the RSS feed preset, for feeds whose items have no link, like MangaUpdates' series feeds.",
		NameSelector: &HTMLSelector{Selector: "xpath://item/title", Regex: chapterNumberRegex, GetFirst: true},
	},
}

// UpsertBuiltinPresetsDB creates or updates the builtin presets in the database,
// and deletes builtin presets that don't exist anymore.
func UpsertBuiltinPresetsDB() error {
	contextError := "error upserting builtin presets in DB"

	conn, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	tx, err := conn.Begin()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	names := make([]string, 0, len(BuiltinPresets))
	for _, p := range BuiltinPresets {
		names = append(names, p.Name)
		_, err = tx.Exec(`
            INSERT INTO custom_manga_presets (
                name, description, domains, builtin, use_browser,
                name_selector, name_attribute, name_regex, name_get_first,
                url_selector, url_attribute, url_get_first
            )
            VALUES ($1, $2, $3, TRUE, $4, $5, $6, $7, $8, $9, $10, $11)
            ON CONFLICT (name) DO UPDATE SET
                description = EXCLUDED.description, domains = EXCLUDED.domains, use_browser = EXCLUDED.use_browser,
                name_selector = EXCLUDED.name_selector, name_attribute = EXCLUDED.name_attribute,
                name_regex = EXCLUDED.name_regex, name_get_first = EXCLUDED.name_get_first,
                url_selector = EXCLUDED.url_selector, url_attribute = EXCLUDED.url_attribute,
                url_get_first = EXCLUDED.url_get_first
            WHERE custom_manga_presets.builtin;
        `, append([]any{p.Name, p.Description, pq.Array(nonNilDomains(p.Domains)), p.UseBrowser}, presetSelectorValues(p)...)...)
		if err != nil {
			tx.Rollback()
			return util.AddErrorContext(contextError, err)
		}
	}

	_, err = tx.Exec(`DELETE FROM custom_manga_presets WHERE builtin AND NOT (name = ANY($1));`, pq.Array(names))
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(contextError, err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	return nil
}

// GetPresetsDB gets all presets from the database, builtin presets first.
func GetPresetsDB() ([]*Preset, error) {
	contextError := "error getting presets from DB"

	conn, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	rows, err := conn.Query(`
        SELECT
            id, name, description, domains, builtin, use_browser,
            name_selector, name_attribute, name_regex, name_get_first,
            url_selector, url_attribute, url_get_first
        FROM custom_manga_presets
        ORDER BY builtin DESC, id;
    `)
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}
	defer rows.Close()

	presets := []*Preset{}
	for rows.Next() {
		p := &Preset{}
		var nameSelector, URLSelector HTMLSelector
		err = rows.Scan(
			&p.ID, &p.Name, &p.Description, pq.Array(&p.Domains), &p.Builtin, &p.UseBrowser,
			&nameSelector.Selector, &nameSelector.Attribute, &nameSelector.Regex, &nameSelector.GetFirst,
			&URLSelector.Selector, &URLSelector.Attribute, &URLSelector.GetFirst,
		)
		if err != nil {
			return nil, util.AddErrorContext(contextError, err)
		}
		if nameSelector.Selector != "" {
			p.NameSelector = &nameSelector
		}
		if URLSelector.Selector != "" {
			p.URLSelector = &URLSelector
		}
		p.Domains = nonNilDomains(p.Domains)
		presets = append(presets, p)
	}
	if err := rows.Err(); err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	return presets, nil
}

// InsertPresetDB saves a new preset created by the user in the database.
func InsertPresetDB(p *Preset) error {
	contextError := fmt.Sprintf("error inserting preset '%s' in DB", p.Name)

	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return &errordefs.UserError{Message: "Provide a name for the preset"}
	}
	if p.NameSelector == nil && p.URLSelector == nil {
		return &errordefs.UserError{Message: "Provide at least one selector for the preset"}
	}
	for _, s := range []struct {
		selector *HTMLSelector
		field    string
	}{{p.NameSelector, chapterNameField}, {p.URLSelector, chapterURLField}} {
		if s.selector == nil {
			continue
		}
		if err := selectorPrefixError(s.selector, s.field); err != nil {
			return err
		}
	}
	domains, err := normalizeDomains(p.Domains)
	if err != nil {
		return err
	}
	p.Domains = domains
	if p.URLSelector != nil {
		p.URLSelector.Regex = ""
	}

	conn, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	err = conn.QueryRow(`
        INSERT INTO custom_manga_presets (
            name, description, domains, builtin, use_browser,
            name_selector, name_attribute, name_regex, name_get_first,
            url_selector, url_attribute, url_get_first
        )
        VALUES ($1, $2, $3, FALSE, $4, $5, $6, $7, $8, $9, $10, $11)
        RETURNING id;
    `, append([]any{p.Name, p.Description, pq.Array(p.Domains), p.UseBrowser}, presetSelectorValues(p)...)...).Scan(&p.ID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return &errordefs.UserError{Message: fmt.Sprintf("A preset named \"%s\" already exists", p.Name)}
		}
		return util.AddErrorContext(contextError, err)
	}

	return nil
}

// DeletePresetDB deletes a preset created by the user. Builtin presets can't be deleted.
func DeletePresetDB(id int) error {
	contextError := fmt.Sprintf("error deleting preset '%d' from DB", id)

	conn, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	var builtin bool
	err = conn.QueryRow(`SELECT builtin FROM custom_manga_presets WHERE id = $1;`, id).Scan(&builtin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &errordefs.UserError{Message: "Preset not found"}
		}
		return util.AddErrorContext(contextError, err)
	}
	if builtin {
		return &errordefs.UserError{Message: "Presets that come with Mantium can't be deleted"}
	}

	_, err = conn.Exec(`DELETE FROM custom_manga_presets WHERE id = $1;`, id)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	return nil
}

// PresetDetection is the result of applying a preset to a manga page.
type PresetDetection struct {
	Preset *Preset `json:"preset"`
	// Chapter is the last released chapter found by the preset, nil if the preset doesn't work for the page.
	Chapter *Chapter `json:"chapter"`
	// UserMessage tells why the preset doesn't work for the page.
	UserMessage string `json:"user_message"`
}

// DetectPresets applies each preset to the manga page and tells which work.
// The page is fetched only once, or twice if some presets use the browser.
func DetectPresets(mangaURL string, presets []*Preset) ([]*PresetDetection, error) {
	contextError := fmt.Sprintf("error detecting presets for page '%s'", mangaURL)

	var page, browserPage *scrapedPage
	var browserErr error
	detections := make([]*PresetDetection, 0, len(presets))
	for _, p := range presets {
		detection := &PresetDetection{Preset: p}
		detections = append(detections, detection)

		var pageToUse *scrapedPage
		if p.UseBrowser {
			if browserPage == nil && browserErr == nil {
				browserPage, browserErr = getPageUsingBrowser(mangaURL, nil, nil)
			}
			if browserErr != nil {
				detection.UserMessage = userMessage(browserErr)
				continue
			}
			pageToUse = browserPage
		} else {
			if page == nil {
				var err error
				page, err = getPageUsingHTTPRequest(mangaURL)
				if err != nil {
					return nil, util.AddErrorContext(contextError, err)
				}
			}
			pageToUse = page
		}

		chapter, err := chapterFromPage(pageToUse, p.NameSelector, p.URLSelector)
		if err != nil {
			detection.UserMessage = userMessage(err)
			continue
		}
		detection.Chapter = chapter
	}

	return detections, nil
}

// PresetsForURL returns the presets whose domains match the URL host.
func PresetsForURL(rawURL string, presets []*Preset) []*Preset {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return []*Preset{}
	}
	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")

	matched := []*Preset{}
	for _, p := range presets {
		for _, domain := range p.Domains {
			if host == domain || strings.HasSuffix(host, "."+domain) {
				matched = append(matched, p)
				break
			}
		}
	}

	return matched
}

func userMessage(err error) string {
	var userErr *errordefs.UserError
	if errors.As(err, &userErr) {
		return userErr.Message
	}
	return err.Error()
}

// normalizeDomains turns things like "https://www.Example.com/manga" into "example.com".
func normalizeDomains(domains []string) ([]string, error) {
	normalized := []string{}
	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}
		if !strings.Contains(domain, "://") {
			domain = "http://" + domain
		}
		parsed, err := url.Parse(domain)
		if err != nil || parsed.Hostname() == "" {
			return nil, &errordefs.UserError{Message: fmt.Sprintf("Invalid preset domain: %s", domain)}
		}
		normalized = append(normalized, strings.TrimPrefix(parsed.Hostname(), "www."))
	}

	return normalized, nil
}

func nonNilDomains(domains []string) []string {
	if domains == nil {
		return []string{}
	}
	return domains
}

func presetSelectorValues(p *Preset) []any {
	nameSelector, URLSelector := &HTMLSelector{}, &HTMLSelector{}
	if p.NameSelector != nil {
		nameSelector = p.NameSelector
	}
	if p.URLSelector != nil {
		URLSelector = p.URLSelector
	}
	return []any{
		nameSelector.Selector, nameSelector.Attribute, nameSelector.Regex, nameSelector.GetFirst,
		URLSelector.Selector, URLSelector.Attribute, URLSelector.GetFirst,
	}
}
