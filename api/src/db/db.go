// Package db implements the database connection
package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // postgres driver
	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src/config"
)

func getConnString() string {
	configs := config.GlobalConfigs.DB

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", configs.Host, configs.Port, configs.User, configs.Password, configs.DB)
}

// OpenConn opens a connection to the database
func OpenConn() (*sql.DB, error) {
	db, err := sql.Open("postgres", getConnString())
	if err != nil {
		return nil, err
	}

	err = db.Ping()

	return db, err
}

// CreateTables creates the tables in the database
func CreateTables(db *sql.DB, log *zerolog.Logger) error {
	log.Info().Msg("Creating tables if not exists...")
	tx, err := db.Begin()
	if err != nil {
		return err
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
          "last_upload_chapter" smallint,
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
    `)
	if err != nil {
		tx.Rollback()
		return err
	}

	log.Info().Msg("Creating constraints if not exists...")
	_, err = tx.Exec(`
        do $$
       	begin
       		if not exists (
       			select 1
       			from pg_catalog.pg_constraint
       			where conname = 'mangas_last_upload_chapter'
       		) then
       			ALTER TABLE "mangas" ADD CONSTRAINT mangas_last_upload_chapter FOREIGN KEY ("last_upload_chapter") REFERENCES "chapters" ("id");
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
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	log.Info().Msg("Database tables created")

	return nil
}
