package mangadex

import "testing"

func TestGetMangaMetadata(t *testing.T) {
	t.Run("should return the metadata of a manga given its URL", func(t *testing.T) {
		source := &Source{}
		mangaURL := "https://mangadex.org/title/87ebd557-8394-4f16-8afe-a8644e555ddc/hirayasumi"
		_, err := source.GetMangaMetadata(mangaURL)
		if err != nil {
			t.Errorf("Error: %s", err)
			return
		}
	})
}

func TestGetMangaID(t *testing.T) {
	t.Run("should return the ID of a manga URL", func(t *testing.T) {
		mangaURL := "https://mangadex.org/title/87ebd557-8394-4f16-8afe-a8644e555ddc/hirayasumi"
		expected := "87ebd557-8394-4f16-8afe-a8644e555ddc"
		result, err := getMangaID(mangaURL)
		if err != nil {
			t.Errorf("Error: %s", err)
			return
		}
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
			return
		}
	})
}
