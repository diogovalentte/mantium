# Mantium

**Mantium is a cross-site manga tracker**, which means that you can track manga from multiple source sites, like [Mangadex](https://mangadex.org) and [ComicK](https://comick.io). Mantium doesn't download the chapter images, it downloads the manga metadata (name, URL, cover, etc.) and chapter metadata (number, name, URL) from the source site, and shows on the dashboard and iframe, where you manage the mangas you're tracking.

- Mantium currently can track mangas on [Manga Plus](https://mangaplus.shueisha.co.jp), [MangaDex](https://mangadex.org), [ComicK](https://comick.io), [MangaHub](https://mangahub.io), and [MangaUpdates](https://www.mangaupdates.com/).

**The basic workflow is:**

1. You find an interesting manga on a site.
2. Add the manga to Mantium. Set its status (reading, dropped, etc.), and the last chapter you read from the list of all released chapters.
3. You configure Mantium to periodically check for new chapters (like every 30 minutes) and notify you when a new chapter is released.
4. After getting notified that a new chapter has been released, you read it and set in Mantium that the last chapter you read is the last released chapter.
5. That's how you track a manga in Mantium.

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/W7W012U1PL)

# Dashboard

<p align="center">
  <img src="https://github.com/user-attachments/assets/0c696396-fc78-4843-bdad-710888c0d016">
</p>

You can add a manga to Mantium using the manga URL or by searching it by name in Mantium:

Using the Manga URL | Searching by Name
:-------------------------:|:-------------------------:
![](https://github.com/user-attachments/assets/a3e99e90-9b7f-4a7d-870f-cec25bf15b05)  |  ![](https://github.com/user-attachments/assets/1a615598-d56e-4bfe-9e14-40d01d1ce0ff)

When you click to highlight a manga, a popup opens where you can edit or delete the manga.

![image](https://github.com/user-attachments/assets/da3c454f-3362-418e-a064-aa75afdb5aaa)

## Sidebar
- On the sidebar, you can:
  - Filter the mangas by name, and status (_reading, completed, dropped, on hold, plan to read, all_).
  - Order the mangas by name, last chapter read, last chapter upload, number of chapters, and unread (_shows unread mangas first, ordering by last upload chapter_), and reverse the sort.
  - Click to add a manga to Mantium.
  - Set the dashboard configs like the number of columns in the dashboard.

# iFrame

The Mantium API has an endpoint that returns an iFrame. It's a minimalist version of the dashboard, showing only mangas with unread chapters with status reading or completed. You can add it to a dashboard, for example.

- **Note**: check the API docs to see which query arguments the iFrame endpoint needs.

![image](https://github.com/diogovalentte/mantium/assets/49578155/e88d85f2-0109-444a-b225-878a5db01400)

When you add an iFrame widget in your dashboard, it's **>your<** web browser that fetches the iFrame from the API and shows it to you, not the dashboard. So your browser needs to be able to access the API, that's how an iFrame works.

- **Examples**:
  - If you run the API on your server, add your server IP address + port in the dashboard and make sure your browser can access this IP + port.
  - If you're accessing Homarr or another dashboard with a domain and using HTTPS (like `https://dash.domain.com`), you also need to access this API with a domain and use HTTPS (like `https://mantium-api.domain.com`) to add the iFrame to your dashboard. If you try to use HTTP with your HTTPS, your browser will block the iFrame.

# Custom Manga
Mantium allows you to add mangas, manhwa, light novels, etc. that aren't from one of the supported source sites. You must manually track these mangas, providing info about them like name, URL, cover image, etc. You also **can** provide the **next chapter** you should read from the manga, and update it every time you read a chapter.

When you read the last available chapter from a custom manga, check the "**No more chapters available**" checkbox so that the manga is considered read instead of unread.

![image](https://github.com/user-attachments/assets/a8b3c88f-da7e-4a23-b6c6-077133edecc9)

# Multimanga
Imagine the same manga is released in the comick and mangadex sources, but sometimes the newest chapter is released first in comick and other times in mangadex. You may want to track the manga in both sources so that you are notified whenever a new chapter is released in both sources. But adding the same manga from different sources makes them act as completely different mangas. You would need to set the last read chapters for all mangas manually and be notified for all mangas.

The multimanga feature solves this issue. With it, **you can track the same manga from multiple sources, and they will be treated as the same manga**! No multiple notifications or setting the last read chapter for all of them!
- A deeper explanation of the multimanga feature and a demo video can be found [here](https://github.com/diogovalentte/mantium/blob/main/multimanga.md).

![image](https://github.com/user-attachments/assets/020681aa-8e59-4f2f-aefe-c92a89251fe8)

# Check manga updates and notify

You can set Mantium to periodically get the metadata of the mangas you're tracking from their source sites (like every 30 minutes). If the manga metadata (like the cover image, name changes, or last release chapter) changes from the stored metadata, Mantium updates it.

You can also set Mantium to notify you when a manga with the status "reading" or "completed" has a newly released chapter.

- If an error occurs in the background while updating the manga's metadata or notifying, a warning will appear on the dashboard and iframe.

# Integrations

Mantium has integrations, like:
- [Kaizoku](https://github.com/oae/kaizoku) and [Tranga](https://github.com/C9Glax/tranga/tree/master).
- [Ntfy](https://github.com/binwiederhier/ntfy).

More about the integrations [here](https://github.com/diogovalentte/mantium/blob/main/kaizoku-integration.md).

# Running

By default, the API will be available on port `8080` and the dashboard on port `8501`, and they're not accessible by other machines. To access the API and dashboard in other machines, you run the containers in [host network mode](https://docs.docker.com/network/drivers/host/) or behind a reverse proxy.

- For convenience, you can change the API and dashboard ports using the environment variables `API_PORT` and `DASHBOARD_PORT`.

## Docker Compose

1. There is a `docker-compose.yml` file in this repository. You can clone this repository to use this file or create one yourself.
1. Create a `.env` file, it should be like the `.env.example` file and be in the same directory as the `docker-compose.yml` file.
1. Start the containers:

```sh
docker compose up -d
```

## Manually

The steps are at the bottom of this README.

# Notes:

### Mantium doesn't have any authentication system

The dashboard and the API don't have any authentication system, so anyone who can access the dashboard or the API can do whatever they want. You can add an authentication portal like [Authelia](https://github.com/authelia/authelia) or [Authentik](https://github.com/goauthentik/authentik) in front of the dashboard to protect it and don't expose the API at all.

- If you want to use the iFrame returned by the API, you can still put an authentication portal in front of the API if the API and dashboard containers are in the same Docker network. The dashboard will communicate with the API using the API's container name.

### Manga Plus source

Only the first and last chapters are available on the Manga Plus site, so most chapters do not show on Mantium. I recommend reading the manga in the other source sites and when you get to the last chapter, remove the manga and add it again from the Manga Plus source.

### Manga Updates source
The Manga Updates source is very different from the other sources:
- Mantium tracks the releases instead of actual chapters. The same chapter can be released by different groups.
- The chapters are listed by upload date. This means that if a group uploads chapter 2 from a manga that another group already uploaded 50 chapters, chapter 2 will be considered the last chapter released. This is a limitation of MangaUpdates that can't properly sort the chapters by chapter number.

### Source site down
Sometimes the source sites can be down for some time, like in maintenance. In these cases, there is nothing Mantium can do about it, and all interactions with manga from this source site will fail.

### What to do when a manga is removed from the source site or its URL changes

If a manga is removed from the source site (_like Mangedex_) or its URL changes, the API will not be able to track it, as it saves the manga URL on the database when you add the manga in the dashboard and continues to use this URL forever. If this happens, the dashboard/API logs will show an error like this:

```
{"message":"(comick.io) Error while getting manga with URL 'https://comick.io/comic/witch-hat-atelier' chapters from source: Error while getting chapters metadata: Error while making 'GET' request: Non-200 status code -\u003e (404). Body: {\"statusCode\":404,\"message\":\"Not Found\"}"}
```

To fix this, delete the manga and add it again from another source site or use its new URL.

### Source site URL changes

Sometimes the URL of a source site changes (_like comick.fun to comick.io_). In this case, please open an issue if a new release with the updated URL is not released yet.

### API

The API docs are under the path `/v1/swagger/index.html`.

# Running manually

## Database

1. Create a `docker-compose.yml` file with the database service (`Docker` and `Docker Compose` are required):

```yml
services:
  mantium-db:
    container_name: mantium-db
    image: postgres:16-alpine
    volumes:
      - ./data/postgres-vol:/var/lib/postgresql/data
    environment:
      - POSTGRES_PORT=5432
      - POSTGRES_DB=postgres
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
    ports:
      - 5432:5432
    restart: unless-stopped
```

2. Start the database container:

```sh
docker compose up -d
```

## API

3. Export the API environment variables. The variables below are the only ones necessary to run the API, but more can be found in the `.env.example` file.

```
export POSTGRES_HOST=http://localhost
export POSTGRES_PORT=5432
export POSTGRES_DB=postgres
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export API_PORT=8080
```

4. Inside the `api/` folder, install the API dependencies:

```bash
go mod download
```

5. Start the API:

```bash
go run main.go
```

## Dashboard

6. The dashboard expects to connect to the API on the address `http://localhost:8080`. If the API is running in a different address, export the API address environment variable:

```bash
export API_ADDRESS=http://localhost:8081
```

7. Inside the `dashboard/` folder, install the dashboard dependencies:

```bash
pip install -r requirements.txt
```

8. Start the dashboard:

```bash
streamlit run 01_📖_Dashboard.py
```

9. Access the dashboard on `http://localhost:8501`
