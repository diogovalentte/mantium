package config

import (
	"sync"

	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/diogovalentte/mantium/api/src/util"
)

func LoadConfigsFromDB(configs *DashboardConfigs) error {
	contextError := "error loading configs from DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	defer db.Close()

	err = db.QueryRow(`
		SELECT
			columns, show_background_error_warning, search_results_limit, display_mode,
			add_all_multimanga_mangas_to_download_integrations, enqueue_all_suwayomi_chapters_to_download
		FROM
			configs;
	`).Scan(
		&configs.Display.Columns, &configs.Display.ShowBackgroundErrorWarning,
		&configs.Display.SearchResultsLimit, &configs.Display.DisplayMode,
		&configs.Integrations.AddAllMultiMangaMangasToDownloadIntegrations,
		&configs.Integrations.EnqueueAllSuwayomiChaptersToDownload,
	)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	return nil
}

func SetDefaultConfigsInDB() error {
	contextError := "error setting default configs in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	_, err = tx.Exec(`
		INSERT INTO
			configs
		DEFAULT VALUES
		;
	`)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(contextError, err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	return nil
}

var updateDBConfigsMutex = &sync.Mutex{}

func SaveConfigsToDB(configs *DashboardConfigs) error {
	contextError := "error saving configs to DB"

	updateDBConfigsMutex.Lock()
	defer updateDBConfigsMutex.Unlock()

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	_, err = tx.Exec(`
		UPDATE
			configs
		SET
			columns = $1, show_background_error_warning = $2, search_results_limit = $3, display_mode = $4,
			add_all_multimanga_mangas_to_download_integrations = $5, enqueue_all_suwayomi_chapters_to_download = $6
		;
	`, configs.Display.Columns, configs.Display.ShowBackgroundErrorWarning,
		configs.Display.SearchResultsLimit, configs.Display.DisplayMode,
		configs.Integrations.AddAllMultiMangaMangasToDownloadIntegrations,
		configs.Integrations.EnqueueAllSuwayomiChaptersToDownload,
	)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(contextError, err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	return nil
}
