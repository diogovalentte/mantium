// Package klmanga provides the implementation of the manga.Source interface for the KLManga source
package klmanga

import (
	"fmt"
	"regexp"

	"github.com/gocolly/colly/v2"
)

var baseSiteURL = "https://klmanga.talk"

// Source is the struct for the KLManga source
type Source struct {
	c *colly.Collector
}

func (Source) GetName() string {
	return "klmanga"
}

var userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"

func newCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent(userAgent),
	)

	return c
}

func (s *Source) resetCollector() {
	if s.c != nil {
		s.c.Wait()
	}

	s.c = newCollector()
}

func extractChapter(s string) (string, error) {
	re := regexp.MustCompile(`第(.*?)話`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 1 {
		return matches[1], nil
	}
	return "", fmt.Errorf("could not extract chapter number from %s", s)
}
