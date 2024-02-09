# Mangas Dashboard
The dashboard manages mangas for the user. The user can add mangas from multiple **sources** (sites), like [Mangadex](mangadex.org) and [MangaHub](mangahub.io), and keep tracking it but setting status (like Reading, Completed, Dropped) and the last readed chapter.

# Mangas Dashboard API
The API is used by the Dashboard.

# Running
1. Create a .env.prod file, it should be like the .env.example file.
2. Build and start the services:
```sh
docker compose up -d --build
```
3. Access the dashboard on http://localhost:8501

# Testing
1. Create a .env.test file for testing, it should be like the .env.example file.
2. Start a PostgreSQL container like in the docker-compose.yml file.
3. Inside the **api/** folder, download the API Golang dependencies:
```sh
go mod download
```
4. Unncomment the lines on the **api/main.go** file and start the API:
```sh
go run main.go
```
5. Inside the **dashboard/** folder, install the dashboard Python requirements:
```sh
pip install -r requirements.txt
```
6. Start the dashboard:
```sh
streamlit run 01_📖_Dashboard.py
```
7. Access the dashboard on http://localhost:8501


# API Simple doc
This project's objective is to manage mangas from multiple **sources** (sites), like [Mangadex](https://mangadex.org) and [MangaHub](mangahub.io), and keep track of the user defined status and last readed chapter of the mangas.
With the API routes you can:
- **Add a manga to the DB**: by supplying the manga URL, the API will identify if it's from one the sources, if yes, the source will get the manga metadata, like name, cover, and last released chapter (and the chapter metadata as well), and insert it on the DB.
  - Depending on the source, a manga can also have a **preferred group**, which can be the group that uploades chapters of the manga. This is useful when you want the links of chapters from a specific group in a source that enables multiple groups to upload the same chapters (_like Mangadex_).
- **Update the manga last readed chapter**: by supplying the manga URL and a chapter, a source will get the chapter metadata, like name and URL, and update the last readed chapter of the manga.
- **Delete a manga from DB**.
- **Get a manga from DB using the manga ID (in the DB) or URL**.
- **Get all mangas**.

## DB schema
![DB](https://github.com/diogovalentte/manga-dashboard-api/assets/49578155/45764965-9fc9-4b76-b1a2-a3b4742ab0b1)
- The chapters table stores two types of chapters: the last uploaded chapter of a manga scraped from the manga page and the last readed chapter of the manga.
- The column chapters.chapter is the chapter.
- The column chapters.type is the type of the chapter. 1 == last uploaded chapter, 2 == last readed chapter.
- The column chapters.updated_at can be one of two things: the time where the last uploaded chapter was uploaded, or the time where the chapter was read.
