package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

var GlobalConfigs *Configs = &Configs{
	DB:                       &DBConfigs{},
	Ntfy:                     &NtfyConfigs{},
	PeriodicallyUpdateMangas: &PeriodicallyUpdateMangasConfigs{},
}

type Configs struct {
	LogLevel                 zerolog.Level
	DB                       *DBConfigs
	Ntfy                     *NtfyConfigs
	PeriodicallyUpdateMangas *PeriodicallyUpdateMangasConfigs
}

type DBConfigs struct {
	Host     string
	Port     string
	DB       string
	User     string
	Password string
}

type NtfyConfigs struct {
	Address string
	Topic   string
	Token   string
}

type PeriodicallyUpdateMangasConfigs struct {
	Update  bool
	Notify  bool
	Minutes int
}

func SetConfigs(filePath string) error {
	if filePath != "" {
		err := godotenv.Load(filePath)
		if err != nil {
			return err
		}
	}

	var err error

	logLevel := zerolog.InfoLevel
	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr != "" {
		logLevel, err = zerolog.ParseLevel(logLevelStr)
		if err != nil {
			return err
		}
	}
	GlobalConfigs.LogLevel = logLevel

	GlobalConfigs.DB.Host = os.Getenv("POSTGRES_HOST")
	GlobalConfigs.DB.Port = os.Getenv("POSTGRES_PORT")
	GlobalConfigs.DB.DB = os.Getenv("POSTGRES_DB")
	GlobalConfigs.DB.User = os.Getenv("POSTGRES_USER")
	GlobalConfigs.DB.Password = os.Getenv("POSTGRES_PASSWORD")

	GlobalConfigs.Ntfy.Address = os.Getenv("NTFY_ADDRESS")
	GlobalConfigs.Ntfy.Topic = os.Getenv("NTFY_TOPIC")
	GlobalConfigs.Ntfy.Token = os.Getenv("NTFY_TOKEN")

	if os.Getenv("UPDATE_MANGAS_PERIODICALLY") == "true" {
		GlobalConfigs.PeriodicallyUpdateMangas.Update = true
	}
	if os.Getenv("UPDATE_MANGAS_PERIODICALLY_NOTIFY") == "true" {
		GlobalConfigs.PeriodicallyUpdateMangas.Notify = true
	}
	minutes := 30
	envMinutes := os.Getenv("UPDATE_MANGAS_PERIODICALLY_MINUTES")
	if envMinutes != "" {
		minutes, err = strconv.Atoi(envMinutes)
		if err != nil {
			return fmt.Errorf("Error converting UPDATE_MANGAS_PERIODICALLY_MINUTES to int: %s", err)
		}
	}
	GlobalConfigs.PeriodicallyUpdateMangas.Minutes = minutes

	return nil
}
