package mangadex

import "testing"

func TestURLIDs(t *testing.T) {
	const id = "87ebd557-8394-4f16-8afe-a8644e555ddc"
	for _, suffix := range []string{"", "/hirayasumi", "/hirayasumi/"} {
		got, err := getMangaID("https://mangadex.org/title/" + id + suffix)
		if err != nil || got != id {
			t.Fatalf("manga ID %q: %q, %v", suffix, got, err)
		}
	}
	got, err := getChapterID("https://mangadex.org/chapter/" + id)
	if err != nil || got != id {
		t.Fatalf("chapter ID: %q, %v", got, err)
	}
	if _, err := getMangaID("https://mangadex.org/no-title"); err == nil {
		t.Fatal("accepted invalid manga URL")
	}
	if _, err := getChapterID("https://mangadex.org/no-chapter"); err == nil {
		t.Fatal("accepted invalid chapter URL")
	}
}
