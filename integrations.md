> Integrations are optional and disabled by default. They can be enabled using environment variables. See the [.env.example](https://github.com/diogovalentte/mantium/blob/main/.env.example) file for configuration details.

# Ntfy

Mantium can publish a notification to an ntfy topic when a new chapter is released for a manga whose status is **reading** or **completed**.

---

# Tranga

The [Tranga](https://github.com/c9glax/tranga) integration:

- Attempts to add a manga to Tranga when it is added to the Mantium dashboard.
  - By default, only the original manga of a Multimanga is added. Entries added later to the same Multimanga are not automatically added. It can be changed in the dashboard settings.
- When the background metadata update detects new chapters, Mantium triggers Tranga to check and download them.
- Existing dashboard entries can be imported into Tranga through an API route (see the [API docs](https://github.com/diogovalentte/mantium?tab=readme-ov-file#api)).

## Limitations

Tranga downloads chapters through connectors (equivalent to Mantium source sites). Currently, the only shared source between Mantium and Tranga is MangaDex, so this integration works only for MangaDex entries.

---

# Suwayomi

The [Suwayomi](https://github.com/Suwayomi) integration:

- Attempts to add a manga to Suwayomi when it is added to the dashboard.
  - By default, only the original manga of a Multimanga is added. Entries added later to the same Multimanga are not automatically added. It can be changed in the dashboard settings.
- By default, all chapters are queued for download when the manga is added.
  - This behavior can be disabled in the dashboard settings.
- When the background update detects a new chapter, Mantium queues that chapter for download.
- Existing dashboard entries can be imported into Tranga through an API route (see the [API docs](https://github.com/diogovalentte/mantium?tab=readme-ov-file#api)).

## Required extension repositories and sources

Mantium expects the following extension repositories to be installed on your Suwayomi server:

- https://github.com/keiyoushi/extensions
- https://github.com/zosetsu-repo/tachi-repo

It also expects the following sources to be installed:

- MANGA Plus by SHUEISHA
- MangaDex
- MangaHub

Currently, only these sources are supported because compatible extensions for the other Mantium sources are not available. If you discover a Suwayomi source compatible with Mantium sources, feel free to open an issue about it.

---

# Kaizoku

The [Kaizoku](https://github.com/oae/kaizoku) integration:

- Attempts to add a manga to Suwayomi when it is added to the dashboard.
  - By default, only the original manga of a Multimanga is added. Entries added later to the same Multimanga are not automatically added. It can be changed in the dashboard settings.
- When the background update detects a new chapter, Mantium queues that chapter for download.
- Existing dashboard entries can be imported into Tranga through an API route (see the [API docs](https://github.com/diogovalentte/mantium?tab=readme-ov-file#api)).

## Limitations

#### Mangal sources

Kaizoku uses [Mangal](https://github.com/metafates/mangal) under the hood to download chapters. Mangal can download only from configured sources, and the default MangaDex source in Kaizoku no longer works reliably.

This repository provides updated source files in the `defaults/` directory for:

- MangaDex
- MangaHub
- RawKuma
- KLManga
- JManga

You should copy these files into the Kaizoku container:

```
/config/.config/mangal/sources
```

Then restart the container.

**MangaDex source replacement**

If you want to use MangaDex:

1. Remove the existing `MangaDex.lua` inside the container.
2. Replace it with the file from this repository.
3. Prevent Kaizoku from overwriting it:

```
sudo chattr +i <path to MangaDex.lua>
```

To allow editing or removal later:

```
sudo chattr -i <path to MangaDex.lua>
```

**Alternative mangal build**

The original mangal binary cannot properly download from some sources (KLManga and MangaHub). A patched version is available:

- Fork: https://github.com/diogovalentte/mangal
- Kaizoku image using the fork: ghcr.io/diogovalentte/kaizoku

Using this image is recommended if you intend to download from these sources. The image contains only the forked mangal binary, not the custom source files.

`KAIZOKU_TRY_OTHER_SOURCES`

If `KAIZOKU_TRY_OTHER_SOURCES=true`, Mantium will attempt to add the manga using a different source if the original source fails.

Example:

- You track a manga from Manga Plus.
- Kaizoku cannot add it from Manga Plus.
- Mantium attempts to add it using MangaDex instead.

> [!NOTE]
If a new chapter appears on Manga Plus, Mantium will trigger Kaizoku to check and download new chapter for the MangaDex source. If is not yet available on MangaDex, Kaizoku will not download it.

**Anilist**

When Mantium adds a manga to Kaizoku, it sends the manga name and source. Kaizoku searches for the manga’s AniList ID and adds it if found.

Possible outcomes:

- No AniList entry → manga is not added.
- Wrong match → incorrect manga metadata in Kaizoku, but the correct chapters are still downloaded.

The AniList matching process is handled entirely by Kaizoku, but, if you use my custom Kaizoku image, Kaizoku will add the manga even if it doesn't find the AniList ID.

**Kaizoku job queue and timeouts**

When new chapters are detected:

1. Mantium requests Kaizoku to update.
2. Kaizoku schedules jobs for all tracked manga in a **queue**.
3. Mantium waits for completion.

If the queue takes too long, Mantium times out and returns an error (*Kaizoku will continue to process the queue in background*).

Default timeout: **5 minutes**

You can change it using:

```
KAIZOKU_WAIT_UNTIL_EMPTY_QUEUES_TIMEOUT_MINUTES
```

Set the value to the number of minutes Mantium should wait before timing out.
