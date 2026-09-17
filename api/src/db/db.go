// Package db implements the database connection
package db

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/lib/pq" // postgres driver
	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src/util"
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 5
	connMaxIdleTime = 5 * time.Minute
)

type dbConfigs struct {
	Host     string
	Port     string
	DB       string
	User     string
	Password string
}

func getConfigs() *dbConfigs {
	return &dbConfigs{
		Host:     os.Getenv("POSTGRES_HOST"),
		Port:     os.Getenv("POSTGRES_PORT"),
		DB:       os.Getenv("POSTGRES_DB"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
	}
}

func (c *dbConfigs) connString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", c.Host, c.Port, c.User, c.Password, c.DB)
}

// String describes the target database without the password, so it is safe to
// put in an error message or a log line.
func (c *dbConfigs) String() string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s", c.Host, c.Port, c.User, c.DB)
}

var (
	instance atomic.Pointer[sql.DB]
	openMu   sync.Mutex
)

// OpenConn returns the database handle, opening it on first use.
//
// *sql.DB is a pool, not a connection, and it is safe for concurrent use, so
// there is exactly one for the whole process. Callers must not close it:
// every call site used to open its own pool and close it again, which meant a
// fresh TCP connection and a round trip per database operation.
func OpenConn() (*sql.DB, error) {
	if db := instance.Load(); db != nil {
		return db, nil
	}

	openMu.Lock()
	defer openMu.Unlock()

	if db := instance.Load(); db != nil {
		return db, nil
	}

	configs := getConfigs()

	db, err := sql.Open("postgres", configs.connString())
	if err != nil {
		return nil, util.AddErrorContext("error opening database connection", err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	// Not cached on failure, so a database that is still starting up can be
	// picked up by a later call instead of poisoning every one of them.
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, util.AddErrorContext(fmt.Sprintf("error pinging database %s", configs), err)
	}

	instance.Store(db)

	return db, nil
}

// CloseConn closes the shared pool. Only meant for process shutdown and tests.
func CloseConn() error {
	openMu.Lock()
	defer openMu.Unlock()

	db := instance.Swap(nil)
	if db == nil {
		return nil
	}

	return db.Close()
}

// CreateTables creates the tables in the database
func CreateTables(db *sql.DB, log *zerolog.Logger) error {
	log.Info().Msg("Creating tables if not exists...")
	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext("error starting transaction to create database", err)
	}

	_, err = tx.Exec(`
        CREATE TABLE IF NOT EXISTS "mangas" (
          "id" serial UNIQUE,
          "source" varchar(30) NOT NULL,
          "url" text NOT NULL PRIMARY KEY,
          "name" varchar(255) NOT NULL,
          "status" smallint NOT NULL,
          "internal_id" VARCHAR(100) NOT NULL DEFAULT '',
          "cover_img" bytea,
          "cover_img_resized" bool,
          "cover_img_url" text,
          "preferred_group" varchar(30),
          "last_released_chapter" integer,
          "last_read_chapter" integer,
		  "last_released_chapter_name_selector" text,
		  "last_released_chapter_name_attribute" varchar(30),
		  "last_released_chapter_name_regex" varchar(255),
		  "last_released_chapter_name_get_first" boolean NOT NULL DEFAULT FALSE,
		  "last_released_chapter_url_selector" text,
		  "last_released_chapter_url_attribute" varchar(30),
		  "last_released_chapter_url_get_first" boolean NOT NULL DEFAULT FALSE,
		  "last_released_chapter_selector_use_browser" boolean NOT NULL DEFAULT FALSE
        );

        CREATE INDEX IF NOT EXISTS "mangas_id_idx" ON "mangas" ("id");

        CREATE TABLE IF NOT EXISTS "multimangas" (
          "id" serial UNIQUE,
          "status" smallint NOT NULL,
          "current_manga" integer REFERENCES mangas(id),
          "last_read_chapter" integer,
          "cover_img" bytea NOT NULL DEFAULT '',
          "cover_img_resized" bool NOT NULL DEFAULT FALSE,
          "cover_img_url" text NOT NULL DEFAULT '',
          "cover_img_fixed" boolean NOT NULL DEFAULT FALSE
        );

        CREATE INDEX IF NOT EXISTS "multimangas_id_idx" ON "multimangas" ("id");

        CREATE TABLE IF NOT EXISTS "chapters" (
          "id" serial UNIQUE,
          "manga_id" integer,
          "multimanga_id" integer,
          "url" text,
          "chapter" varchar(255),
          "name" varchar(255),
          "internal_id" VARCHAR(100) NOT NULL DEFAULT '',
          "updated_at" timestamp,
          "type" smallint NOT NULL CONSTRAINT chapter_type_check CHECK (type IN (1, 2)),
          "from_source_site" bool NOT NULL DEFAULT TRUE,
          PRIMARY KEY ("url", "type")
        );

        CREATE INDEX IF NOT EXISTS "chapters_id_idx" ON "chapters" ("id");

		CREATE TABLE IF NOT EXISTS "configs" (
			"columns" integer NOT NULL DEFAULT 5,
			"show_background_error_warning" boolean NOT NULL DEFAULT TRUE,
			"search_results_limit" integer NOT NULL DEFAULT 20,
			"display_mode" varchar(50) NOT NULL DEFAULT 'Grid View' CHECK ("display_mode" IN ('Grid View', 'List View')),
			"add_all_multimanga_mangas_to_download_integrations" boolean NOT NULL DEFAULT FALSE,
			"enqueue_all_suwayomi_chapters_to_download" boolean NOT NULL DEFAULT TRUE
		);

		CREATE TABLE IF NOT EXISTS "version" (
			"version" VARCHAR(15) NOT NULL DEFAULT '4.0.4'
		);

		INSERT INTO version (version)
		SELECT '4.0.4'
		WHERE NOT EXISTS (SELECT 1 FROM version);
    `)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext("error creating tables in the database", err)
	}

	log.Info().Msg("Creating constraints if not exists...")
	_, err = tx.Exec(`
        do $$
       	begin
       		if not exists (
       			select 1
       			from pg_catalog.pg_constraint
       			where conname = 'mangas_last_released_chapter'
       		) then
       			ALTER TABLE "mangas" ADD CONSTRAINT mangas_last_released_chapter FOREIGN KEY ("last_released_chapter") REFERENCES "chapters" ("id");
       		end if;
       	end $$;

        do $$
       	begin
       		if not exists (
       			select 1
       			from pg_catalog.pg_constraint
       			where conname = 'mangas_last_read_chapter'
       		) then
                ALTER TABLE "mangas" ADD CONSTRAINT mangas_last_read_chapter FOREIGN KEY ("last_read_chapter") REFERENCES "chapters" ("id");
       		end if;
       	end $$;

        do $$
       	begin
       		if not exists (
       			select 1
       			from pg_catalog.pg_constraint
       			where conname = 'chapters_manga_id'
       		) then
                ALTER TABLE "chapters" ADD CONSTRAINT chapters_manga_id FOREIGN KEY ("manga_id") REFERENCES "mangas" ("id") ON DELETE CASCADE;
       		end if;
       	end $$;

        do $$
       	begin
       		if not exists (
       			select 1
       			from pg_catalog.pg_constraint
       			where conname = 'chapters_manga_id_type_unique'
       		) then
                ALTER TABLE "chapters" ADD CONSTRAINT chapters_manga_id_type_unique UNIQUE (manga_id, type);
       		end if;
       	end $$;

		do $$
		begin
			if not exists (
				select 1
				from pg_catalog.pg_constraint
				where conname = 'chapter_type_check'
			) then
				ALTER TABLE "chapters"
				ADD CONSTRAINT chapter_type_check
				CHECK (type IN (1, 2));
			end if;
		end
		$$;
    `)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext("error creating constraints in the database", err)
	}

	log.Info().Msg("Doing migrations...")
	_, err = tx.Exec(`
        DO $$
        BEGIN
            IF EXISTS (
                SELECT 1 
                FROM information_schema.columns 
                WHERE table_name='mangas' 
                  AND column_name='last_upload_chapter'
            ) AND NOT EXISTS (
                SELECT 1 
                FROM information_schema.columns 
                WHERE table_name='mangas' 
                  AND column_name='last_released_chapter'

            ) THEN
                ALTER TABLE mangas RENAME COLUMN last_upload_chapter TO last_released_chapter;
            END IF;
        END $$;

        ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "cover_img_fixed" BOOLEAN NOT NULL DEFAULT FALSE;
        ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "internal_id" VARCHAR(100) NOT NULL DEFAULT '';
        ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "multimanga_id" integer REFERENCES multimangas(id) ON DELETE CASCADE DEFAULT NULL;
        ALTER TABLE "mangas" ALTER COLUMN "last_released_chapter" TYPE integer;
        ALTER TABLE "mangas" ALTER COLUMN "last_read_chapter" TYPE integer;
        ALTER TABLE "mangas" ALTER COLUMN "url" TYPE text;
        ALTER TABLE "mangas" ALTER COLUMN "cover_img_url" TYPE text;
		ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "last_released_chapter_name_selector" text;
		ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "last_released_chapter_name_attribute" varchar(30);
		ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "last_released_chapter_name_regex" varchar(255);
		ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "last_released_chapter_name_get_first" boolean NOT NULL DEFAULT FALSE;
		ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "last_released_chapter_url_selector" text;
		ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "last_released_chapter_url_attribute" varchar(30);
		ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "last_released_chapter_url_get_first" boolean NOT NULL DEFAULT FALSE;
		ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "last_released_chapter_selector_use_browser" boolean NOT NULL DEFAULT FALSE;
        ALTER TABLE "chapters" ADD COLUMN IF NOT EXISTS "internal_id" VARCHAR(100) NOT NULL DEFAULT '';
        ALTER TABLE "chapters" ADD COLUMN IF NOT EXISTS "multimanga_id" integer DEFAULT NULL;
		ALTER TABLE "chapters" ADD COLUMN IF NOT EXISTS "from_source_site" boolean NOT NULL DEFAULT TRUE;
        ALTER TABLE "chapters" ALTER COLUMN "manga_id" DROP NOT NULL;
        ALTER TABLE "chapters" ALTER COLUMN "url" TYPE text;
        ALTER TABLE "multimangas" ALTER COLUMN "cover_img_url" TYPE text;

        -- Postgres does not index the referencing side of a foreign key, so
        -- every join on multimanga_id and every cascading delete of a
        -- multimanga had to scan the whole mangas table.
        CREATE INDEX IF NOT EXISTS "mangas_multimanga_id_idx" ON "mangas" ("multimanga_id");
        CREATE INDEX IF NOT EXISTS "mangas_source_idx" ON "mangas" ("source");

        do $$
       	begin
       		if not exists (
       			select 1
       			from pg_catalog.pg_constraint
       			where conname = 'chapters_multimanga_id_type_unique'
       		) then
                ALTER TABLE "chapters" ADD CONSTRAINT chapters_multimanga_id_type_unique UNIQUE (multimanga_id, type);
       		end if;
       	end $$;
        do $$
       	begin
       		if not exists (
       			select 1
       			from pg_catalog.pg_constraint
       			where conname = 'chapters_multimanga_id'
       		) then
                ALTER TABLE "chapters" ADD CONSTRAINT chapters_multimanga_id FOREIGN KEY ("multimanga_id") REFERENCES "multimangas" ("id") ON DELETE CASCADE;
       		end if;
       	end $$;
    `)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext("error applying migrations in the database", err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext("error committing transaction to create tables in the database", err)
	}

	log.Info().Msg("Database tables created")

	return nil
}

func GetVersionFromDB(db *sql.DB) (string, error) {
	const query = `SELECT version FROM version`
	var version string
	err := db.QueryRow(query).Scan(&version)
	if err != nil {
		return "", err
	}

	return version, nil
}
