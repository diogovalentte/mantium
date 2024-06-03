// Package db implements the database connection
package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // postgres driver
	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/util"
)

func getConnString() string {
	configs := config.GlobalConfigs.DB

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
          "url" varchar(255) NOT NULL PRIMARY KEY,
          "name" varchar(255) NOT NULL,
          "status" smallint NOT NULL,
          "cover_img" bytea,
          "cover_img_resized" bool,
          "cover_img_url" varchar(255),
          "preferred_group" varchar(30),
          "last_released_chapter" smallint,
          "last_read_chapter" smallint
        );

        CREATE TABLE IF NOT EXISTS "chapters" (
          "id" serial UNIQUE,
          "manga_id" integer NOT NULL,
          "url" varchar(255),
          "chapter" varchar(255),
          "name" varchar(255),
          "updated_at" timestamp,
          "type" smallint,
          PRIMARY KEY ("url", "type")
        );

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

	log.Info().Msg("Doing migrations if not exists...")
	_, err = tx.Exec(`
        ALTER TABLE "mangas" ADD COLUMN IF NOT EXISTS "cover_img_fixed" BOOLEAN NOT NULL DEFAULT FALSE;
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
