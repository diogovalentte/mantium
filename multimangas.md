Imagine the same manga is released in the comick and mangadex sources, but sometimes the newest chapter is released first in comick and other times in mangadex. You may want to track the manga in both sources so that you are notified whenever a new chapter is released in both sources.

But adding the same manga from different sources makes them act as completely different mangas. You would need to set the last read chapters for all mangas manually and you'll be notified for all mangas, like in the image below:

![image](https://github.com/user-attachments/assets/947cd396-4ed4-4043-84c1-376852dffce2)

The **multimanga** feature solves this issue. With it, you can track the same manga from multiple sources, and they will be treated as the same manga! No multiple notifications or setting the last read chapter for all of them!

# How does it work?
The multimanga feature is like a facade for multiple mangas. You add the same manga from different sources to the multimanga, but you'll not directly interact with them, you'll interact with the multimanga.

For example, when you set the last read chapter, you're setting the multimanga's last read chapter. When you edit the cover image, you're editing the multimanga's cover image.

## Current manga
A multimanga always has a manga called **current manga**, which is one of the multimanga's mangas. Mantium decides which manga should be the current manga whenever you **add/remove** a manga from a multimanga, and in the [periodic job that updates the mangas in the background](https://github.com/diogovalentte/mantium?tab=readme-ov-file#check-manga-updates-and-notify).

Mantium decides which manga should be the current manga based on the mangas' last released chapter. Mantium **tries** to set the current manga to the manga that has the newest released chapter.

In the dashboard/iframe, the current manga's name, cover image, and last released chapter's link are shown. When you click to select the last read chapter, the current manga's chapters list is shown, as it has the newest chapter. When you're notified of a new chapter, the current manga's newest chapter is sent in the notification.

[Here is a demo video of the multimanga feature](https://imgur.com/a/Ev7hcLK). It shows:

1. Turning a Mangaplus manga into a multimanga. The Mangaplus manga becomes the **current manga**. It's possible to set the last read chapters to one of the chapters from the current manga.
    - MangaPlus makes available only the 3 first and last chapters, so **only 6 chapters are shown in the chapters list**.
2. The multimanga is highlighted and **the same manga but from the ComicK source is added to the multimanga**.
3. As the ComicK manga's last released chapter is the same as the Mangaplus one, but was released later, **the ComicK manga becomes the current manga**.
4. The dashboard now shows the **ComicK manga's name, cover image, and last released chapter link**.
5. When clicking to select the last read chapter, the **ComicK manga's chapters list is shown**. The ComicK source makes all chapters available.
6. Then, the ComicK manga, currently the current manga, is removed from the multimanga, and the **Mangaplus manga becomes the current manga**.
