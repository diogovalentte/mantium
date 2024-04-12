# Mantium

Mantium is a dashboard for tracking from multiple sites, like [Mangadex](mangadex.org) and [ComicK](comick.io). This project doesn't download the chapter images, it downloads the manga metadata (name, URL, cover, etc.) and chapter metadata (number, name, URL), and shows them in the dashboard, where you manage the mangas you're tracking. It also has links to the mangas and chapters.

- It currently can track mangas on: [Mangadex](mangadex.org), [ComicK](comick.io), and [MangaHub](mangahub.io).

![image](https://github.com/diogovalentte/mantium/assets/49578155/69e5d417-e3c8-4a4e-9613-b47eff54ecce)

- I'm not a web developer or a UX designer, the dashboard is not great, but it's useful.
- I don't really plan on actively maintaining the project. I'll add features or new sites only when want, but I'll fix bugs and accept pull requests. And of course, anyone can fork the project.

# Project structure and configuration

The project is divided into two: the **dashboard** and the **API**.

## Dashboard

The dashboard shows you the mangas you're tracking and is where you interact with the system.
- On the sidebar, you can:
  - Search for a manga using its name, filter the mangas by status (*reading, completed, dropped, on hold, plan to read, all*), order the mangas by name, last chapter read, last chapter upload, number of chapters, and unread (*shows unread mangas first, ordering by last upload chapter*), and reverse the sort.
  - You can add a manga to the dashboard and start tracking it:

<p align="center">
  <img src="https://github.com/diogovalentte/mantium/assets/49578155/7b7842ad-cf3b-410d-955a-f58b3544664d">
</p>

  - When you highlight a manga, it shows a form to update the manga status or last read chapter, or delete the manga and stop tracking it.

<p align="center">
  <img src="https://github.com/diogovalentte/mantium/assets/49578155/2e05b853-958f-411a-815a-6809d2a7c8e8">
</p>

- In the main part, there are columns of the mangas you're tracking in cards (*you can configure the number of columns in the dashboard*):

<p align="center">
  <img src="https://github.com/diogovalentte/mantium/assets/49578155/83cc24e4-31de-435b-9ea6-22a4aecb8c66">
</p>


## API

The API is where your mangas are actually managed and tracked, it gets the mangas metadata from the sites and stores them on the database.

After starting the API, you can find the API docs under the path `/v1/swagger/index.html`, like `http://192.168.1.44/v1/swagger/index.html` or `https://sub.domain.com/v1/swagger/index.html`, depending on how you access the API.

You can set the API to automatically update the metadata (last upload chapter, cover image, etc.) of all your mangas periodically. You can also get notified when a new chapter of a manga with the status *reading or completed* is released in [Ntfy](https://github.com/binwiederhier/ntfy).

# Running

By default, the API will be available on port `8080` and the dashboard on port `8501`, and they're not accessible by other machines. To be accessible by other machines, you need to run the containers behind a reverse proxy or run the containers in [host network mode](https://docs.docker.com/network/drivers/host/).

- You can change the API and dashboard ports using the environment variables `API_PORT` and `DASHBOARD_PORT` for convenience.

## Docker Compose

1. There is a `docker-compose.yml` file in this repository. You can clone this repository to use this file or create one yourself.
1. Create a `.env` file, it should be like the `.env.example` file and be in the same directory as the `docker-compose.yml` file.
1. Start the containers:

```sh
docker compose up -d
```

## Manually

1. Export the environment variables in the `.env.example` file.

### API

2. Inside the `api/` folder, install the API dependencies:

```bash
go mod download
```

3. Start the API:

```bash
go run main.go
```

### Dashboard

4. Inside the `dashboard/` folder, install the dashboard dependencies:

```bash
pip install -r requirements.txt
```

5. Start the dashboard:

```bash
streamlit run 01_📖_Dashboard.py
```

6. Access the dashboard on `http://localhost:8501`

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
  - If you're accessing Homarr or another dashboard with a domain and using HTTPS (like `https://dash.domain.com`), you also need to access this API with a domain and use HTTPS (like `https://mantium-api.domain.com`) in order to add the iFrame to Homarr. If you try to use HTTP with your HTTPS, your browser will block the iFrame.
