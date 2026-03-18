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

	"github.com/diogovalentte/mantium/api/src"
	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/dashboard"
	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources"
	"github.com/diogovalentte/mantium/api/src/sources/mangadex"
	"github.com/diogovalentte/mantium/api/src/sources/mangahub"
	"github.com/diogovalentte/mantium/api/src/util"
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
					log.Error().Msg(errMessage)
					log.Error().Msgf("Request response: %v", res)

					if res != nil {
						var respMessage string
						body, err := io.ReadAll(res.Body)
						if err != nil {
							respMessage = fmt.Sprintf("Error while reading response body: %s", err)
						} else {
							respMessage = fmt.Sprintf("Request response text: %s", string(body))
						}
						log.Error().Msg(respMessage)
						dashboard.SetLastBackgroundError(fmt.Sprintf("%s\n%s", errMessage, respMessage))
						res.Body.Close() // cannot be defer because it's an infinite loop
					} else {
						dashboard.SetLastBackgroundError(fmt.Sprintf("%s\n%s", errMessage, "No response to get the body"))
					}
				} else {
					log.Info().Msg("Mangas metadata updated")
					dashboard.ResetConsecutiveErrors()
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
	version        = "6.0.1"
	updatedMessage = `
# Custom Manga in MultiManga
Previously, custom manga were different from normal manga because they were not integrated into multimanga. Now, all previous custom manga are turned into multimanga, and new custom manga added to Mantium automatically are turned into multimanga.

You can also add custom manga to multimanga and be treated like normal manga tracked by native sources:

|                                 MultiManga Edit Form                                 |                                  Edit Custom Manga form                                   |
| :----------------------------------------------------------------------------------: | :----------------------------------------------------------------------------------: |
| ![](https://github.com/user-attachments/assets/254919b4-1ad6-4dc9-a838-c6da8be73daa) | ![](https://github.com/user-attachments/assets/43edab08-9610-4bd1-a447-2e56cf57cd31) |

You can edit the custom manga properties with the multimanga "**Manage Mangas**" button:

![](https://github.com/user-attachments/assets/e868c0c1-83c0-4249-b618-2039394b8ab6)


# Custom Last Read Chapter
Previously, you could only set the last read chapter for normal manga by selecting one from the current manga chapters list, or manually setting it with custom manga.

Now that normal manga and custom manga are integrated in multimanga, you can select the chapter from the multimanga's current manga (_or the next normal manga candidate if the current manga is a custom manga_), manually provide it, or just delete the last read chapter:


![](https://github.com/user-attachments/assets/c990d54b-0a13-4b7e-bf0e-8519fb4882a4)

# Consecutive Background Errors
Added the UPDATE_MANGAS_PERIODICALLY_NUMBER_OF_CONSECUTIVE_ERRORS_TO_SHOW environment variable. Previously, every time an error happened a warning would show in the dashboard and iFrame (_if enabled_). Now, it'll only show when an error occurs $x$ consecutive times based on this new env var.

When the background job finishes without errors, it resets the counter.

# API Breaking Changes
Due to the changes mentioned above, the API was changed. Some methods were deleted, changed, and added.

# Other changes:

- Fixed: RawKuma nil pointer dereference error.
- Improved: dashboard CSS for manga header, forms with a red button.
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
		Version: "6.0.0",
		Up:      turnCustomMangasIntoMultiMangas,
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
	mangas, err := manga.GetMangasWithoutMultiMangasDB(false)
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

func turnCustomMangasIntoMultiMangas(log *zerolog.Logger) error {
	log.Info().Msg("Turning custom mangas into multimangas...")

	contextError := "error turning all custom mangas into multimangas"
	mangas, err := manga.GetMangasWithoutMultiMangasDB(true)
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
