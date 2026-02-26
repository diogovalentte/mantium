# Mantium

**Mantium is a cross-site manga tracker**. It allows you to track manga from multiple source sites such as [Mangadex](https://mangadex.org) and [MangaPlus](https://mangaplus.shueisha.co.jp).

Mantium does **not** download the chapter images. Instead, it retrieves:

- Manga metadata (name, URL, cover, etc.)
- Chapter metadata (number, name, URL)

This data is displayed and managed in a web dashboard and can be embedded via an iFrame in your own dashboard.

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/W7W012U1PL)

---

# Supported Sources

Mantium natively supports:

- [Manga Plus](https://mangaplus.shueisha.co.jp)
- [MangaDex](https://mangadex.org)
- [MangaHub](https://mangahub.io)
- [MangaUpdates](https://www.mangaupdates.com/)
- [RawKuma](https://rawkuma.com/)
- [KLManga](https://klmanga.rs/)
- [JManga](https://jmanga.is)

It can also automatically track manga from nearly all sites using the [Custom Manga](#custom-manga) feature.

# Basic Workflow

1. Find a manga on a supported site.
2. Add it to Mantium.
3. Set its status (reading, dropped, completed, etc.).
4. Optionally set the last chapter you have read.
5. Configure periodic checks (e.g., every 30 minutes) to detect new chapters.
6. When notified about a new chapter, read it.
7. Update the last read chapter in Mantium.

This keeps your reading progress synchronized across sources.

---

# Dashboard

By default, the dashboard:

- Shows manga with unread chapters first.
- Orders them by latest released chapter.
- Highlights unread manga titles with an animation effect.

Two display modes are available:

- Grid view
- List view

## Grid View

<p align="center">
  <img src="https://github.com/user-attachments/assets/0c696396-fc78-4843-bdad-710888c0d016">
</p>

## List View

<p align="center">
  <img src="https://github.com/user-attachments/assets/8b0f810f-d2ce-4fe8-a9c5-bd215541d67b">
</p>

You can add manga from natively supported sites in two ways:

- By pasting the manga URL.
- By searching for the manga by name directly in Mantium.

|                                 Using the Manga URL                                  |                                  Searching by Name                                   |
| :----------------------------------------------------------------------------------: | :----------------------------------------------------------------------------------: |
| ![](https://github.com/user-attachments/assets/a3e99e90-9b7f-4a7d-870f-cec25bf15b05) | ![](https://github.com/user-attachments/assets/1a615598-d56e-4bfe-9e14-40d01d1ce0ff) |

## Sidebar

From the sidebar, you can:

- Filter manga by:
  - Name
  - Status (reading, completed, dropped, on hold, plan to read, all)
- Sort by:
  - Name
  - Last chapter read
  - Last uploaded chapter
  - Number of chapters
  - Unread (prioritizes unread manga)
- Reverse the sorting order.
- Add new manga.
- Configure dashboard settings (display mode, number of columns, etc.).

---

# iFrame

Mantium provides an endpoint that returns a minimalist iFrame version of the dashboard.

The iFrame:

- Shows only manga with unread chapters.
- Includes only manga with status reading or completed.
- Is designed to be embedded in external dashboards.
- More details are available in the [iFrame usage section below](#iframe-usage).

![image](https://github.com/diogovalentte/mantium/assets/49578155/e88d85f2-0109-444a-b225-878a5db01400)

# Multimanga

A **Multimanga** is a container for multiple entries of the same manga across different source sites.

When you add a manga to Mantium, a Multimanga is automatically created. You always interact with the Multimanga — not the individual source entries.

## Why Multimanga?

If you track the same manga on multiple sites independently:

- Each entry appears separately in the dashboard and iFrame.
- You receive multiple notifications.
- You must update the last read chapter for each source.

**With Multimanga:**

- All sources are grouped.
- Only one dashboard and iFrame card is shown.
- You set the last read chapter once.
- You receive a single notification.

## Current manga

Each Multimanga has a **current manga**, which determines:

- Which display name, cover image, and last released chapter to be shown in the dashboard and iFrame
- Chapter list shown when selecting last read chapter
- Chapter included in notifications

Mantium automatically selects the current manga to be the one with the newest released chapter using these rules:

1. If chapter numbers are equal → choose the one released most recently.
2. If chapter numbers differ → choose the one with the highest chapter number.
3. If chapter numbers are non-numeric → choose the one released most recently.

This selection runs:

- When adding/removing a manga from a Multimanga.
- During the [background update job](https://github.com/diogovalentte/mantium?tab=readme-ov-file#background-upgrades-and-notifications).

You can manage Multimanga settings via the Highlight button, including:

- Updating status
- Changing last read chapter
- Updating the cover image to one you prefer
- Adding or removing source entries
- Deleting the Multimanga

|                                   Edit Multimanga                                    |                             Edit Multimanga Cover Image                              |                               Manage Multimanga Mangas                               |
| :----------------------------------------------------------------------------------: | :----------------------------------------------------------------------------------: | :----------------------------------------------------------------------------------: |
| ![](https://github.com/user-attachments/assets/441d2423-2276-4512-ab6c-edfc80e909f2) | ![](https://github.com/user-attachments/assets/e0c4acb1-6840-4824-817c-cd208867a04a) | ![](https://github.com/user-attachments/assets/020681aa-8e59-4f2f-aefe-c92a89251fe8) |

---

# Custom Manga

Custom Manga allows you to track content not supported natively (manga, manhwa, light novels, etc.).

You must manually provide:

- Name
- URL
- Cover image (or use default)
- Last read chapter

Custom manga are **not part of a Multimanga**.

## Last Released Chapter Selectors

Custom manga can automatically detect new chapters by getting the web page defined in the custom manga's URL field and applying selectors to get the chapter number and URL:

- **Selector**:
  CSS or XPath selector used to locate the element.
  Add the prefix `css:` for CSS selectors, and `xpath:` for XPATH selectors (ex: `css:div.chapter-box > h4:first-child > a span`). XPATH selectors also work with XML pages.
- **Attribute (optional)**:
  Attribute containing the chapter name or URL (e.g., href).
  If empty, the element text is used.
- **Regex (optional)**:
  Extracts the chapter number from the selected value.
- **Get First (optional)**:
  If enabled, selects the first matching element; otherwise, selects the last.
- **Use Browser (optional)**:
  Uses a headless browser if the page requires JavaScript rendering.
  Otherwise, a simple HTTP GET request is used.

> [!NOTE]
> If the URL selector doesn't return a string that starts with `http`, Mantium will consider it a relative URL and will prepend the manga URL to it. For example, if the manga URL is `https://example.com/one-piece` and the URL selector returns `/chapter1000`, Mantium will consider the chapter URL to be `https://example.com/chapter1000`.

<img width="436" height="1351" alt="image" src="https://github.com/user-attachments/assets/a057fa8a-8ebd-4b95-a648-388d366b7fbb" />

---

# Background Updates and Notifications

Mantium can periodically:

- Refresh manga metadata from the source sites.
- Detect new chapters.
- Send notifications for manga with status reading or completed.

If an error occurs during background processing:

- A warning is displayed on the dashboard and iFrame.
- This warning can be disabled.

Custom manga selectors are also checked during background updates.

---

# Integrations

Mantium has integrations with:

- [Ntfy](https://github.com/binwiederhier/ntfy) for new chapter notifications
- [Kaizoku](https://github.com/oae/kaizoku)
- [Tranga](https://github.com/C9Glax/tranga/tree/master)
- [Suwayomi](https://github.com/Suwayomi)

See [integrations.md](https://github.com/diogovalentte/mantium/blob/main/integrations.md) for details.

---

# Running

By default:
- API runs on port `8080`.
- It is not exposed externally unless configured.

To expose it:
- Use [Docker host network mode](https://docs.docker.com/network/drivers/host/), or
- Place it behind a reverse proxy.

`API_PORT` can be used to change the default port.

## Docker Compose

1. Clone the repository or create your own `docker-compose.yml` based on the one in this repository.
1. Create a `.env` file based on `.env.example`.
1. Start services:

```sh
docker compose up -d
```

## Running manually/setting development environment

The steps are at the [bottom of this README](https://github.com/diogovalentte/mantium/edit/main/README.md#running-manually).

# Notes:

### iFrame usage

Endpoint:

```
/v1/mangas/iframe
```

> [!NOTE]
> - The endpoint is available via the **API** container, not the dashboard.
>   If you run the API on port `8080` on your server, use your server IP address + port `8080`.
> - The iFrame is fetched directly by the **web browser**, not by your dashboard service.
> - You browser must be able to access the Mantium API.
> - If embedding in an HTTPS dashboard, the API must also be served over HTTPS, or else your browser will block the iFrame.

#### Parameters

- `api_url` (required)
  URL used by the browser to access the API.
- `theme` (optional)
  The iFrame theme. Can be `light` or `dark`.
- `limit` (optional)
  Maximum number of manga to display.
- `showBackgroundErrorWarning` (optional)
  If `true` (*default*), when an error occurs in the background job, a warning will appear in the iFrame.

**Example**:

```
https://mantium-api.domain.com/v1/mangas/iframe?api_url=http://mantium-api.domain.com&theme=dark&limit=5&showBackgroundErrorWarning=false
```

### Security

Mantium does **not** include authentication. Anyone with access to the dashboard or API has full control.

To secure it, place an authentication layer in front of it, such as:
- [Authelia](https://github.com/authelia/authelia)
- [Authentik](https://github.com/goauthentik/authentik)

You can still use the iFrame if the API and dashboard are in the same Docker network. The dashboard will communicate with the API using the API's container name.

### API

Swagger documentation:

```
/v1/swagger/index.html
```

### Source-Specific Notes

**Manga Plus**

Only the first and latest chapters are available in the MangaPlus site. Intermediate chapters may not appear.

I recommend reading the manga in the other source sites, and when you get to the last chapter, remove the manga and add it again from the Manga Plus source.

**KLManga and JManga**

These sources do not provide chapter release timestamps. Mantium sets the release time to the current time when you add the manga and when it detects a newly released chapter.

**MangaUpdates**

Due to MangaUpdates nature:

- Mantium tracks releases, not canonical chapters.
- Sorting is based on upload date.
- Different groups can release the same chapter. Rereleases may appear as the newest chapter.

### Source Downtime

If a source site is temporarily unavailable, Mantium cannot retrieve data. All related operations will fail until the site is accessible again.

### Removed or Changed URLs

If a manga is removed or its URL changes:

- Mantium cannot continue tracking it.
- Delete the entry and add it again using the new URL or a different source.

If a source domain changes (e.g., `klmanga.dm` → `klmanga.io`), open an issue if it has not yet been updated in Mantium.

# Running manually

## Database

Create a `docker-compose.yml`:

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

Start the database service:

```sh
docker compose up -d
```

## API

Export required environment variables below. Optional variables can be found in the `.env.example` file.

```
export POSTGRES_HOST=http://localhost
export POSTGRES_PORT=5432
export POSTGRES_DB=postgres
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export API_PORT=8080
```

Install dependencies:

```bash
go mod download
```

Run:

```bash
go run main.go
```

## Dashboard

If the API is not running on `http://localhost:8080`, set:

```bash
export API_ADDRESS=http://<host>:<port>
```

Install dependencies:

```bash
pip install -r requirements.txt
```

Run:

```bash
streamlit run 01_📖_Dashboard.py
```

Access the dashboard at

```
http://localhost:8501
```
