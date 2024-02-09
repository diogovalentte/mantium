from datetime import datetime

import pytest

from src.api.manga_api import MangaAPIClient

test_mangas = {
    "manga without last read chapter": {
        "manga_url": "https://mangahub.io/manga/death-note_119",
        "manga_status": 1,
        "manga": {
            "ID": 0,
            "Name": "Death Note",
            "Status": 1,
            "Source": "mangahub.io",
            "URL": "https://mangahub.io/manga/death-note_119",
            "CoverImgURL": "https://thumb.mangahub.io/mn/death-note.jpg",
            "CoverImg": "",
            "PreferredGroup": "",
            "LastUploadChapter": {
                "Name": "Chapter 112",
                "Chapter": "112",
                "URL": "https://mangahub.io/chapter/death-note_119/chapter-112",
                "UpdatedAt": datetime(2018, 6, 16, 0, 0, 0),
                "Type": 1,
            },
            "LastReadChapter": None,
        },
    },
    "manga with last read chapter": {
        "manga_url": "https://mangahub.io/manga/vagabond_119",
        "manga_status": 5,
        "last_read_chapter": 200,
        "manga": {
            "ID": 0,
            "Name": "Vagabond",
            "Status": 5,
            "Source": "mangahub.io",
            "URL": "https://mangahub.io/manga/vagabond_119",
            "CoverImgURL": "https://thumb.mangahub.io/mn/vagabond.jpg",
            "CoverImg": "",
            "PreferredGroup": "",
            "LastUploadChapter": {
                "Chapter": "327",
                "Name": "The Man Named Tadaoki",
                "URL": "https://mangahub.io/chapter/vagabond_119/chapter-327",
                "UpdatedAt": datetime(2016, 1, 20, 0, 0, 0),
                "Type": 1,
            },
            "LastReadChapter": {
                "Chapter": "200",
                "Name": "Two Kojiros",
                "URL": "https://mangahub.io/chapter/vagabond_119/chapter-200",
                "UpdatedAt": datetime.today(),
                "Type": 2,
            },
        },
    },
}


@pytest.mark.api_client
class TestMangaAPI:
    def setup_method(self, _):
        self.manga = MangaAPIClient("http://localhost:8080")
        self.update_manga_status_to = 5
        self.update_manga_last_read_chapter_to = 300
        self.updated_manga_last_read_chapter = {
            "Chapter": "300",
            "Name": "The Future of Our Freedom",
            "URL": "https://mangahub.io/chapter/vagabond_119/chapter-300",
            "UpdatedAt": "2016-01-20T00:00:00Z",
            "Type": 2,
        }

    def test_add_manga_without_last_read_chapter(self):
        manga_test = test_mangas["manga without last read chapter"]
        res = self.manga.add_manga(manga_test["manga_url"], manga_test["manga_status"])
        assert res["message"] == "Manga added successfully"

    def test_add_manga_with_last_read_chapter(self):
        manga_test = test_mangas["manga with last read chapter"]
        res = self.manga.add_manga(
            manga_test["manga_url"],
            manga_test["manga_status"],
            manga_test["last_read_chapter"],
        )
        assert res["message"] == "Manga added successfully"

    def test_update_manga_status(self):
        manga_test = test_mangas["manga without last read chapter"]
        res = self.manga.update_manga_status(
            status=self.update_manga_status_to, manga_url=manga_test["manga_url"]
        )
        assert res["message"] == "Manga status updated successfully"

    def test_update_manga_last_read_chapter(self):
        manga_test = test_mangas["manga with last read chapter"]
        res = self.manga.update_manga_last_read_chapter(
            chapter=self.update_manga_last_read_chapter_to,
            manga_url=manga_test["manga_url"],
        )
        assert res["message"] == "Manga last read chapter updated successfully"

    def test_get_manga_without_last_read_chapter(self):
        manga_test = test_mangas["manga without last read chapter"]
        manga = self.manga.get_manga(manga_url=manga_test["manga_url"])
        manga["CoverImg"] = ""
        manga["ID"] = 0
        manga_test["manga"]["Status"] = self.update_manga_status_to
        assert manga == manga_test["manga"]

    def test_get_manga_with_last_read_chapter(self):
        manga_test = test_mangas["manga with last read chapter"]
        manga = self.manga.get_manga(manga_url=manga_test["manga_url"])
        manga["CoverImg"] = ""
        manga["ID"] = 0
        manga_test["manga"]["LastReadChapter"] = self.updated_manga_last_read_chapter
        # set UpdatedAt to today's date
        manga_test["manga"]["LastReadChapter"]["UpdatedAt"] = datetime.combine(
            datetime.today().date(), datetime.min.time()
        )
        assert manga == manga_test["manga"]

    def test_get_mangas(self):
        mangas = self.manga.get_mangas()
        # hardcoded mangas length
        assert len(mangas) == 2

    def test_delete_manga_without_last_read_chapter(self):
        manga = test_mangas["manga without last read chapter"]
        res = self.manga.delete_manga(manga_url=manga["manga_url"])
        assert res["message"] == "Manga deleted successfully"

    def test_delete_manga_with_last_read_chapter(self):
        manga = test_mangas["manga with last read chapter"]
        res = self.manga.delete_manga(manga_url=manga["manga_url"])
        assert res["message"] == "Manga deleted successfully"
