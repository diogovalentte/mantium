// Package main implements the init and main function
package main

import (
	"github.com/joho/godotenv"

	"github.com/diogovalentte/manga-dashboard-api/src"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
}

func main() {
	router := api.SetupRouter()

	router.Run()
}
