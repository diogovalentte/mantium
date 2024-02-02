# Mangas Dashboard API
This is the API used by the [Manga Dashboard](https://github.com/diogovalentte/manga-dashboard) project.

# Steps to run:
1. Create the Docker services, this will start the PostgreSQL database of the API:
```sh
docker compose up -d
```
2. Run the migrations to create the database tables:
```sh
go run migrate/init_db.go
```
3. Run the API:
```sh
go run main.go
```

# Simple doc
This project's objective is to manage mangas from multiple **sources** (sites), like [Mangadex](https://mangadex.org) and [MangaHub](mangahub.io), and keep track of the user defined status and last readed chapter of the mangas.
With the API routes you can:
- **Add a manga to the DB**: by supplying the manga URL, the API will identify if it's from one the sources, if yes, the source will get the manga metadata, like name, cover, and last released chapter (and the chapter metadata as well), and insert it on the DB.
  - Depending on the source, a manga can also have a **preferred group**, which can be the group that uploades chapters of the manga. This is useful when you want the links of chapters from a specific group in a source that enables multiple groups to upload the same chapters (_like Mangadex_).
- **Update the manga last readed chapter**: by supplying the manga URL and a chapter number, a source will get the chapter metadata, like name and URL, and update the last readed chapter of the manga.
- **Delete a manga from DB**.
- **Get a manga from DB using the manga ID (in the DB) or URL**.
- **Get all mangas**.

## DB schema
![DB](https://github.com/diogovalentte/manga-dashboard-api/assets/49578155/45764965-9fc9-4b76-b1a2-a3b4742ab0b1)
- The chapters table stores two types of chapters: the last uploaded chapter of a manga scraped from the manga page and the last readed chapter of the manga.
- The column chapters.chapter is the chapter number.
- The column chapters.type is the type of the chapter. 1 == last uploaded chapter, 2 == last readed chapter.
- The column chapters.updated_at can be one of two things: the time where the last uploaded chapter was uploaded, or the time where the chapter was read.
