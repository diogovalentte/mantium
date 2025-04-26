# Ntfy

Mantium can notify a Ntfy topic when a new chapter from a manga with the status "reading" or "completed" is released. To activate the Ntfy integration, use the environment variables in the [docker-compose.yml](https://github.com/diogovalentte/mantium/blob/main/docker-compose.yml) file or in the [.env.examples](https://github.com/diogovalentte/mantium/blob/main/.env.example) file.

# Tranga

You can enable the [Tranga](https://github.com/c9glax/tranga) integration using environment variables. The integration will:

- Try to add the manga to Tranga when you add it to the dashboard.
  - Only the original manga of a multimanga will be added to Tranga. Mangas added to a multimanga later are not automatically added to Tranga.
- If the background job to update the mangas metadata detects newly released chapters, it will trigger Tranga to check and download new chapters.
- If there are already mangas on your dashboard, the API has a route to add your mangas to Tranga. To know more, check the [API docs](https://github.com/diogovalentte/mantium?tab=readme-ov-file#api).

## Limitations

Tranga can download chapters from mangas from many sources (called connectors in Tranga). They are like Mantium source sites (*MangaDex, ComicK, etc.*). Currently, the only connector/source site that Mantium and Tranga both share is MangaDex. This means that only mangas from MangaDex will work with this integration.

# Suwayomi

You can enable the [Suwayomi](https://github.com/Suwayomi) integration using environment variables. The integration will:

- Try to add the manga to Suwayomi when you add it to the dashboard.
  - Only the original manga of a multimanga will be added to Suwayomi. Mangas added to a multimanga later are not automatically added to Suwayomi.
- If there are already mangas on your dashboard, the API has a route to add your mangas to Suwayomi. To know more, check the [API docs](https://github.com/diogovalentte/mantium?tab=readme-ov-file#api).

## Extensions repositories and sources
Mantium expects that you installed both the [keiyoushi](https://github.com/keiyoushi/extensions) and [zosetsu-repo](https://github.com/zosetsu-repo/tachi-repo) extension repositories on your Suwayomi server.

It also expects that you have installed the ComicK, MANGA Plus by SHUEISHA, MangaDex, and MangaHub sources on your server. The integration will only work with these sources, as I didn't find Suwayomi-compatible sources for the other Mantium sources.

# Kaizoku

You can enable the [Kaizoku](https://github.com/oae/kaizoku) integration using environment variables. The integration will:

- Try to add the manga to Kaizoku when you add it to the dashboard.
  - Only the original manga of a multimanga will be added to Kaizoku. Mangas added to a multimanga later are not automatically added to Kaizoku.
- If the background job to update the mangas metadata detects newly released chapters, it will trigger Kaizoku to check and download new chapters.
- If there are already mangas on your dashboard, the API has a route to add your mangas to Kaizoku. To know more, check the [API docs](https://github.com/diogovalentte/mantium?tab=readme-ov-file#api).

## Limitations

#### Mangal sources

Kaizoku uses [Mangal](https://github.com/metafates/mangal) under the hood to download the chapters. Mangal can only download from configured sources, but from the sources that Mantium supports, only the mangadex source is configured by default in Kaizoku, and this source doesn't work properly anymore.

In this repository, there are source files for the [MangaDex](https://github.com/diogovalentte/mantium/blob/main/defaults/MangaDex.lua), [ComicK](https://github.com/diogovalentte/mantium/blob/main/defaults/ComicK.lua), [MangaHub](https://github.com/diogovalentte/mantium/blob/main/defaults/MangaHub.lua), [RawKuma](https://github.com/diogovalentte/mantium/blob/main/defaults/RawKuma.lua), [KLManga](https://github.com/diogovalentte/mantium/blob/main/defaults/KLManga.lua), and [JManga](https://github.com/diogovalentte/mantium/blob/main/defaults/JManga.lua) in the `defaults/` folder. I recommend downloading these files and adding them to Kaizoku, this way, you will be able to download chapters from these sources.

> After downloading the files, add them to the folder `/config/.config/mangal/sources` of your Kaizoku Docker container, and restart it.
>
> - If you want to use the MangaDex source, delete the source file `MangaDex.lua` in your Kaizoku container and add the one from this repository. After this, execute `sudo chattr +i <path to MangaDex.lua file>` so Kaizoku doesn't overwrite this file after the container's restart. You also will not be able to edit/delete the file after executing the command. To be able to edit/delete again, execute `sudo chattr -i <path to MangaDex.lua file>`.
> - The KLManga source sometimes can't download some chapters when using the original `mangal` binary that Kaizoku uses, and the MangaHub source doesn't work at all. To address this, I created a [fork of mangal](https://github.com/diogovalentte/mangal), allowing it to download from these sources, and I also published a [Kaizoku docker image](https://github.com/diogovalentte/kaizoku-custom-mangal) that uses this forked version of mangal under `ghrc.io/diogovalentte/kaizoku-custom-mangal`. I recommend using this Docker image to download from these sources.

> The Manga Plus and MangaUpdates source sites don't have a built-in or custom source; the integration will not work with mangas tracked in them.

> If you set the environment variable `KAIZOKU_TRY_OTHER_SOURCES` to `true`, Mantium will try to add the manga to Kaizoku using other sources if the manga's source fails instead of just returning an error.
>
> - For example, if you set it to true and try to add a manga from Manga Plus to Kaizoku, it'll fail, and Mantium will try to add the manga from other sources, like the Mangadex source. This way, you can track the manga from Manga Plus and still have Kaizoku downloading the chapters, but from other sources.
>   - **Note**: using the example above, if Mantium detects a new chapter in Manga Plus, it'll trigger Kaizoku to check for new chapters in the Mangadex source, but maybe the newly released chapter is not available in Mangadex yet, so Kaizoku will not download the chapter.

#### Anilist

When Mantium tries to add a manga to Kaizoku, it just sends a request with the manga name and the source to Kaizoku, and then Kaizoku searches for the manga [Anilist](https://anilist.co/search/manga) ID. If it finds it, it'll add the manga and start downloading chapters. If it doesn't find it, it'll not add the manga to the library. The process of searching the manga's Anilist ID is Kaizoku's job. Sometimes Kaizoku will get the wrong Anilist ID.

In my case, I tried to add 96 mangas to Kaizoku, 10 were not added because they didn't have an Anilist ID. 3 had the wrong Anilist ID, and from these, 2 I just needed to set the right Anilist ID, but one of them didn't have a page on Anilist, so I can't change it to the right Anilist ID unless the correct manga is added to Anilist.

#### Kaizoku jobs queue

If the background job to update the mangas metadata detects newly released chapters, it will trigger Kaizoku to check and download new chapters. Kaizoku will add all mangas to the queues as jobs and process each job. The more mangas you have in Kaizoku, the more time these jobs will take. Mantium will wait for the jobs to be processed, but not forever. Mantium will timeout and return an error indicating the timeout. By default, Mantium will wait for 5 minutes, but you can change it using the environment variable `KAIZOKU_WAIT_UNTIL_EMPTY_QUEUES_TIMEOUT_MINUTES` and setting it to the number of minutes Mantium should wait.
