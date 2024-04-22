# Mantium

Mantium is a dashboard for tracking from multiple source sites, like [Mangadex](mangadex.org) and [ComicK](comick.io). This project doesn't download the chapter images, it downloads the manga metadata (name, URL, cover, etc.) and chapter metadata (number, name, URL), and shows them in the dashboard, where you manage the mangas you're tracking. It also has links to the mangas and chapters.

- This project currently can track mangas on: [Mangadex](mangadex.org), [ComicK](comick.io), and [MangaHub](mangahub.io).

![image](https://github.com/diogovalentte/mantium/assets/49578155/69e5d417-e3c8-4a4e-9613-b47eff54ecce)

- I'm not a web developer or a UX designer, the dashboard is not great, but it's useful.
- I don't really plan on actively maintaining the project. I'll add features or new sites only when want, but I'll fix bugs and accept pull requests. And of course, anyone can fork the project.

# Project structure and configuration

The project is divided into two: the **dashboard** and the **API**.

## Dashboard

The dashboard shows you the mangas you're tracking and is where you interact with the system.
- In the main part, there are columns of the mangas you're tracking in cards (*you can configure the number of columns in the dashboard*):

<p align="center">
  <img src="https://github.com/diogovalentte/mantium/assets/49578155/83cc24e4-31de-435b-9ea6-22a4aecb8c66">
</p>

- On the sidebar, you can:
  - Search for a manga using its name, filter the mangas by status (*reading, completed, dropped, on hold, plan to read, all*), order the mangas by name, last chapter read, last chapter upload, number of chapters, and unread (*shows unread mangas first, ordering by last upload chapter*), and reverse the sort.
  - You can add a manga to the dashboard using the manga URL. You also have to set the manga status and the last chapter you read.
  - When you click the button to highlight a manga, it shows a form to update the manga status or last read chapter or delete the manga.

## API

The API is where your mangas are actually managed and tracked, it gets the mangas metadata from the sites and stores them on the database.

After starting the API, you can find the API docs under the path `/v1/swagger/index.html`, like `http://192.168.1.44/v1/swagger/index.html` or `https://sub.domain.com/v1/swagger/index.html`, depending on how you access the API.

You can set the API to automatically update the metadata (last upload chapter, cover image, etc.) of all your mangas periodically. You can also get notified when a new chapter of a manga with the status *reading or completed* is released in [Ntfy](https://github.com/binwiederhier/ntfy).
- If an error occurs in the background while updating the mangas metadata, the dashboard and iframe will show this error.

# Running

By default, the API will be available on port `8080` and the dashboard on port `8501`. They're not accessible by other machines. To access the API and the dashboard by other machines, you need to run them behind a reverse proxy or run the containers in [host network mode](https://docs.docker.com/network/drivers/host/).

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

# Homarr iFrame

The API has an endpoint that returns an HTML code to be used as iFrame (designed to be used in [Homarr](https://github.com/ajnart/homarr)). It's a minimalist version of the dashboard, showing only mangas with unread chapters.

- **Note**: check the API docs to see which query arguments the iFrame endpoint needs.

![image](https://github.com/diogovalentte/mantium/assets/49578155/e88d85f2-0109-444a-b225-878a5db01400)

When you add an iFrame widget in your Homarr dashboard, it's **>your<** web browser that fetches the HTML content from the API and shows it to you, not Homarr. So your browser needs to be able to access the API, that's how an iFrame works.

- **Examples**:
  - If you run the API on your server, you need to add your server IP address + port in the Homarr widget, and you need to make sure your browser can access this IP + port.
  - If you're accessing Homarr or another dashboard with a domain and using HTTPS (like `https://dash.domain.com`), you also need to access this API with a domain and use HTTPS (like `https://mantium-api.domain.com`) to add the iFrame to Homarr. If you try to use HTTP with your HTTPS, your browser will block the iFrame.
 
# Kaizoku integration
You can enable the [Kaizoku](https://github.com/oae/kaizoku) integration using environment variables. The integration will:
- Try to add the manga to Kaizoku when you add it to the dashboard.
- If the background job to update the mangas metadata detects newly released chapters, it will add a job to the Kaizoku queue to check all your Kaizoku mangas and download the new chapters.
- If there are already mangas on your dashboard, the API has a route to add the mangas on your dashboard to Kaizoku. Check the API docs.

### Limitations
- Kaizoku uses [Mangal](https://github.com/metafates/mangal) under the hood to download the chapters. Mangal can only download from configured sources, like the built-in Mangadex source.
  - There is no built-in source for ComicK, but you can add a custom one. Download the Lua script `api/defaults/ComicK.lua` in this repository and add it to the folder `/config/.config/mangal/sources` of your Kaizoku Docker container and restart it.
  - There is no built-in or custom source for Mangahub, so the mangas from Mangahub in your dashboard will not work with Kaizoku.
- Kaizoku only accepts mangas that have a page on [Anilist](https://anilist.co/search/manga). Mangal will find Anilist mangas with the same name as your manga, if it doesn't find any, Kaizoku will not add your manga.
  - Sometimes Kaizoku gets the wrong Anilist ID for the manga.
    - If multiple mangas have the same name, Kaizoku will use the Anilist ID of the oldest manga. In this case, you have to manually set the right Anilist ID on Kaizoku.
    - If the manga doesn't have a page on Anilist, but there is another manga with the same name in Anilist, Kaizoku will use this Anilist manga ID. In this case, you can only delete the manga from Kaizoku, or add the correct manga to Anilist and set the right Anilist ID on Kaizoku.
- In my case, I tried to add 96 mangas to Kaizoku, 10 were not added because they didn't have an Anilist ID. 3 had the wrong Anilist ID, from these, 2 I just needed to set the right Anilist ID, but one of them didn't have a page on Anilist, so I can't change it to the right Anilist ID unless the correct manga is added to Anilist.

# Commom problems:
### A manga is removed from the source site or its URL changes
If a manga is removed from the source site (*like Mangedex*) or its URL changes, the API will not be able to track it, as it saves the manga URL on the database when you add the manga in the dashboard and continues to use this URL forever. If this happens, the dashboard/API logs will show an error like this:

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
