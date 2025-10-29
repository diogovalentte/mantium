// Package main implements the init and main function
package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/mendoncart/mantium/api/src"
	"github.com/mendoncart/mantium/api/src/config"
	"github.com/mendoncart/mantium/api/src/dashboard"
	"github.com/mendoncart/mantium/api/src/db"
	"github.com/mendoncart/mantium/api/src/manga"
	"github.com/mendoncart/mantium/api/src/sources"
	"github.com/mendoncart/mantium/api/src/sources/mangadex"
	"github.com/mendoncart/mantium/api/src/sources/mangahub"
	"github.com/mendoncart/mantium/api/src/util"
	"github.com/mendoncart/mantium/api/src/integrations/telegram"
)

func init() {
	// You can set the path to use an .env file below.
	// It can be an absolute path or relative to this file (main.go)
	filePath := ""
	if err := config.SetConfigs(filePath); err != nil {
		panic(err)
	}

	logLevelInt := config.GlobalConfigs.API.LogLevelInt
	logLevel, _ := zerolog.ParseLevel(strconv.Itoa(logLevelInt))
	log := util.GetLogger(logLevel)

	log.Info().Msg("Trying to connect to DB...")
	_db, err := db.OpenConn()
	if err != nil {
		panic(err)
	}
	defer _db.Close()

	log.Info().Msg("Creating tables and applying DB migrations...")
	err = db.CreateTables(_db, log)
	if err != nil {
		panic(err)
	}

	log.Info().Msg("Loading configs from DB...")
	err = config.LoadConfigsFromDB(config.GlobalConfigs.DashboardConfigs)
	if err != nil {
		if util.ErrorContains(err, sql.ErrNoRows.Error()) {
			err = config.SetDefaultConfigsInDB()
			if err != nil {
				panic(err)
			}
			err = config.LoadConfigsFromDB(config.GlobalConfigs.DashboardConfigs)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	log.Info().Msg("Updating manga URLs TLDs...")
	err = updateMangasTLDs()
	if err != nil {
		panic(err)
	}

	log.Info().Msg("Getting version from DB...")
	version, err := db.GetVersionFromDB(_db)
	if err != nil {
		panic(err)
	}
	log.Info().Msgf("Current version in DB: %s", version)
	config.GlobalConfigs.DashboardConfigs.Mantium.Version = version

	for _, m := range migrations {
		if m.Version == "update_version" {
			// This migration is always executed to update the version in the database
			err = m.Up(log)
			if err != nil {
				panic(err)
			}
			newVersion, err := db.GetVersionFromDB(_db)
			if err != nil {
				panic(err)
			}

			if newVersion != version {
				log.Info().Msg("Version updated in DB")
				log.Info().Msgf("New version in DB: %s", newVersion)
				config.GlobalConfigs.DashboardConfigs.Mantium.Version = newVersion
			}

			continue
		}
		if compareVersions(version, m.Version) < 0 {
			err = m.Up(log)
			if err != nil {
				panic(err)
			}
		}
	}

	setUpdateMangasMetadataPeriodicallyJob(log)
	dashboard.UpdateDashboard()

	if config.GlobalConfigs.Kaizoku.Valid {
		log.Info().Msg("Will use the Kaizoku integration")
	} else {
		log.Info().Msg("Will not use the Kaizoku integration")
	}
	if config.GlobalConfigs.Tranga.Valid {
		log.Info().Msg("Will use the Tranga integration")
	} else {
		log.Info().Msg("Will not use the Tranga integration")
	}

	// Initialize Telegram bot if polling is enabled
	if err := telegram.InitializeBotIfEnabled(); err != nil {
		log.Info().Msg("Failed to initialize Telegram bot")
	}
}

func main() {
	router := api.SetupRouter()
	router.SetTrustedProxies(nil)

	router.Run(":" + os.Getenv("API_PORT"))
}

func updateMangasTLDs() error {
	for k, v := range sources.SourcesTLDs {
		err := sources.ChangeSourceTLDInDB(k, v)
		if err != nil {
			return err
		}
	}

	return nil
}

// setUpdateMangasMetadataPeriodicallyJob sets a job to update mangas metadata periodically
// based on the configs set in the .env file in another goroutine.
func setUpdateMangasMetadataPeriodicallyJob(log *zerolog.Logger) {
	configs := config.GlobalConfigs.PeriodicallyUpdateMangas
	if configs.Update {
		log.Info().Msg("Starting to update mangas metadata periodically...")

		if configs.Notify {
			log.Info().Msg("Will notify when updating mangas metadata")
		} else {
			log.Info().Msg("Will not notify when updating mangas metadata")
		}

		log.Info().Msgf("Will update mangas metadata every %d minutes", configs.Minutes)
		log.Info().Msgf("First update in %d minutes", configs.Minutes)

		go func() {
			for {
				time.Sleep(time.Duration(configs.Minutes) * time.Minute)

				log.Info().Msg("Updating mangas metadata...")
				res, err := util.RequestUpdateMangasMetadata(configs.Notify)
				if err != nil {
					errMessage := fmt.Sprintf("Error updating mangas metadata in background: %s", err)
					log.Error().Msgf(errMessage)
					log.Error().Msgf("Request response: %v", res)

					if res != nil {
						var respMessage string
						body, err := io.ReadAll(res.Body)
						if err != nil {
							respMessage = fmt.Sprintf("Error while reading response body: %s", err)
						} else {
							respMessage = fmt.Sprintf("Request response text: %s", string(body))
						}
						log.Error().Msgf(respMessage)
						dashboard.SetLastBackgroundError(fmt.Sprintf("%s\n%s", errMessage, respMessage))
						res.Body.Close() // cannot be defer because it's an infinite loop
					} else {
						dashboard.SetLastBackgroundError(fmt.Sprintf("%s\n%s", errMessage, "No response to get the body"))
					}
				} else {
					log.Info().Msg("Mangas metadata updated")
				}
			}
		}()
	} else {
		log.Info().Msg("Not updating mangas metadata periodically")
	}
}

// Migration to be applied if current version stored in DB is lower than the field Version.
type Migration struct {
	Version string
	Up      func(*zerolog.Logger) error
}

// Change it in every new version
var (
	version        = "5.0.6"
	updatedMessage = `# Custom Manga Update

Custom mangas are now more similar to regular mangas. They still aren't part of a multimanga, but they now have more features.

### Last Released Chapter Selectors
The ability to set last released chapter name and URL selectors for custom mangas was added.

These CSS or XPATH selectors will be used to fetch the custom manga last released chapter name and URL from the custom manga page. In the background job, the custom mangas configured with these selectors will have their last released chapter updated automatically. Notifications will also be sent if enabled in the configs.

- More about it can be found [here](https://github.com/mendoncart/mantium/tree/main?tab=readme-ov-file#custom-manga).

### Next Chapter replaced with Last Read Chapter
The "Next Chapter" feature was removed and replaced with "Last Read Chapter". You can manually set the last read chapter and its URL. This will be used to track your reading progress.

### Custom Manga Forms Updated
![](https://github.com/user-attachments/assets/a057fa8a-8ebd-4b95-a648-388d366b7fbb)

# Other Changes

- **added**: this update message that will be shown in the dashboard after a notable update.
- **removed**: ComicK source, as it was shut down.
- **changed**: API routes for mangas. Check the docs if you use the API directly.
`
)

// If the current version in the database is lower than the Version field, the Up function will be executed.
var migrations = []Migration{
	{
		Version: "4.1.0",
		Up:      turnMangasIntoMultiMangas,
	},
	{
		Version: "4.1.0",
		Up:      updateMangas,
	},
	{
		Version: version,
		Up: func(_ *zerolog.Logger) error {
			if updatedMessage != "" {
				dashboard.UpdatedMessageToShow = updatedMessage
				dashboard.UpdatedMessageVersion = version
			}

			return nil
		},
	},
	{
		Version: "update_version", // It's not a valid version, so it will always be executed
		Up: func(*zerolog.Logger) error {
			const query = `UPDATE version SET version = $1`
			db, err := db.OpenConn()
			if err != nil {
				return util.AddErrorContext("error opening database connection", err)
			}
			defer db.Close()
			_, err = db.Exec(query, version)
			if err != nil {
				return util.AddErrorContext("error updating version in database", err)
			}

			return nil
		},
	},
}

func turnMangasIntoMultiMangas(log *zerolog.Logger) error {
	log.Info().Msg("Turning mangas into multimangas...")

	contextError := "error turning all mangas into multimangas"
	mangas, err := manga.GetMangasWithoutMultiMangasDB()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	for _, m := range mangas {
		_, err = manga.TurnIntoMultiManga(m)
		if err != nil {
			return util.AddErrorContext(contextError, err)
		}
	}

	return nil
}

func updateMangas(log *zerolog.Logger) error {
	log.Info().Msg("Truncating dates to second...")
	log.Info().Msg("Updating mangas sources...")
	log.Info().Msg("Updating mangas URL format...")

	multimangas, err := manga.GetMultiMangasDB(true)
	if err != nil {
		return err
	}

	for _, mm := range multimangas {
		for _, m := range mm.Mangas {
			// Truncate the dates to seconds
			contextError := fmt.Sprintf("error updating manga '%s'", m)
			if m.LastReleasedChapter != nil {
				m.LastReleasedChapter.UpdatedAt = m.LastReleasedChapter.UpdatedAt.Truncate(time.Second)
				err = m.UpsertChapterIntoDB(m.LastReleasedChapter)
				if err != nil {
					return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
				}
			}
			if m.LastReadChapter != nil {
				m.LastReadChapter.UpdatedAt = m.LastReadChapter.UpdatedAt.Truncate(time.Second)
				err = m.UpsertChapterIntoDB(m.LastReadChapter)
				if err != nil {
					return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
				}
			}

			// Update the source
			contextError = "error updating source for manga '%s'"
			source, err := sources.GetSource(m.URL)
			if err != nil {
				return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
			}

			err = m.UpdateSourceInDB(source.GetName())
			if err != nil {
				return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
			}

			// Update the URL to the new format
			contextError = "error updating URL for manga '%s'"
			var newURL string
			switch m.Source {
			case "mangadex":
				newURL, err = mangadex.GetFormattedMangaURL(m.URL)
				if err != nil {
					return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
				}
			case "mangahub":
				newURL, err = mangahub.GetFormattedMangaURL(m.URL)
				if err != nil {
					return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
				}
			}

			if newURL != "" {
				err = m.UpdateURLInDB(newURL)
				if err != nil {
					return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
				}
			}
		}
		contextError := "error updating multimanga '%s'"
		if mm.LastReadChapter != nil {
			mm.LastReadChapter.UpdatedAt = mm.LastReadChapter.UpdatedAt.Truncate(time.Second)
			err = mm.UpsertChapterIntoDB(mm.LastReadChapter)
			if err != nil {
				return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
			}
		}
	}

	contextError := "error updating custom mangas"
	customMangas, err := manga.GetCustomMangasDB()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	contextError = "error updating custom manga '%s'"
	for _, cm := range customMangas {
		if cm.LastReleasedChapter != nil {
			cm.LastReleasedChapter.UpdatedAt = cm.LastReleasedChapter.UpdatedAt.Truncate(time.Second)
			err = cm.UpsertChapterIntoDB(cm.LastReleasedChapter)
			if err != nil {
				return util.AddErrorContext(fmt.Sprintf(contextError, cm), err)
			}
		}
		if cm.LastReadChapter != nil {
			cm.LastReadChapter.UpdatedAt = cm.LastReadChapter.UpdatedAt.Truncate(time.Second)
			err = cm.UpsertChapterIntoDB(cm.LastReadChapter)
			if err != nil {
				return util.AddErrorContext(fmt.Sprintf(contextError, cm), err)
			}
		}
	}

	return nil
}

func compareVersions(v1, v2 string) int {
	splitV1 := strings.Split(v1, ".")
	splitV2 := strings.Split(v2, ".")

	maxLen := max(len(splitV1), len(splitV2))

	for i := range maxLen {
		var n1, n2 int

		if i < len(splitV1) {
			n1, _ = strconv.Atoi(splitV1[i])
		}
		if i < len(splitV2) {
			n2, _ = strconv.Atoi(splitV2[i])
		}

		if n1 < n2 {
			return -1 // v1 < v2
		} else if n1 > n2 {
			return 1 // v1 > v2
		}
	}

	return 0 // v1 == v2
}
