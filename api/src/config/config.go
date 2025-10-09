// Package config implements the configurations for the application.
package config

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src/util"
)

// GlobalConfigs is a pointer to the Configs struct that holds all the configurations.
// It is used to access the configurations throughout the application.
// Should be initialized by the SetConfigs function.
var GlobalConfigs = &Configs{
	API:                      &APIConfigs{},
	DashboardConfigs:         &DashboardConfigs{},
	Ntfy:                     &NtfyConfigs{},
	PeriodicallyUpdateMangas: &PeriodicallyUpdateMangasConfigs{},
	Kaizoku:                  &KaizokuConfigs{},
	Tranga:                   &TrangaConfigs{},
	Suwayomi:                 &SuwayomiConfigs{},
}

// Configs is a struct that holds all the configurations.
type Configs struct {
	API                      *APIConfigs
	DashboardConfigs         *DashboardConfigs
	Ntfy                     *NtfyConfigs
	PeriodicallyUpdateMangas *PeriodicallyUpdateMangasConfigs
	Kaizoku                  *KaizokuConfigs
	Tranga                   *TrangaConfigs
	Suwayomi                 *SuwayomiConfigs
}

// APIConfigs is a struct that holds the API configurations.
type APIConfigs struct {
	Port        string
	LogLevelInt int
}

// NtfyConfigs is a struct that holds the ntfy configurations.
type NtfyConfigs struct {
	Address string
	Topic   string
	Token   string
}

// PeriodicallyUpdateMangasConfigs is a struct that holds the configurations for updating mangas metadata periodically.
type PeriodicallyUpdateMangasConfigs struct {
	Update       bool
	Notify       bool
	Minutes      int
	ParallelJobs int
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
	Address  string
	Username string
	Password string
	Valid    bool
}

// DashboardConfigs is a struct that holds the configurations for the dashboard.
// This will be set mostly by the dashboard configs form.
type DashboardConfigs struct {
	Display struct {
		Columns                    int    `json:"columns"`
		ShowBackgroundErrorWarning bool   `json:"showBackgroundErrorWarning"`
		SearchResultsLimit         int    `json:"searchResultsLimit"`
		DisplayMode                string `json:"displayMode"`
	} `json:"display"`
	Integrations struct {
		AddAllMultiMangaMangasToDownloadIntegrations bool `json:"addAllMultiMangaMangasToDownloadIntegrations"`
		EnqueueAllSuwayomiChaptersToDownload         bool `json:"enqueueAllSuwayomiChaptersToDownload"`
	} `json:"integrations"`
	Manga struct {
		AllowedSources       []string `json:"allowedSources"`
		AllowedAddingMethods []string `json:"allowedAddingMethods"`
	} `json:"manga"`
	Mantium struct {
		Version string `json:"version"`
	}
}

var (
	ValidDisplayModeValues = []string{"Grid View", "List View"}
	ValidAddingMethods     = []string{"Search", "URL"}
	SourcesList            = []string{
		"mangadex",
		"mangahub",
		"mangaplus",
		"mangaupdates",
		"rawkuma",
		"klmanga",
		"jmanga",
	}
)

var oldConfigsFilePath = "./configs/configs.json"

// SetConfigs sets the configurations based on a .env file if provided or using environment variables.
func SetConfigs(filePath string) error {
	var err error

	if util.FileExists(oldConfigsFilePath) {
		return fmt.Errorf("old configs file '%s' found. Settings are stored in the database in Mantium version 4.0. Please remove it and set the configs in the dashboard again", oldConfigsFilePath)
	}

	if filePath != "" {
		err = godotenv.Load(filePath)
		if err != nil {
			return fmt.Errorf("error loading env file '%s': %s", filePath, err)
		}
	}

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
	GlobalConfigs.Suwayomi.Username = os.Getenv("SUWAYOMI_USERNAME")
	GlobalConfigs.Suwayomi.Password = os.Getenv("SUWAYOMI_PASSWORD")

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
	if envUpdateMangasJobGoRoutines := os.Getenv("UPDATE_MANGAS_JOB_PARALLEL_JOBS"); envUpdateMangasJobGoRoutines != "" {
		updateMangasJobGoRoutines, err = strconv.Atoi(envUpdateMangasJobGoRoutines)
		if err != nil {
			return fmt.Errorf("error converting UPDATE_MANGAS_JOB_PARALLEL_JOBS '%s' to int: %s", envUpdateMangasJobGoRoutines, err)
		}
	}
	GlobalConfigs.PeriodicallyUpdateMangas.ParallelJobs = updateMangasJobGoRoutines

	GlobalConfigs.DashboardConfigs.Manga.AllowedSources = SourcesList
	envAllowedSources := os.Getenv("ALLOWED_SOURCES")
	if envAllowedSources != "" {
		GlobalConfigs.DashboardConfigs.Manga.AllowedSources = strings.Split(envAllowedSources, ",")
		for _, source := range GlobalConfigs.DashboardConfigs.Manga.AllowedSources {
			if !slices.Contains(SourcesList, source) {
				return fmt.Errorf("error parsing ALLOWED_SOURCES '%s': source '%s' not found in available sources: %s", envAllowedSources, source, SourcesList)
			}
		}
	}

	GlobalConfigs.DashboardConfigs.Manga.AllowedAddingMethods = ValidAddingMethods
	envAllowedAddingMethods := os.Getenv("ALLOWED_ADDING_METHODS")
	if envAllowedAddingMethods != "" {
		GlobalConfigs.DashboardConfigs.Manga.AllowedAddingMethods = strings.Split(envAllowedAddingMethods, ",")
		for _, method := range GlobalConfigs.DashboardConfigs.Manga.AllowedAddingMethods {
			if !slices.Contains(ValidAddingMethods, method) {
				return fmt.Errorf("error parsing ALLOWED_ADDING_METHODS '%s': method '%s' not found in available methods: %s", envAllowedAddingMethods, method, ValidAddingMethods)
			}
		}
	}

	return nil
}
