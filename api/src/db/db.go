// Package db implements the database connection
package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq" // postgres driver
	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src/util"
)

type dbConfigs struct {
	Host     string
	Port     string
	DB       string
	User     string
	Password string
}

func getConnString() string {
	configs := &dbConfigs{
		Host:     os.Getenv("POSTGRES_HOST"),
		Port:     os.Getenv("POSTGRES_PORT"),
		DB:       os.Getenv("POSTGRES_DB"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", configs.Host, configs.Port, configs.User, configs.Password, configs.DB)
}

// OpenConn opens a connection to the database
func OpenConn() (*sql.DB, error) {
	db, err := sql.Open("postgres", getConnString())
	if err != nil {
		return nil, util.AddErrorContext("error opening database connection", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf("error pinging database %s", getConnString()), err)
	}

	return db, nil
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
          "type" smallint,
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
        ALTER TABLE "chapters" ALTER COLUMN "manga_id" DROP NOT NULL;
        ALTER TABLE "chapters" ALTER COLUMN "url" TYPE text;
        ALTER TABLE "multimangas" ALTER COLUMN "cover_img_url" TYPE text;

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
