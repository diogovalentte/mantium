// Package main implements the init and main function
package main

import (
	"io"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/diogovalentte/manga-dashboard-api/api/src"
	"github.com/diogovalentte/manga-dashboard-api/api/src/db"
	"github.com/diogovalentte/manga-dashboard-api/api/src/util"
)

func init() {
	// For testing purposes
	// err := godotenv.Load("../.env.test")
	// if err != nil {
	// 	panic(err)
	// }

	log := util.GetLogger()

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

	setUpdateMangasMetadataPeriodicallyJob(log)
}

func main() {
	router := api.SetupRouter()
	router.SetTrustedProxies(nil)

	router.Run()
}

func setUpdateMangasMetadataPeriodicallyJob(log *zerolog.Logger) {
	if os.Getenv("UPDATE_MANGAS_PERIODICALLY") == "true" {
		log.Info().Msg("Starting to update mangas metadata periodically...")
		var notify bool

		if os.Getenv("UPDATE_MANGAS_PERIODICALLY_NOTIFY") == "true" {
			notify = true
			log.Info().Msg("Will notify when updating mangas metadata")
		} else {
			log.Info().Msg("Will not notify when updating mangas metadata")
		}

		minutes := 30
		envMinutes := os.Getenv("UPDATE_MANGAS_PERIODICALLY_MINUTES")
		if envMinutes != "" {
			var err error
			minutes, err = strconv.Atoi(envMinutes)
			if err != nil {
				log.Error().Msgf("Error converting UPDATE_MANGAS_PERIODICALLY_MINUTES to int: %s", err)
				os.Exit(1)
			}
		}
		log.Info().Msgf("Will update mangas metadata every %d minutes", minutes)
		log.Info().Msgf("First update in %d minutes", minutes)

		go func() {
			for {
				time.Sleep(time.Duration(minutes) * time.Minute)

				log.Info().Msg("Updating mangas metadata...")
				res, err := util.RequestUpdateMangasMetadata(notify)
				if err != nil {
					log.Error().Msgf("Error updating mangas metadata: %s", err)
					log.Error().Msgf("Request response: %s", res)
					body, err := io.ReadAll(res.Body)
					if err != nil {
						log.Error().Msgf("Error while getting the response body: %s", err)
					}
					log.Error().Msgf("Request response text: %s", string(body))
				} else {
					log.Info().Msg("Mangas metadata updated")
				}
			}
		}()
	} else {
		log.Info().Msg("Not updating mangas metadata periodically")
	}
}
