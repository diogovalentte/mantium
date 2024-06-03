package mangahub

import (
	"testing"
	"time"
)

type releaseTimeTestType struct {
	arg      string
	expected time.Time
	sub      time.Duration
}

var absReleaseTimeTestTable = []releaseTimeTestType{
	{
		arg:      "01-18-2023",
		expected: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC),
	},
	{
		arg:      "12-01-1999",
		expected: time.Date(1999, 12, 1, 0, 0, 0, 0, time.UTC),
	},
}

// relativeReleaseTimeTestTable is a table of test cases for getMangaReleaseTime
// function. Each test case should test the function with a relative time string
// where the relative time is expected to be greater than one day ago.
var relativeReleaseTimeTestTable = []releaseTimeTestType{
	{
		arg: "Yesterday",
		sub: 24 * time.Hour,
	},
	{
		arg: "5 days ago",
		sub: (5 * time.Hour) * 24,
	},
	{
		arg: "1 week ago",
		sub: (1 * time.Hour) * 24 * 7,
	},
	{
		arg: "2 weeks ago",
		sub: (2 * time.Hour) * 24 * 7,
	},
}

// relativeHourTestTable is a table of test cases for getMangaReleaseTime
// function. Each test case should test the function with a relative time string
// where the relative time is expected to be less than one day ago.
var relativeHourTestTable = []releaseTimeTestType{
	{
		arg: "just now",
		sub: 0 * time.Hour,
	},
	{
		arg: "less than an hour",
		sub: 30 * time.Minute,
	},
	{
		arg: "1 hour ago",
		sub: 1 * time.Hour,
	},
	{
		arg: "3 hours ago",
		sub: 3 * time.Hour,
	},
}

func TestGetMangaReleaseTime(t *testing.T) {
	t.Run("Should return a time.Time from absolute time args", func(t *testing.T) {
		for _, test := range absReleaseTimeTestTable {
			actual, err := getMangaReleaseTime(test.arg)
			if err != nil {
				t.Fatalf("error while getting manga release time: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		}
	})
	t.Run("Should return a time.Time from relative time args where expected is greater than one day ago", func(t *testing.T) {
		for _, test := range relativeReleaseTimeTestTable {
			actual, err := getMangaReleaseTime(test.arg)
			if err != nil {
				t.Fatalf("error while getting manga release time: %v", err)
			}

			expectedDate := time.Now().Add(test.sub * -1)
			expected := time.Date(expectedDate.Year(), expectedDate.Month(), expectedDate.Day(), 0, 0, 0, 0, time.Local)
			if actual != expected {
				t.Fatalf("expected %v, got %v", expected, actual)
			}
		}
	})
	t.Run("Should return a time.Time from relative time args where expected is less than one day ago", func(t *testing.T) {
		for _, test := range relativeHourTestTable {
			actual, err := getMangaReleaseTime(test.arg)
			if err != nil {
				t.Fatalf("error while getting manga release time: %v", err)
			}

			expectedDate := time.Now().Add(test.sub * -1)
			beforeExpected := expectedDate.Add(1 * time.Second)
			afterExpected := expectedDate.Add(-1 * time.Second)
			if !actual.Before(beforeExpected) {
				t.Fatalf("expected %v to be before %v", actual, beforeExpected)
			}
			if !actual.After(afterExpected) {
				t.Fatalf("expected %v to be after %v", actual, afterExpected)
			}
		}
	})
}
