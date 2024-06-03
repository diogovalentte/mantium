from datetime import datetime

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
            "CoverImgURL": "https://thumb.mghcdn.com/mn/death-note.jpg",
            "CoverImg": "",
            "CoverImgFixed": False,
            "CoverImgResized": True,
            "PreferredGroup": "",
            "LastReleasedChapter": {
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
        "last_read_chapter": "200",
        "manga": {
            "ID": 0,
            "Name": "Vagabond",
            "Status": 5,
            "Source": "mangahub.io",
            "URL": "https://mangahub.io/manga/vagabond_119",
            "CoverImgURL": "https://thumb.mghcdn.com/mn/vagabond.jpg",
            "CoverImg": "",
            "CoverImgFixed": False,
            "CoverImgResized": True,
            "PreferredGroup": "",
            "LastReleasedChapter": {
                "Name": "The Man Named Tadaoki",
                "Chapter": "327",
                "URL": "https://mangahub.io/chapter/vagabond_119/chapter-327",
                "UpdatedAt": datetime(2016, 1, 20, 0, 0, 0),
                "Type": 1,
            },
            "LastReadChapter": {
                "Name": "The Man Named Tadaoki",
                "Chapter": "327",
                "URL": "https://mangahub.io/chapter/vagabond_119/chapter-327",
                "UpdatedAt": datetime.today(),
                "Type": 2,
            },
        },
    },
}


class TestMangaAPI:
    def setup_method(self, _):
        self.manga = MangaAPIClient("http://localhost:8080")
        self.update_manga_status_to = 5
        self.update_manga_last_read_chapter_to = "300"
        self.updated_manga_last_read_chapter = {
            "Chapter": "300",
            "Name": "The Future of Our Freedom",
            "URL": "https://mangahub.io/chapter/vagabond_119/chapter-300",
            "UpdatedAt": "2016-01-20T00:00:00Z",
            "Type": 2,
        }
        self.update_manga_cover_img_to = "https://thumb.mghcdn.com/mn/vagabond.jpg"

    def test_add_manga_without_last_read_chapter(self):
        manga_test = test_mangas["manga without last read chapter"]
        res = self.manga.add_manga(
            manga_test["manga_url"], manga_test["manga_status"], "", ""
        )
        assert res["message"] == "Manga added successfully"

    def test_add_manga_with_last_read_chapter(self):
        manga_test = test_mangas["manga with last read chapter"]
        res = self.manga.add_manga(
            manga_test["manga_url"],
            manga_test["manga_status"],
            manga_test["last_read_chapter"],
            "",
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
            chapter_url="",
            manga_url=manga_test["manga_url"],
            manga_id=0,
        )
        assert res["message"] == "Manga last read chapter updated successfully"

    def test_update_manga_cover_img_to_url(self):
        manga_test = test_mangas["manga without last read chapter"]
        res = self.manga.update_manga_cover_img(
            manga_url=manga_test["manga_url"],
            manga_id=0,
            cover_img_url=self.update_manga_cover_img_to,
        )
        assert res["message"] == "Manga cover image updated successfully"

    def test_get_manga_without_last_read_chapter(self):
        manga_test = test_mangas["manga without last read chapter"]
        manga_test["manga"]["Status"] = self.update_manga_status_to
        manga_test["manga"]["CoverImgURL"] = self.update_manga_cover_img_to
        manga_test["manga"]["CoverImgFixed"] = True
        manga_test["manga"]["LastReadChapter"] = {
            "Chapter": "",
            "URL": "https://mangahub.io/manga/death-note_119",
            "UpdatedAt": datetime(1970, 1, 1, 0, 0, 0),
        }

        actual_manga = self.manga.get_manga(manga_url=manga_test["manga_url"])
        actual_manga["CoverImg"] = ""
        actual_manga["ID"] = 0
        assert actual_manga == manga_test["manga"]

    def test_update_manga_cover_img_to_default(self):
        manga_test = test_mangas["manga without last read chapter"]
        res = self.manga.update_manga_cover_img(
            manga_url=manga_test["manga_url"],
            manga_id=0,
            get_cover_img_from_source=True,
        )
        assert res["message"] == "Manga cover image updated successfully"

    def test_get_manga_without_last_read_chapter_after_cover_img_change(self):
        manga_test = test_mangas["manga without last read chapter"]
        manga_test["manga"]["Status"] = self.update_manga_status_to
        manga_test["manga"]["LastReadChapter"] = {
            "Chapter": "",
            "URL": "https://mangahub.io/manga/death-note_119",
            "UpdatedAt": datetime(1970, 1, 1, 0, 0, 0),
        }

        actual_manga = self.manga.get_manga(manga_url=manga_test["manga_url"])
        actual_manga["CoverImg"] = ""
        actual_manga["ID"] = 0
        assert actual_manga == manga_test["manga"]

    def test_get_manga_with_last_read_chapter(self):
        manga_test = test_mangas["manga with last read chapter"]
        manga_test["manga"]["LastReadChapter"] = self.updated_manga_last_read_chapter
        manga_test["manga"]["LastReadChapter"]["UpdatedAt"] = datetime.combine(
            datetime.today().date(), datetime.min.time()
        )

        actual_manga = self.manga.get_manga(manga_url=manga_test["manga_url"])
        actual_manga["CoverImg"] = ""
        actual_manga["ID"] = 0
        actual_manga["LastReadChapter"]["UpdatedAt"] = datetime.combine(
            datetime.today().date(), datetime.min.time()
        )
        assert actual_manga == manga_test["manga"]

    def test_get_mangas(self):
        mangas = self.manga.get_mangas()
        assert len(mangas) > 1

    def test_get_manga_chapters(self):
        manga_test = test_mangas["manga with last read chapter"]
        chapters = self.manga.get_manga_chapters(manga_url=manga_test["manga_url"])
        assert len(chapters) > 1

    def test_delete_manga_without_last_read_chapter(self):
        manga = test_mangas["manga without last read chapter"]
        res = self.manga.delete_manga(manga_url=manga["manga_url"])
        assert res["message"] == "Manga deleted successfully"

    def test_delete_manga_with_last_read_chapter(self):
        manga = test_mangas["manga with last read chapter"]
        res = self.manga.delete_manga(manga_url=manga["manga_url"])
        assert res["message"] == "Manga deleted successfully"
