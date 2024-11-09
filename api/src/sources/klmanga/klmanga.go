// Package klmanga provides the implementation of the manga.Source interface for the KLManga source
package klmanga

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/gocolly/colly/v2"
	"golang.org/x/image/webp"

	"github.com/diogovalentte/mantium/api/src/util"
)

var baseSiteURL = "https://klmanga.rs"

// Source is the struct for the KLManga source
type Source struct {
	c *colly.Collector
}

var userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"

func newCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains("klmanga.rs"),
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

func getImageFromURL(url string, retries int, retryInterval time.Duration) (imageBytes []byte, resized bool, err error) {
	contextError := "error downloading image '%s'"

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	webpImgBytes := make([]byte, 0)
	for i := 0; i < retries; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			if i == retries-1 {
				return nil, resized, util.AddErrorContext(fmt.Sprintf(contextError, url), util.AddErrorContext("error while creating request", err))
			}
			time.Sleep(retryInterval)
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0")

		resp, err := httpClient.Do(req)
		if err != nil {
			if i == retries-1 {
				return nil, resized, util.AddErrorContext(fmt.Sprintf(contextError, url), util.AddErrorContext("error while executing request", err))
			}
			time.Sleep(retryInterval)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if i == retries-1 {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				return nil, resized, util.AddErrorContext(fmt.Sprintf(contextError, url), fmt.Errorf("non-200 status code -> (%d). Body: %s", resp.StatusCode, string(body)))
			}
			time.Sleep(retryInterval)
			continue
		}

		webpImgBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			if i == retries-1 {
				return nil, resized, util.AddErrorContext(fmt.Sprintf(contextError, url), util.AddErrorContext("could not read the image data from request body", err))
			}
			time.Sleep(retryInterval)
			continue
		}
	}

	imageBytes, err = webpToJPEG(webpImgBytes)
	if err != nil {
		return nil, resized, util.AddErrorContext(fmt.Sprintf(contextError, url), util.AddErrorContext("could not convert webp image to jpeg", err))
	}

	if !util.IsImageValid(imageBytes) {
		return nil, resized, util.AddErrorContext(fmt.Sprintf(contextError, url), fmt.Errorf("invalid image"))
	}

	img, err := util.ResizeImage(imageBytes, uint(util.DefaultImageWidth), uint(util.DefaultImageHeight))
	if err != nil {
		// JPEG format that has an unsupported subsampling ratio
		// It's a valid image but the standard library doesn't support it
		// and other libraries use the standard library under the hood
		if util.ErrorContains(err, "unsupported JPEG feature: luma/chroma subsampling ratio") {
			img = imageBytes
		} else {
			return nil, resized, util.AddErrorContext(fmt.Sprintf(contextError, url), err)
		}
	} else {
		resized = true
	}

	return img, resized, nil
}

func webpToJPEG(webpImgBytes []byte) ([]byte, error) {
	webpReader := bytes.NewReader(webpImgBytes)
	img, err := webp.Decode(webpReader)
	if err != nil {
		return nil, fmt.Errorf("could not decode webp image")
	}

	var jpegImgBytes bytes.Buffer
	err = jpeg.Encode(&jpegImgBytes, img, nil)
	if err != nil {
		return nil, fmt.Errorf("could not encode image to jpeg")
	}

	return jpegImgBytes.Bytes(), nil
}
