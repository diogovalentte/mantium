# Mantium

**Mantium is a cross-site manga tracker**, which means that you can track manga from multiple source sites, like [Mangadex](https://mangadex.org) and [ComicK](https://comick.io). Mantium doesn't download the chapter images; it downloads the manga metadata (name, URL, cover, etc.) and chapter metadata (number, name, URL) from the source site and shows them in a dashboard and iFrame to put in your dashboard service.

- Mantium currently can track mangas on [Manga Plus](https://mangaplus.shueisha.co.jp), [MangaDex](https://mangadex.org), [ComicK](https://comick.io), [MangaHub](https://mangahub.io), [MangaUpdates](https://www.mangaupdates.com/), [RawKuma](https://rawkuma.com/), [KLManga](https://klmanga.rs/), and [JManga](https://jmanga.is).

**The basic workflow is:**

1. You find an interesting manga on a site.
2. Add the manga to Mantium. Set its status (reading, dropped, etc.) and (_optionally_) the last chapter you read from the list of the manga's released chapters.
3. You also configure Mantium to periodically check for new chapters (like every 30 minutes) in your mangas and notify you when they are released.
4. After being notified that a new chapter has been released, you read it and set it in Mantium that you read the last released chapter.
5. That's how you track a manga in Mantium.

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/W7W012U1PL)

# Dashboard

By default, the dashboard shows the mangas with unread chapters first, ordering by the last released chapter. Unread manga names have an animation effect that changes the name color to catch your attention.

You have two display options: **grid view** and **list view**.

## Grid View

<p align="center">
  <img src="https://github.com/user-attachments/assets/0c696396-fc78-4843-bdad-710888c0d016">
</p>

## List View

<p align="center">
  <img src="https://github.com/user-attachments/assets/8b0f810f-d2ce-4fe8-a9c5-bd215541d67b">
</p>

You can add a manga to Mantium using the manga URL or by searching it by name in Mantium:

|                                 Using the Manga URL                                  |                                  Searching by Name                                   |
| :----------------------------------------------------------------------------------: | :----------------------------------------------------------------------------------: |
| ![](https://github.com/user-attachments/assets/a3e99e90-9b7f-4a7d-870f-cec25bf15b05) | ![](https://github.com/user-attachments/assets/1a615598-d56e-4bfe-9e14-40d01d1ce0ff) |

## Sidebar

- On the sidebar, you can:
  - Filter the mangas by name, and status (_reading, completed, dropped, on hold, plan to read, all_).
  - Order the mangas by name, last chapter read, last chapter upload, number of chapters, and unread (_shows unread mangas first, ordering by last upload chapter_), and reverse the sort.
  - Click to add a manga to Mantium.
  - Set the dashboard configs like the display mode, number of columns in the dashboard, etc.

# iFrame

Mantium also has an endpoint that returns an iFrame. It's a minimalist version of the dashboard, showing only mangas with unread chapters with status reading or completed. You can add it to a dashboard, for example. More about the iframe [here](https://github.com/diogovalentte/mantium/edit/main/README.md#iframe-usage).

![image](https://github.com/diogovalentte/mantium/assets/49578155/e88d85f2-0109-444a-b225-878a5db01400)

# Multimanga

A **Multimanga** is like a container for multiple mangas. When you add a manga to Mantium, it creates a multimanga and adds the manga to the multimanga. You actually never interact with the manga, only the multimanga. But why?

The **multimanga** feature solves the issue of when you want to track the **same manga** in multiple source sites so that you are notified whenever a new chapter is released as soon as possible in whatever source. You could add the same manga for each source, but they would act as **completely different mangas**. Each one would appear on a card in the dashboard/iframe, you would need to set the last read chapter for each of them and be notified of new chapters from each of them.

The multimanga feature solves this issue. With it, you add the same manga from multiple sources to the same multimanga and interact only with the multimanga. No multiple notifications or setting the last read chapter for all of them, you just set the multimanga last read chapter, and only one card will appear in the dashboard/iframe!

## Current manga

A multimanga always has a manga called **current manga**, which is one of the multimanga's mangas. The dashboard/iframe shows the current manga's name, cover image, and last released chapter. When you select the last read chapter, the list of the current manga chapters is shown. When you're notified of a new chapter, the newest chapter of the current manga is sent in the notification.

Based on the last released chapter, Mantium **tries** to set the current manga to the manga with the newest released chapter by using the following rules. Take, for example, a multimanga with two mangas. Mantium will compare the mangas last released chapter this way:

1. If the chapters' numbers are equal, Mantium sets the manga that released the chapter last as the current manga.
2. If they're not equal, Mantium will pick the manga with the biggest chapter number.
3. Depending on the manga and source, the chapter's number can not be a number at all. When one of the manga's chapter numbers is not a number, Mantium will pick the manga that released the chapter last as the current manga.

Mantium decides which manga should be the current manga whenever you **add/remove** a manga from a multimanga and in the [periodic job that updates the mangas in the background](https://github.com/diogovalentte/mantium?tab=readme-ov-file#check-manga-updates-and-notify).

The images below show the pop-up when you click the "**Highlight**" button of one of the mangas in the dashboard. It shows a form to delete the multimanga, update its status, last read chapter, and cover image, add mangas to the multimanga by searching by name or using a URL, and manage the multimanga's mangas.

|                                   Edit Multimanga                                    |                             Edit Multimanga Cover Image                              |                               Manage Multimanga Mangas                               |
| :----------------------------------------------------------------------------------: | :----------------------------------------------------------------------------------: | :----------------------------------------------------------------------------------: |
| ![](https://github.com/user-attachments/assets/441d2423-2276-4512-ab6c-edfc80e909f2) | ![](https://github.com/user-attachments/assets/e0c4acb1-6840-4824-817c-cd208867a04a) | ![](https://github.com/user-attachments/assets/020681aa-8e59-4f2f-aefe-c92a89251fe8) |

# Custom Manga

Mantium allows you to add manga, manhwa, light novels, etc., that aren't from one of the supported source sites. You must manually track these manga, providing info about them like name, URL, cover image, etc. You **can** also provide the **next chapter** you should read from the manga and update it every time you read a chapter.

- Custom mangas are not treated as **multimangas**.

When you read the last available chapter from a custom manga, check the "**No more chapters available**" checkbox so the manga is considered read instead of unread.

![image](https://github.com/user-attachments/assets/a8b3c88f-da7e-4a23-b6c6-077133edecc9)

# Check manga updates and notify

Mantium can periodically get the metadata of the mangas you're tracking from their source sites (like every 30 minutes). If the manga metadata (like the cover image, name, or last release chapter) changes from the currently stored metadata, Mantium updates it.

You can also set Mantium to notify you when a manga with the status "reading" or "completed" has a newly released chapter.

- If an error occurs in the background while updating the manga's metadata or notifying, a warning will appear on the dashboard and iframe. You can disable this warning.

# Integrations

Mantium has integrations, like:

- [Ntfy](https://github.com/binwiederhier/ntfy) to notify you of new chapters.
- [Kaizoku](https://github.com/oae/kaizoku), [Tranga](https://github.com/C9Glax/tranga/tree/master), and [Suwayomi](https://github.com/Suwayomi) to actually download your chapters.

More about the integrations [here](https://github.com/diogovalentte/mantium/blob/main/integrations.md).

# Running

By default, the API will be available on port `8080` and it's not accessible by other machines. To access the API in other machines, you run the container in [host network mode](https://docs.docker.com/network/drivers/host/) or behind a reverse proxy.

- For convenience, you can change the API port using the environment variables `API_PORT`.

## Docker Compose

1. There is a `docker-compose.yml` file in this repository. You can clone this repository to use this file or create one yourself.
1. Create a `.env` file, it should be like the `.env.example` file and be in the same directory as the `docker-compose.yml` file.
1. Start the containers:

```sh
docker compose up -d
```

## Running manually/setting development environment

The steps are at the [bottom of this README](https://github.com/diogovalentte/mantium/edit/main/README.md#running-manually).

# Notes:

### iFrame usage

The **Mantium API** has the endpoint `/v1/mangas/iframe` that returns an iFrame. When you add an iFrame to your dashboard, it's **>your<** web browser that fetches the iFrame from the API and shows it to you, not your dashboard service running on your server. So your browser needs to be able to access the Mantium API.

- **Examples**:
  - If you run the API on port 8080 on your server, use your server IP address + port 8080 to see the iframe, and make sure your browser can access this IP + port.
  - If you want to use the iframe in a dashboard that uses HTTPS (like `https://dashboard.domain.com`), you also need to get the iFrame using HTTPS (like `https://mantium-api.domain.com`) to add the iFrame to your dashboard. If you try to use HTTP with your HTTPS, your browser will block the iFrame.

The iFrame has the following arguments:

- `api_url` (**not optional**): The URL of the Mantium API that your browser uses to connect to the API, like `https://mantium-api.domain.com`.
- `theme` (**optional**): The theme of the iFrame. Can be `light` or `dark`. Defaults to `light`.
- `limit` (**optional**): The number of mangas to show in the iFrame.
- `showBackgroundErrorWarning` (**optional**): If an error occurs in the background, a warning will appear on the iFrame. Defaults to `true`.

**Example**: `https://mantium-api.domain.com/v1/mangas/iframe?api_url=http://mantium-api.domain.com&theme=dark&limit=5&showBackgroundErrorWarning=false`

### Mantium doesn't have any authentication system

The dashboard and the API don't have any authentication system, so anyone who can access the dashboard or the API can do whatever they want. You can add an authentication portal like [Authelia](https://github.com/authelia/authelia) or [Authentik](https://github.com/goauthentik/authentik) in front of the dashboard to protect it and not expose the API at all.

- If you want to use the iFrame returned by the API, you can still put an authentication portal in front of the API if the API and dashboard containers are in the same Docker network. The dashboard will communicate with the API using the API's container name.

### API

The API docs are under the path `/v1/swagger/index.html`.

### Manga Plus source

Only the first and last chapters are available on the Manga Plus site, so most chapters do not show on Mantium. I recommend reading the manga in the other source sites and when you get to the last chapter, remove the manga and add it again from the Manga Plus source.

### KLManga and JManga sources

The KLManga and JManga sources don't show the time when the chapters are released, so when you add a manga to Mantium, it sets the last released chapter's release date to the current time. In the background job that updates the mangas metadata, if it detects that the last released chapter's release date is the current time, it sets the release date to the current time.

### Manga Updates source

The Manga Updates source is very different from the other sources:

- Mantium tracks the releases instead of actual chapters. Different groups can release the same chapter.
- The chapters are listed by upload date. This means that if a group releases chapters 1-50 and another group rereleases chapter 2, it'll be considered the latest chapter instead of chapter 50 of the other group.
  - This is a limitation of MangaUpdates, which can't properly sort the chapters by chapter number.

### Source site down

Sometimes the source sites can be down for some time, like in maintenance. In these cases, there is nothing Mantium can do about it, and all interactions with manga from these source sites will fail.

### What to do when a manga is removed from the source site or its URL changes

If a manga is removed from the source site (_like Mangedex_) or its URL changes, the API will not be able to track it, as it saves the manga URL on the database when you add the manga in the dashboard and continues to use this URL forever. If this happens, the dashboard/API logs will show an error like this:

```
{"message":"(comick.io) Error while getting manga with URL 'https://comick.io/comic/witch-hat-atelier' chapters from source: Error while getting chapters metadata: Error while making 'GET' request: Non-200 status code -\u003e (404). Body: {\"statusCode\":404,\"message\":\"Not Found\"}"}
```

To fix this, delete the manga and add it again from another source site or use its new URL.

### Source site URL changes

Sometimes the URL of a source site changes (_like comick.fun to comick.io_). In this case, please open an issue if a new release with the updated URL has not been released yet.

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

6. The dashboard expects to connect to the API on the address `http://localhost:8080`. If the API is running at a different address, export the API address environment variable:

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
