// Package main implements the init and main function
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src"
	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/dashboard"
	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources"
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

	err := dashboard.SetDefaultConfigsFile()
	if err != nil {
		panic(err)
	}

	log.Info().Msg("Trying to connect to DB...")
	_db, err := db.OpenConn()
	if err != nil {
		panic(err)
	}
	defer _db.Close()

	err = db.CreateTables(_db, log)
	if err != nil {
		panic(err)
	}

	log.Info().Msg("Turning mangas into multimangas...")
	err = turnMangasIntoMultiMangas()
	if err != nil {
		panic(err)
	}

	log.Info().Msg("Truncating dates to second...")
	err = truncateDatesToSecond()
	if err != nil {
		panic(err)
	}

	log.Info().Msg("Updating mangas sources...")
	err = updateMangasSources()
	if err != nil {
		panic(err)
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

func turnMangasIntoMultiMangas() error {
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

func truncateDatesToSecond() error {
	multimangas, err := manga.GetMultiMangasDB(true)
	if err != nil {
		return err
	}

	for _, mm := range multimangas {
		for _, m := range mm.Mangas {
			if m.LastReleasedChapter != nil {
				m.LastReleasedChapter.UpdatedAt = m.LastReleasedChapter.UpdatedAt.Truncate(time.Second)
				err = m.UpsertChapterIntoDB(m.LastReleasedChapter)
				if err != nil {
					return err
				}
			}
			if m.LastReadChapter != nil {
				m.LastReadChapter.UpdatedAt = m.LastReadChapter.UpdatedAt.Truncate(time.Second)
				err = m.UpsertChapterIntoDB(m.LastReadChapter)
				if err != nil {
					return err
				}
			}
		}
		if mm.LastReadChapter != nil {
			mm.LastReadChapter.UpdatedAt = mm.LastReadChapter.UpdatedAt.Truncate(time.Second)
			err = mm.UpsertChapterIntoDB(mm.LastReadChapter)
			if err != nil {
				return err
			}
		}
	}

	customMangas, err := manga.GetCustomMangasDB()
	if err != nil {
		return err
	}
	for _, cm := range customMangas {
		if cm.LastReleasedChapter != nil {
			cm.LastReleasedChapter.UpdatedAt = cm.LastReleasedChapter.UpdatedAt.Truncate(time.Second)
			err = cm.UpsertChapterIntoDB(cm.LastReleasedChapter)
			if err != nil {
				return err
			}
		}
		if cm.LastReadChapter != nil {
			cm.LastReadChapter.UpdatedAt = cm.LastReadChapter.UpdatedAt.Truncate(time.Second)
			err = cm.UpsertChapterIntoDB(cm.LastReadChapter)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func updateMangasSources() error {
	contextError := "error updating mangas sources"
	multimangas, err := manga.GetMultiMangasDB(true)
	if err != nil {
		return err
	}

	for _, mm := range multimangas {
		for _, m := range mm.Mangas {
			source, err := sources.GetSource(m.URL)
			if err != nil {
				return util.AddErrorContext(contextError, err)
			}

			err = m.UpdateSourceInDB(source.GetName())
			if err != nil {
				return util.AddErrorContext(contextError, err)
			}
		}
	}

	return nil
}
