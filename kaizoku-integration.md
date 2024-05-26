You can enable the [Kaizoku](https://github.com/oae/kaizoku) integration using environment variables. The integration will:

- Try to add the manga to Kaizoku when you add it to the dashboard.
- If the background job to update the mangas metadata detects newly released chapters, it will add a job to the Kaizoku queue to check all your Kaizoku mangas and download the new chapters.
- If there are already mangas on your dashboard, the API has a route to add the mangas on your dashboard to Kaizoku. Check the API docs.

# Limitations

- Kaizoku uses [Mangal](https://github.com/metafates/mangal) under the hood to download the chapters. Mangal can only download from configured sources, like the built-in Mangadex source.
  - There is no built-in source for ComicK, but you can add a custom one. Download the Lua script `api/defaults/ComicK.lua` in this repository and add it to the folder `/config/.config/mangal/sources` of your Kaizoku Docker container and restart it.
  - There is no built-in or custom source for Mangahub, so the mangas from Mangahub in your dashboard will not work with Kaizoku.
- Kaizoku only download chapters of mangas that have a page on [Anilist](https://anilist.co/search/manga). Mangal will find Anilist mangas with the same name as your manga, if it doesn't find any, Kaizoku will not add your manga.
  - Sometimes Kaizoku gets the wrong Anilist ID for the manga.
    - If multiple mangas have the same name, Kaizoku will use the Anilist ID of the oldest manga. In this case, you must manually set the right Anilist ID on Kaizoku.
    - If the manga doesn't have a page on Anilist, but there is another manga with the same name in Anilist, Kaizoku will use this Anilist manga ID. In this case, you can only delete the manga from Kaizoku, or add the correct manga to Anilist and set the right Anilist ID on Kaizoku.
  - In my case, I tried to add 96 mangas to Kaizoku, 10 were not added because they didn't have an Anilist ID. 3 had the wrong Anilist ID, from these, 2 I just needed to set the right Anilist ID, but one of them didn't have a page on Anilist, so I can't change it to the right Anilist ID unless the correct manga is added to Anilist.
- The API waits until the Kaizoku job queues are empty to add jobs to download new chapters. But as it can take a long time, the API will timeout after some minutes waiting. You can set the number minutes the API will wait using an environment variable.
  - The more mangas you have in Kaizoku, the longer it'll take to empty its job queues. I have 100 mangas and it takes +- 3 minutes.
