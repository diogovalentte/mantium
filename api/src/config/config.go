// Package config implements the configurations for the application.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

// GlobalConfigs is a pointer to the Configs struct that holds all the configurations.
// It is used to access the configurations throughout the application.
// Should be initialized by the SetConfigs function.
var GlobalConfigs = &Configs{
	API:                       &APIConfigs{},
	DB:                        &DBConfigs{},
	Ntfy:                      &NtfyConfigs{},
	PeriodicallyUpdateMangas:  &PeriodicallyUpdateMangasConfigs{},
	Kaizoku:                   &KaizokuConfigs{},
	Tranga:                    &TrangaConfigs{},
	Suwayomi:                  &SuwayomiConfigs{},
	ConfigsFilePath:           "./configs/configs.json",
	DefaultConfigsFilePath:    "../defaults/configs.json",
	UpdateMangasJobGoRoutines: 1, // default value
}

// Configs is a struct that holds all the configurations.
type Configs struct {
	API                      *APIConfigs
	DB                       *DBConfigs
	Ntfy                     *NtfyConfigs
	PeriodicallyUpdateMangas *PeriodicallyUpdateMangasConfigs
	Kaizoku                  *KaizokuConfigs
	Tranga                   *TrangaConfigs
	Suwayomi                 *SuwayomiConfigs
	// A file with configs that should be persisted
	// Relative to main.go
	ConfigsFilePath           string
	DefaultConfigsFilePath    string
	UpdateMangasJobGoRoutines int
}

// APIConfigs is a struct that holds the API configurations.
type APIConfigs struct {
	Port        string
	LogLevelInt int
}

// DBConfigs is a struct that holds the database configurations.
type DBConfigs struct {
	Host     string
	Port     string
	DB       string
	User     string
	Password string
}

// NtfyConfigs is a struct that holds the ntfy configurations.
type NtfyConfigs struct {
	Address string
	Topic   string
	Token   string
}

// PeriodicallyUpdateMangasConfigs is a struct that holds the configurations for updating mangas metadata periodically.
type PeriodicallyUpdateMangasConfigs struct {
	Update  bool
	Notify  bool
	Minutes int
}

// KaizokuConfigs is a struct that holds the configurations for the Kaizoku integration.
type KaizokuConfigs struct {
	Address                     string
	DefaultInterval             string
	WaitUntilEmptyQueuesTimeout time.Duration
	TryOtherSources             bool
	Valid                       bool
}

// TrangaConfigs is a struct that holds the configurations for the Tranga integration.
type TrangaConfigs struct {
	Address         string
	DefaultInterval string
	Valid           bool
}

// SuwayomiConfigs is a struct that holds the configurations for the Suwayomi integration.
type SuwayomiConfigs struct {
	Address string
	Valid   bool
}

// SetConfigs sets the configurations based on a .env file if provided or using environment variables.
func SetConfigs(filePath string) error {
	if filePath != "" {
		err := godotenv.Load(filePath)
		if err != nil {
			return fmt.Errorf("error loading env file '%s': %s", filePath, err)
		}
	}

	var err error

	logLevel := zerolog.InfoLevel
	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr != "" {
		logLevel, err = zerolog.ParseLevel(logLevelStr)
		if err != nil {
			return fmt.Errorf("error parsing error level '%s': %s", logLevelStr, err)
		}
	}
	GlobalConfigs.API.LogLevelInt = int(logLevel)

	GlobalConfigs.API.Port = os.Getenv("API_PORT")

	GlobalConfigs.DB.Host = os.Getenv("POSTGRES_HOST")
	GlobalConfigs.DB.Port = os.Getenv("POSTGRES_PORT")
	GlobalConfigs.DB.DB = os.Getenv("POSTGRES_DB")
	GlobalConfigs.DB.User = os.Getenv("POSTGRES_USER")
	GlobalConfigs.DB.Password = os.Getenv("POSTGRES_PASSWORD")

	GlobalConfigs.Ntfy.Address = os.Getenv("NTFY_ADDRESS")
	GlobalConfigs.Ntfy.Topic = os.Getenv("NTFY_TOPIC")
	GlobalConfigs.Ntfy.Token = os.Getenv("NTFY_TOKEN")

	GlobalConfigs.Kaizoku.Address = os.Getenv("KAIZOKU_ADDRESS")
	GlobalConfigs.Kaizoku.DefaultInterval = os.Getenv("KAIZOKU_DEFAULT_INTERVAL")
	if GlobalConfigs.Kaizoku.DefaultInterval != "" && GlobalConfigs.Kaizoku.Address != "" {
		waitUntilEmptyQueuesTimeoutStr := os.Getenv("KAIZOKU_WAIT_UNTIL_EMPTY_QUEUES_TIMEOUT_MINUTES")
		if waitUntilEmptyQueuesTimeoutStr != "" {
			waitUntilEmptyQueuesTimeout, err := strconv.Atoi(waitUntilEmptyQueuesTimeoutStr)
			if err != nil {
				return fmt.Errorf("error converting KAIZOKU_WAIT_UNTIL_EMPTY_QUEUES_TIMEOUT_MINUTES '%s' to int: %s", waitUntilEmptyQueuesTimeoutStr, err)
			}
			GlobalConfigs.Kaizoku.WaitUntilEmptyQueuesTimeout = time.Duration(waitUntilEmptyQueuesTimeout) * time.Minute
		} else {
			GlobalConfigs.Kaizoku.WaitUntilEmptyQueuesTimeout = 5 * time.Minute
		}

		tryOtherSources := os.Getenv("KAIZOKU_TRY_OTHER_SOURCES")
		if tryOtherSources != "" {
			switch tryOtherSources {
			case "true":
				GlobalConfigs.Kaizoku.TryOtherSources = true
			case "false":
			default:
				return fmt.Errorf("error parsing KAIZOKU_TRY_OTHER_SOURCES '%s': must be 'true' or 'false'", tryOtherSources)
			}
		}

		GlobalConfigs.Kaizoku.Valid = true
	}

	GlobalConfigs.Tranga.Address = os.Getenv("TRANGA_ADDRESS")
	GlobalConfigs.Tranga.DefaultInterval = os.Getenv("TRANGA_DEFAULT_INTERVAL")
	if GlobalConfigs.Tranga.DefaultInterval == "" {
		GlobalConfigs.Tranga.DefaultInterval = "03:00:00"
	}
	if GlobalConfigs.Tranga.Address != "" {
		GlobalConfigs.Tranga.Valid = true
	}

	GlobalConfigs.Suwayomi.Address = os.Getenv("SUWAYOMI_ADDRESS")
	if GlobalConfigs.Suwayomi.Address != "" {
		GlobalConfigs.Suwayomi.Valid = true
	}

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
			return fmt.Errorf("error converting UPDATE_MANGAS_PERIODICALLY_MINUTES '%s' to int: %s", envMinutes, err)
		}
	}
	GlobalConfigs.PeriodicallyUpdateMangas.Minutes = minutes

	updateMangasJobGoRoutines := 1
	envUpdateMangasJobGoRoutines := os.Getenv("UPDATE_MANGAS_JOB_PARALLEL_JOBS")
	if envUpdateMangasJobGoRoutines != "" {
		updateMangasJobGoRoutines, err = strconv.Atoi(envUpdateMangasJobGoRoutines)
		if err != nil {
			return fmt.Errorf("error converting UPDATE_MANGAS_JOB_PARALLEL_JOBS '%s' to int: %s", envUpdateMangasJobGoRoutines, err)
		}
	}
	GlobalConfigs.UpdateMangasJobGoRoutines = updateMangasJobGoRoutines

	return nil
}
