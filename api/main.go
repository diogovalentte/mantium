// Package main implements the init and main function
package main

import (
	"time"

    // For testing purposes
	// "github.com/joho/godotenv"

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

	log.Info().Msg("Waiting 10 seconds for database finish set up...")
	time.Sleep(10 * time.Second)

	_db, err := db.OpenConn()
	if err != nil {
		panic(err)
	}
	defer _db.Close()

	err = db.CreateTables(_db, log)
	if err != nil {
		panic(err)
	}
}

func main() {
	router := api.SetupRouter()

	router.Run()
}
