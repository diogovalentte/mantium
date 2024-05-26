# Mantium

Mantium is a dashboard for tracking mangas from multiple source sites, like [Mangadex](https://mangadex.org) and [ComicK](https://comick.io). This project doesn't download the chapter images, it downloads the manga metadata (name, URL, cover, etc.) and chapter metadata (number, name, URL), and shows them in the dashboard, where you manage the mangas you're tracking.

- This project currently can track mangas on [Mangadex](https://mangadex.org), [ComicK](https://comick.io), and [MangaHub](https://mangahub.io).

The basic workflow is:

1. You find an interesting manga on a site.
2. Add it to Mantium, set its status (reading, dropped, etc.), and the last chapter you read. Now you see the manga in the dashboard.
3. You configure Mantium to periodically (like every 30 minutes) get the metadata of all your mangas from the source sites, like new cover images or the newly released chapter. You also configure it to notify you when a new chapter is released.
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
  - You can add a manga to the dashboard using the manga URL. You also have to set the manga status and the last chapter you read.
  - When you click the button to highlight a manga, it shows a form to update the manga status or last read chapter or delete the manga.
  - Set some configs, like the number of columns in the dashboard.

# iFrame

The API has an endpoint that returns an HTML code to be used as iFrame. It's a minimalist version of the dashboard, showing only mangas with unread chapters with status reading or completed.

- **Note**: check the API docs to see which query arguments the iFrame endpoint needs.

![image](https://github.com/diogovalentte/mantium/assets/49578155/e88d85f2-0109-444a-b225-878a5db01400)

When you add an iFrame widget in your Homarr dashboard, it's **>your<** web browser that fetches the HTML content from the API and shows it to you, not Homarr. So your browser needs to be able to access the API, that's how an iFrame works.

- **Examples**:
  - If you run the API on your server, you need to add your server IP address + port in the Homarr widget, and you need to make sure your browser can access this IP + port.
  - If you're accessing Homarr or another dashboard with a domain and using HTTPS (like `https://dash.domain.com`), you also need to access this API with a domain and use HTTPS (like `https://mantium-api.domain.com`) to add the iFrame to Homarr. If you try to use HTTP with your HTTPS, your browser will block the iFrame.

# Kaizoku integration

More about it [here](https://github.com/diogovalentte/mantium/blob/main/kaizoku-integration.md).

# Running

By default, the API will be available on port `8080` and the dashboard on port `8501`. They're not accessible by other machines. To access the API and the dashboard in other machines, you need to run them behind a reverse proxy or run the containers in [host network mode](https://docs.docker.com/network/drivers/host/).

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

# IMPORTANT!

- The dashboard and the API don't have any authentication system, so anyone who can access the dashboard and the API can do whatever they want. You can add an authentication portal like [Authelia](https://github.com/authelia/authelia) or [Authentik](https://github.com/goauthentik/authentik) in front of the dashboard to protect it and don't expose the API at all.
  - If you want to use the iFrame returned by the API, you can also put an authentication portal in front of it, if the API and dashboard containers are in the same Docker network. The dashboard will communicate with the API using the API's container name.

# Commom problems:

### A manga is removed from the source site or its URL changes

If a manga is removed from the source site (_like Mangedex_) or its URL changes, the API will not be able to track it, as it saves the manga URL on the database when you add the manga in the dashboard and continues to use this URL forever. If this happens, the dashboard/API logs will show an error like this:

```
{"message":"(comick.io) Error while getting manga with URL 'https://comick.io/comic/witch-hat-atelier' chapters from source: Error while getting chapters metadata: Error while making 'GET' request: Non-200 status code -\u003e (404). Body: {\"statusCode\":404,\"message\":\"Not Found\"}"}
```

To fix this, you need to delete the manga and add it again from another source site or use its new URL.

### Other errors

Sometimes the URL of a source site or its API changes or the dashboard can't connect to the API. In these cases, open an issue describing what you tried to do that resulted in an error, the error message if it shows, and the dashboard/API logs at the time.

# Running manually

1. Export the environment variables in the `.env.example` file.

## API

2. Inside the `api/` folder, install the API dependencies:

```bash
go mod download
```

3. Start the API:

```bash
go run main.go
```

## Dashboard

4. Inside the `dashboard/` folder, install the dashboard dependencies:

```bash
pip install -r requirements.txt
```

5. Start the dashboard:

```bash
streamlit run 01_📖_Dashboard.py
```

6. Access the dashboard on `http://localhost:8501`

# API

The API is where your mangas are managed and tracked, it gets the mangas metadata from the sites and stores them on the database.

After starting the API, you can find the API docs under the path `/v1/swagger/index.html`, like `http://192.168.1.44/v1/swagger/index.html` or `https://sub.domain.com/v1/swagger/index.html`, depending on how you access the API.

You can set the API to automatically update the metadata (last upload chapter, cover image, etc.) of all your mangas periodically. You can also get notified when a new chapter of a manga with the status _reading or completed_ is released in [Ntfy](https://github.com/binwiederhier/ntfy).

- If an error occurs in the background while updating the mangas metadata, the dashboard and iframe will show this error.
