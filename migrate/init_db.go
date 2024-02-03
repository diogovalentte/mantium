// Package main implements the database migration script
package main

import (
	"database/sql"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"

	"github.com/diogovalentte/manga-dashboard-api/src/db"
	"github.com/diogovalentte/manga-dashboard-api/src/util"
)

func createTables(db *sql.DB, log *zerolog.Logger) error {
	log.Info().Msg("Creating tables...")
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
          "number" float8,
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

	log.Info().Msg("Creating constraints...")
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

	return nil
}

func init() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
}

func main() {
	log := util.GetLogger()

	db, err := db.OpenConn()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = createTables(db, log)
	if err != nil {
		panic(err)
	}
	log.Info().Msg("Tables created")
}
