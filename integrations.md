

# Kaizoku
Using environment variables, you can enable the [Kaizoku](https://github.com/oae/kaizoku) integration. The integration will:

- Try to add the manga to Kaizoku when you add it to the dashboard.
- If the background job to update the mangas metadata detects newly released chapters, it will trigger Kaizoku to check and download new chapters.
- If there are already mangas on your dashboard, the API has a route to add the mangas on your dashboard to Kaizoku. To know more, check the [API docs](https://github.com/diogovalentte/mantium?tab=readme-ov-file#api).

## Limitations

#### Mangal sources
Kaizoku uses [Mangal](https://github.com/metafates/mangal) under the hood to download the chapters. Mangal can only download from configured sources, like the built-in Mangadex source.
> There is no built-in source for ComicK, but you can add a custom one. Download the Lua script `defaults/ComicK.lua` in this repository and add it to the folder `/config/.config/mangal/sources` of your Kaizoku Docker container.
> - This source downloads unique chapters. For example, if group A uploads chapter 110 of a manga and then group B uploads the same chapter, this source will download the chapter from group B instead of downloading both chapters, as group B's chapter was the last one uploaded.

> There is also the file `defaults/MangaDex.lua` file in this repo. It's an alternative to the Kaizoku MangaDex source where the only difference is that it also downloads unique chapters and has a fix that allows downloading manga categorized as pornographic. If you want to use it, delete the source file `MangaDex.lua` in your Kaizoku container and add the one to this repository. After this, execute `sudo chattr +i <path to MangaDex.lua file>` so Kaizoku doesn't overwrite this file after the container's restart.
> - After adding the file, restart the container and execute `mangal clear -c` inside the container to clear the mangal cache.

> Some site sources that Mantium supports don't have a built-in or custom source, the integration will not work with mangas tracked in them.
> - These sources are Manga Plus and MangaHub.

> If you set the environment variable `KAIZOKU_TRY_OTHER_SOURCES` to true, the Mantium will try to add the manga to Kaizoku using other sources if the source fails instead of just returning an error.
> - For example, if you set it to true and try to add a manga from Manga Plus, Mantium will try to add the manga from other sources, like the Mangadex built-in source. This way, you can track the manga from Manga Plus and still have Kaizoku downloading the chapters but from other sources.
>   - **Note**: using the example above, if Mantium detects a new chapter in Manga Plus, it'll trigger Kaizoku to check for new chapters in the Mangedex source, but maybe the newly released chapter is not available in Mangadex yet, so Kaizoku will not download the chapter.

#### Anilist Page
Kaizoku only allows the addition of mangas that have a page/ID on [Anilist](https://anilist.co/search/manga). Mangal will search Anilist for manga with the same name as your manga, if it doesn't find any, Kaizoku will not add the manga. If it finds at least one Anilist page, Kaizoku will add the manga, but sometimes Kaizoku gets the wrong Anilist ID for the manga.

> If multiple mangas have the same name, Kaizoku will use the Anilist ID of the oldest manga. In this case, you must manually set the right Anilist ID on Kaizoku.

> If the manga isn't on Anilist, but there is another manga with the same name in Anilist, Kaizoku will use this one. In this case, you can't manually set the right Anilist ID on Kaizoku, as there is none. You can delete the manga from Kaizoku, add the correct manga to Anilist and set the right Anilist ID on Kaizoku, or live with the wrong Anilist ID for the manga. Kaizoku will download the correct chapters anyway, only the metadata that Kaizoku and readers like [Kavita](https://github.com/Kareadita/Kavita) use will be wrong.

In my case, I tried to add 96 mangas to Kaizoku, 10 were not added because they didn't have an Anilist ID. 3 had the wrong Anilist ID, from these, 2 I just needed to set the right Anilist ID, but one of them didn't have a page on Anilist, so I can't change it to the right Anilist ID unless the correct manga is added to Anilist.

#### Kaizoku jobs queue
> If the background job to update the mangas metadata detects newly released chapters, it will trigger Kaizoku to check and download new chapters.

To trigger Kaizoku to check and download new chapters, Mantium has to do some things:
1. Wait until the Kaizoku jobs queues are empty.
2. When empty, Mantium will add all mangas on Kaizoku to the `checkOutOfSyncChaptersQueue` queue. This will make Kaizoku check for new chapters of your entire Kaizoku library, each manga will be a job in the queue. Unfortunately, there is no way of adding a specific manga to the queue (at least I didn't find an option to).
3. Mantium will wait for Kaizoku to check out-of-sync chapters of all jobs in the queue. When done, this means the `checkOutOfSyncChaptersQueue` queue is empty, and Kaizoku knows the manga with new chapters.
4. Mantium will then trigger Kaizoku to add the manga to the `fixOutOfSyncChaptersQueue` queue. Each manga is a job in this queue and Kaizoku will download the new chapters of each job. Mantium will then wait until Kaizoku finishes this queue.
5. When finished, the queue will be empty, and then Mantium will trigger Kaizoku to try to download again the chapters that caused an error when downloading from the previous step. Mantium will not wait until Kaizoku finishes it, it'll continue its execution and mark the job as done.

The process of waiting until Kaizoku finishes the jobs in the queues has a timeout. This means Mantium will not wait forever for Kaizoku, it'll timeout and return an error indicating the timeout.

But, the more mangas you have in Kaizoku, the longer Kaizoku will take to empty its queues. I have 100 mangas and it takes +- 3 minutes. You can adjust how much time Mantium should wait for Kaizoku before timeout if you're seeing too many errors due to timeout.

You can do this by setting the `KAIZOKU_WAIT_UNTIL_EMPTY_QUEUES_TIMEOUT_MINUTES` environment variable to the number of minutes that Mantium should wait, like `KAIZOKU_WAIT_UNTIL_EMPTY_QUEUES_TIMEOUT_MINUTES=10` to wait for 10 minutes each of the `checkOutOfSyncChaptersQueue` and `fixOutOfSyncChaptersQueue` queues.

# Tranga
Using environment variables, you can enable the [Tranga](https://github.com/c9glax/tranga) integration. The integration will:

- Try to add the manga to Tranga when you add it to the dashboard.
- If the background job to update the mangas metadata detects newly released chapters, it will trigger Tranga to check and download new chapters of the mangas that have new chapters.
- If there are already mangas on your dashboard, the API has a route to add the mangas on your dashboard to Tranga. To know more, check the [API docs](https://github.com/diogovalentte/mantium?tab=readme-ov-file#api).

## Limitations
Tranga can download chapters from mangas from many sources (called connectors in Tranga). They are like Mantium source sites, like MangaDex and ComicK. Currently, the only connector/source site that Mantium and Tranga both share is MangaDex. This means that only mangas from MangaDex will work with this integration.

# Ntfy
Mantium can send a notification to a Ntfy topic when a new chapter from a manga with the status "reading" or "completed" is released. To activate the Ntfy integration, use the environment variables in the [docker-compose.yml](https://github.com/diogovalentte/mantium/blob/main/docker-compose.yml) file or in the [.env.examples](https://github.com/diogovalentte/mantium/blob/main/.env.example) file.
