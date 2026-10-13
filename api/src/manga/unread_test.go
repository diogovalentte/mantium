package manga

import "testing"

func TestIsUnreadChapter(t *testing.T) {
	tests := []struct {
		lastRead, lastReadURL, lastReleased, lastReleasedURL string
		want                                                 bool
	}{
		{"", "", "", "", false},
		{"10", "", "", "", false},
		{"", "", "10", "", true},
		{"9", "", "10", "", true},
		{"10", "", "10", "", false},
		{"11", "", "10", "", false},
		{"Extra", "", "10", "", true},
		{UnknownChapter, "https://a/1", UnknownChapter, "https://a/2", true},
		{UnknownChapter, "https://a/1", UnknownChapter, "https://a/1", false},
	}
	for _, test := range tests {
		got := IsUnreadChapter(test.lastRead, test.lastReadURL, test.lastReleased, test.lastReleasedURL)
		if got != test.want {
			t.Errorf("IsUnreadChapter(%q, %q, %q, %q) = %v, want %v", test.lastRead, test.lastReadURL, test.lastReleased, test.lastReleasedURL, got, test.want)
		}
	}
}
