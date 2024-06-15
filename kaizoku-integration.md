You can enable the [Kaizoku](https://github.com/oae/kaizoku) integration using environment variables. The integration will:

- Try to add the manga to Kaizoku when you add it to the dashboard.
- If the background job to update the mangas metadata detects newly released chapters, it will add a job to the Kaizoku queue to check all your Kaizoku mangas and download the new chapters.
- If there are already mangas on your dashboard, the API has a route to add the mangas on your dashboard to Kaizoku. Check the API docs.

# Limitations

### Mangal sources
Kaizoku uses [Mangal](https://github.com/metafates/mangal) under the hood to download the chapters. Mangal can only download from configured sources, like the built-in Mangadex source.
- There is no built-in source for ComicK, but you can add a custom one. Download the Lua script `api/defaults/ComicK.lua` in this repository, add it to the folder `/config/.config/mangal/sources` of your Kaizoku Docker container, and restart it.
  - This source downloads only unique chapters. For example, if group A uploads chapter 110 of a manga and then group B uploads the same chapter, this source will download the chapter from group B instead of downloading both chapters.
    - There is also the file `api/defaults/MangaDex.lua` file in this repo. It's an alternative to the Kaizoku MangaDex source where the only difference is that it also downloads only unique chapters. If you want to use it, delete the source file `MangaDex.lua` in your Kaizoku container and add the one in this repository.
- Some sources don't have a built-in or custom source, the integration will not work with mangas tracked in them. These sources are Manga Plus and MangaHub.
- If you set the environment variable `KAIZOKU_TRY_OTHER_SOURCES` to true, the application will try to add the manga to Kaizoku using other sources if the original source fails instead of just returning an error.
  - For example, if you set it to true and try to add a manga from Manga Plus, Mantium will try to add the manga from other sources, like the Mangadex built-in source. This way, you can track the manga from Manga Plus and still have Kaizoku downloading the chapters.
    - **Note**: using the example above, if Mantium detects a new chapter in Manga Plus, it'll trigger Kaizoku to check for new chapters in the Mangedex source, and maybe the newly released chapter is not available in Mangadex yet, so it'll not download the chapter.

### Mangas Anilist page
Kaizoku only downloads chapters of mangas that have a page on [Anilist](https://anilist.co/search/manga). Mangal will find Anilist mangas with the same name as your manga, if it doesn't find any, Kaizoku will not add your manga.
- Sometimes Kaizoku gets the wrong Anilist ID for the manga.
  - If multiple mangas have the same name, Kaizoku will use the Anilist ID of the oldest manga. In this case, you must manually set the right Anilist ID on Kaizoku.
  - If the manga doesn't have a page on Anilist, but there is another manga with the same name in Anilist, Kaizoku will use this Anilist manga ID. In this case, you can only delete the manga from Kaizoku, or add the correct manga to Anilist and set the right Anilist ID on Kaizoku.
- In my case, I tried to add 96 mangas to Kaizoku, 10 were not added because they didn't have an Anilist ID. 3 had the wrong Anilist ID, from these, 2 I just needed to set the right Anilist ID, but one of them didn't have a page on Anilist, so I can't change it to the right Anilist ID unless the correct manga is added to Anilist.

### Kaizoku jobs queue
Mantium waits until the Kaizoku jobs queues are empty before adding new jobs to download newly released chapters. But as it can take a long time, the API will timeout after some minutes of waiting. You can set the number of minutes the API will wait using the environment variable `KAIZOKU_WAIT_UNTIL_EMPTY_QUEUES_TIMEOUT_MINUTES`.
- The more mangas you have in Kaizoku, the longer it'll take to empty its job queues. I have 100 mangas and it takes +- 3 minutes.
