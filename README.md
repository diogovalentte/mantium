# Mantium

Mantium is a dashboard for tracking mangas from multiple source sites, like [Mangadex](https://mangadex.org) and [ComicK](https://comick.io). This project doesn't download the chapter images, it downloads the manga metadata (name, URL, cover, etc.) and chapter metadata (number, name, URL), to show in the dashboard, where you manage the mangas you're tracking.

- This project currently can track mangas on [Manga Plus](https://mangaplus.shueisha.co.jp), [MangaDex](https://mangadex.org), [ComicK](https://comick.io), and [MangaHub](https://mangahub.io).

The basic workflow is:

1. You find an interesting manga on a site.
2. Add it to Mantium, set its status (reading, dropped, etc.), and the last chapter you read. Now you see the manga in the dashboard.
3. You configure Mantium to periodically (like every 30 minutes) check for new chapters. You also configure it to notify you when a new chapter is released.
4. After getting notified that a new chapter has been released, you read it and set in Mantium that the last chapter you read is the last released chapter.
5. That's how you track a manga in Mantium.

![image](https://github.com/diogovalentte/mantium/assets/49578155/69e5d417-e3c8-4a4e-9613-b47eff54ecce)

# Dashboard

The dashboard shows you the mangas you're tracking and is where you interact with them.

- In the main part, there are columns of the mangas you're tracking in cards:

<p align="center">
  <img src="https://github.com/diogovalentte/mantium/assets/49578155/83cc24e4-31de-435b-9ea6-22a4aecb8c66">
</p>

- On the sidebar, you can:
  - Search for a manga using its name, filter the mangas by status (_reading, completed, dropped, on hold, plan to read, all_), order the mangas by name, last chapter read, last chapter upload, number of chapters, and unread (_shows unread mangas first, ordering by last upload chapter_), and reverse the sort.
  - You can add a manga to the dashboard using the manga URL. You also set the manga status and the last chapter you read.
  - When you click the button to highlight a manga, it shows a form to update the manga status, last read chapter, set a custom manga cover, or delete it.
  - Set some configs, like the number of columns in the dashboard.

# iFrame

The Mantium API has an endpoint that returns an iFrame. It's a minimalist version of the dashboard, showing only mangas with unread chapters with status reading or completed. You can add it to a dashboard, for example.

- **Note**: check the API docs to see which query arguments the iFrame endpoint needs.

![image](https://github.com/diogovalentte/mantium/assets/49578155/e88d85f2-0109-444a-b225-878a5db01400)

When you add an iFrame widget in your dashboard, it's **>your<** web browser that fetches the iFrame from the API and shows it to you, not the dashboard. So your browser needs to be able to access the API, that's how an iFrame works.

- **Examples**:
  - If you run the API on your server, add your server IP address + port in the dashboard and make sure your browser can access this IP + port.
  - If you're accessing Homarr or another dashboard with a domain and using HTTPS (like `https://dash.domain.com`), you also need to access this API with a domain and use HTTPS (like `https://mantium-api.domain.com`) to add the iFrame to your dashboard. If you try to use HTTP with your HTTPS, your browser will block the iFrame.

# Check manga updates and notify

You can set Mantium to get metadata of the manga you're tracking from the source sites. If the manga cover image or name changes, Mantium will update its store data. If a chapter is released, Mantium will update the manga's last released chapter. You can set when Mantium will check for updates, like every 30 minutes.

You can set Mantium to notify you in a [Ntfy](https://github.com/binwiederhier/ntfy) topic when a manga with the status "reading" or "completed" has a newly released chapter.

- If an error occurs in the background while updating the mangas metadata or notifying, a warning will appear in the dashboard and iframe. The error will also be printed to the API and dashboard container, and you can click to see the error in the dashboard.

# Kaizoku integration

More about it [here](https://github.com/diogovalentte/mantium/blob/main/kaizoku-integration.md).

# API

When you interact with the dashboard, it requests the API to execute things, like adding, updating, and deleting a manga from Mantium. The API is where your mangas are managed and tracked internally, it gets the mangas metadata from the source sites and stores them on the database.

After starting the API, you can find the API docs under the path `/v1/swagger/index.html`, like `http://192.168.1.44/v1/swagger/index.html` or `https://sub.domain.com/v1/swagger/index.html`, depending on how you access the API.

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

### This project doesn't have any authentication system

The dashboard and the API don't have any authentication system, so anyone who can access the dashboard or the API can do whatever they want. You can add an authentication portal like [Authelia](https://github.com/authelia/authelia) or [Authentik](https://github.com/goauthentik/authentik) in front of the dashboard to protect it and don't expose the API at all.

- If you want to use the iFrame returned by the API, you can still put an authentication portal in front of the API if the API and dashboard containers are in the same Docker network. The dashboard will communicate with the API using the API's container name.

### What to do when a manga is removed from the source site or its URL changes

If a manga is removed from the source site (_like Mangedex_) or its URL changes, the API will not be able to track it, as it saves the manga URL on the database when you add the manga in the dashboard and continues to use this URL forever. If this happens, the dashboard/API logs will show an error like this:

```
{"message":"(comick.io) Error while getting manga with URL 'https://comick.io/comic/witch-hat-atelier' chapters from source: Error while getting chapters metadata: Error while making 'GET' request: Non-200 status code -\u003e (404). Body: {\"statusCode\":404,\"message\":\"Not Found\"}"}
```

To fix this, delete the manga and add it again from another source site or use its new URL.

### Source site URL changes

Sometimes the URL of a source site changes (_like comick.fun to comick.io_). In this case, please open an issue if a new release with the updated URL is not released yet.

### Manga Plus source

Only the first and last chapters are available on the Manga Plus site, so most chapters do not show on Mantium. I recommend reading the manga in the other source sites and when you get to the last chapter, remove the manga and add it again from the Manga Plus source.

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

6. Export the API address environment variable:

```bash
export API_ADDRESS=http://localhost:8080
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
