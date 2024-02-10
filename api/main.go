// Package main implements the init and main function
package main

import (
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
}

func main() {
	router := api.SetupRouter()

	router.Run()
}
