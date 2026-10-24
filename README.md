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
- [KLManga](https://klmanga.zone/)
- [JManga](https://jmanga.locker)

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
  <img src="https://github.com/user-attachments/assets/404a5108-52d2-43a7-9bf3-81fa6721409a">
</p>

## List View

<p align="center">
  <img src="https://github.com/user-attachments/assets/93308cdb-93c0-420b-a32e-def5760b8671">
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
| ![](https://github.com/user-attachments/assets/caf8c6f9-c7e3-42f1-bfa3-a42dfbfe80c1) | ![](https://github.com/user-attachments/assets/abd9027f-bd4b-4b4c-983d-fc11bb9e2d01) | ![](https://github.com/user-attachments/assets/9c36585d-4ecb-4cb0-b04f-3eb701529dcf) |

---

# Custom Manga

Custom Manga allows you to track content not supported natively (manga, manhwa, light novels, etc.).

You must manually provide:

- Name
- URL (optional): the manga page, used as the manga's link
- Cover image (or use default). PNG, JPEG or WebP, up to 10 MB and 40 megapixels
- Chapter selectors (optional)

## Chapter Selectors

Custom manga can automatically detect new chapters by getting a web page and applying selectors to its chapter list to get the chapter number and URL. The page is the one in the **Check URL** field, or the manga URL if it's empty:

- **Check URL (optional)**:
  The page the selectors are applied to, when it's not the manga page, like an RSS feed, an API or a chapters list. The manga URL is still the manga's link. The presets are also tested against this page. A custom manga can track chapters with only the check URL, without a manga URL.
- **Selector**:
  CSS or XPath selector for the chapters in the page's chapter list. If it matches more than one element, the last one is taken as the newest chapter, or the first one if **Newest Chapter Is First** is enabled. A selector that matches only the newest chapter also works.
  Add the prefix `css:` for CSS selectors, and `xpath:` for XPATH selectors (ex: `css:div.chapter-list > a span`). XPATH selectors also work with XML pages.
- **Attribute (optional)**:
  Attribute containing the chapter name or URL (e.g., href).
  If empty, the element text is used.
- **Regex (optional)**:
  Extracts the chapter number from the selected value. The chapter is the text captured by the regex groups (the parts in parentheses). Groups that didn't match are skipped, and when more than one matched, they're joined with a space. For example:
  - `Chapter (\d+)` gets `100` from `Chapter 100`.
  - `Chapter (\d+(?:\.\d+)?)` gets `12.5` from `Chapter 12.5`. Groups inside another group, like `(\d+(\.\d+)?)`, and non-capturing groups, `(?:...)`, are part of the group around them, not separate parts.
  - `Ch\.(\d+)|Chapter (\d+)` works with both `Ch.7` and `Chapter 7`.
  - `CHAPTER(\p{Nd}+).*?(\p{Nd}+)` gets `４ ８` from `CHAPTER４［ BLIND JUSTICE ］８（後編）`, a chapter in parts.

  Regexes use [Go's syntax](https://github.com/google/re2/wiki/Syntax), where `\d` only matches `0-9`. Use `\p{Nd}` to also match digits like the full-width `４`. A chapter with more than one part, like `4 8`, isn't a number: Mantium shows it as unread when it's different from the last read chapter, and in a multimanga, compares it with the other sources by the date it was found.
- **Newest Chapter Is First (optional)**:
  Enable it when the site lists chapters from newest to oldest.
- **Use Browser (optional)**:
  Uses a headless browser if the page requires JavaScript rendering. Mantium waits up to 15 seconds for each selector's element to appear in the page.
  Otherwise, a simple HTTP GET request is used.

The page is fetched once, and both selectors are applied to it.

A chapter found only by its name, without a chapter URL selector, has no URL. In the dashboard and in the iFrame, it links to the manga URL, or shows only its name if the manga has no URL either.

When a custom manga is the current manga of a multimanga, its selectors also get the chapters list shown in the multimanga's **Last Read Chapter** field, so the last read chapter can be picked from it:

- It needs a chapter name selector. If there's also a chapter URL selector, both must match the same number of elements, so the nth URL is the nth chapter's, and **Newest Chapter Is First** must be the same in both.
- Elements the chapter name regex doesn't match, like announcements in a feed, are left out of the list.
- If the list can't be fetched, the multimanga's mangas that aren't custom mangas are tried until one has chapters, and a warning tells what happened.

Use the **Test selectors** button in the add and edit forms to see which chapter the selectors get and how many chapters the chapters list has, without saving anything. If a selector, attribute or regex doesn't work, the message tells which one and why.

> [!NOTE]
> If the URL selector returns a relative URL, Mantium resolves it the same way a browser does: against the page's `<base>` element if it has one, else against the URL of the page the selectors are applied to (the check URL, or else the manga URL, after redirects). For example, if that page is `https://example.com/title/one-piece/`, the URL selector returning `/chapter1000` becomes `https://example.com/chapter1000`, and returning `chapter1000` becomes `https://example.com/title/one-piece/chapter1000`.

<img width="855" height="1335" alt="image" src="https://github.com/user-attachments/assets/4e844149-964e-463f-821f-f59a45c9a644" />

## Private Addresses

By default, the API refuses to get custom manga pages and cover images from private addresses, like `localhost`, `192.168.x.x`, `10.x.x.x` or `169.254.169.254`. This includes redirects and every request made by the browser when **Use Browser** is enabled. It prevents a URL from making the API reach services in your network that aren't meant to be reachable from outside.

If you track a site hosted in your own network, set the API environment variable `ALLOW_PRIVATE_URLS=true`.

> [!WARNING]
> Custom mangas whose URL or check URL points to your network don't update until you set `ALLOW_PRIVATE_URLS=true`.

---

# Background Updates and Notifications

Mantium can periodically:

- Refresh manga metadata from the source sites.
- Detect new chapters.
- Send notifications for manga with status reading or completed.

If an error occurs during background processing:

- A warning is displayed on the dashboard and iFrame.
- This warning can be disabled.
- You can set the minimum number of consecutive errors that must happen to show the warnings.

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

If a source domain changes (e.g., `klmanga.town` → `klmanga.zone`), Mantium usually finds the new one by itself: on startup and before every update of all mangas, it follows the redirects of the KLManga, JManga and RawKuma domains it knows, and moves the URLs of your mangas to the domain they end up at. If it doesn't (the old domain stopped redirecting, for example), open an issue.

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
