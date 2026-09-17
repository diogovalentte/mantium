// Package util implements utility functions
package util

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/nfnt/resize"
	"github.com/rs/zerolog"
	"golang.org/x/image/webp"
)

var logger *zerolog.Logger

// GetLogger returns the zerolog logger instance
func GetLogger(logLevel zerolog.Level) *zerolog.Logger {
	if logger == nil {
		l := zerolog.New(os.Stdout).Level(logLevel).With().Timestamp().Logger()
		logger = &l
	}

	return logger
}

// AddErrorContext adds context to an error, like:
// "Error downloading image: Get "https://example.com/image.jpg": dial tcp: lookup example.com: no such host".
// Should be used in functions that can return multiple errors without a spefic origin/context.
func AddErrorContext(context string, err error) error {
	return fmt.Errorf("%s: %w", context, err)
}

// ErrorContains checks if an error contains a specific string
func ErrorContains(err error, s string) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), s)
}

// RemoveLastOccurrence removes the last occurrence of a string from another string
func RemoveLastOccurrence(s, old string) string {
	if old == "" {
		return s
	}

	lastIndex := strings.LastIndex(s, old)
	modifiedString := s
	if lastIndex != -1 {
		modifiedString = s[:lastIndex] + s[lastIndex+len(old):]
	}

	return modifiedString
}

var (
	// DefaultImageHeight is the default height of an image
	DefaultImageHeight = 355
	// DefaultImageWidth is the default width of an image
	DefaultImageWidth = 250
)

// GetImageFromURL downloads an image from a URL and tries to resize it.
// If the image is not resized, it returns the original image.
func GetImageFromURL(url string, retries int, retryInterval time.Duration) (imgBytes []byte, resized bool, err error) {
	contextError := "error downloading image '%s'"

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	if retries < 1 {
		retries = 1
	}

	var imageBytes []byte
	var lastErr error
	for i := 0; i < retries; i++ {
		if i > 0 {
			time.Sleep(retryInterval)
		}

		imageBytes, lastErr = downloadImage(httpClient, url)
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return nil, resized, AddErrorContext(fmt.Sprintf(contextError, url), lastErr)
	}

	if strings.HasSuffix(url, ".webp") {
		imageBytes, err = webpToJPEG(imageBytes)
		if err != nil {
			return nil, resized, AddErrorContext(fmt.Sprintf(contextError, url), AddErrorContext("could not convert webp image to jpeg", err))
		}
	}

	if !IsImageValid(imageBytes) {
		return nil, resized, AddErrorContext(fmt.Sprintf(contextError, url), fmt.Errorf("invalid image"))
	}

	img, err := ResizeImage(imageBytes, uint(DefaultImageWidth), uint(DefaultImageHeight))
	if err != nil {
		// JPEG format that has an unsupported subsampling ratio
		// It's a valid image but the standard library doesn't support it
		// and other libraries use the standard library under the hood
		if ErrorContains(err, "unsupported JPEG feature: luma/chroma subsampling ratio") {
			img = imageBytes
		} else {
			return nil, resized, AddErrorContext(fmt.Sprintf(contextError, url), err)
		}
	} else {
		resized = true
	}

	return img, resized, nil
}

// downloadImage performs a single image download attempt and always closes the
// response body before returning.
func downloadImage(httpClient *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, AddErrorContext("error while creating request", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, AddErrorContext("error while executing request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("non-200 status code -> (%d). Body: %s", resp.StatusCode, string(body))
	}

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, AddErrorContext("could not read the image data from request body", err)
	}

	return imageBytes, nil
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

// ResizeImage resizes an image to the specified width and height
func ResizeImage(imgBytes []byte, width, height uint) ([]byte, error) {
	contextError := "error resizing image to width %d and height %d"

	_, format, err := image.DecodeConfig(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, AddErrorContext(fmt.Sprintf(contextError, width, height), err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, AddErrorContext(fmt.Sprintf(contextError, width, height), err)
	}

	resizedImg := resize.Resize(width, height, img, resize.Lanczos3)

	var resizedBuf bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&resizedBuf, resizedImg, nil)
	case "png":
		err = png.Encode(&resizedBuf, resizedImg)
	default:
		return nil, AddErrorContext(fmt.Sprintf(contextError, width, height), fmt.Errorf("unsupported image format to resize: %s", format))
	}
	if err != nil {
		return nil, AddErrorContext(fmt.Sprintf(contextError, width, height), err)
	}

	return resizedBuf.Bytes(), nil
}

// GetDefaultCoverImg returns the default cover image with the right size
func GetDefaultCoverImg() ([]byte, error) {
	contextError := "error getting default cover image from file"

	path := "../defaults/default_cover_img.jpg"

	if _, err := os.Stat(path); err != nil {
		// when testing the path is different
		path = "../../../defaults/default_cover_img.jpg"
	}

	img, err := os.ReadFile(path)
	if err != nil {
		return nil, AddErrorContext(contextError, err)
	}

	return img, nil
}

// IsImageValid checks if an image is valid by decoding it
func IsImageValid(imgBytes []byte) bool {
	_, _, err := image.DecodeConfig(bytes.NewReader(imgBytes))
	if err != nil {
		return ErrorContains(err, "luma/chroma subsampling ratio")
	}

	return true
}

// GetRFC3339Datetime returns a time.Time from a RFC3339 formatted string.
// Also truncate the time to seconds.
func GetRFC3339Datetime(date string) (time.Time, error) {
	contextError := "error parsing RFC3339 datetime '%s'"

	parsedDate, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return time.Time{}, AddErrorContext(fmt.Sprintf(contextError, date), err)
	}
	parsedDate = parsedDate.In(time.Local).Truncate(time.Second)

	return parsedDate, nil
}

// RequestUpdateMangasMetadata sends a request to the server to update all mangas metadata
func RequestUpdateMangasMetadata(notify bool) (*http.Response, error) {
	contextErrror := "error requesting to update mangas metadata (notify is %v)"

	client := &http.Client{}

	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	url := fmt.Sprintf("http://localhost:%s/v1/mangas/metadata", apiPort)
	if notify {
		url += "?notify=true"
	}
	req, err := http.NewRequest("PATCH", url, nil)
	if err != nil {
		return nil, AddErrorContext(fmt.Sprintf(contextErrror, notify), err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, AddErrorContext(fmt.Sprintf(contextErrror, notify), err)
	}

	if resp.StatusCode != http.StatusOK {
		return resp, AddErrorContext(fmt.Sprintf(contextErrror, notify), fmt.Errorf("non-200 status code -> (%d)", resp.StatusCode))
	}

	return resp, nil
}

// FileExists checks if a file exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

// GetDomain extracts the domain from a given URL.
func GetDomain(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid URL: missing scheme or host")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

// ChunkSlice splits items into at most n contiguous chunks of roughly equal
// size. An empty slice yields no chunks, n below 1 is treated as 1, and n
// larger than len(items) is capped so no empty or inverted chunk is produced.
func ChunkSlice[T any](items []T, n int) [][]T {
	if len(items) == 0 {
		return nil
	}
	if n < 1 {
		n = 1
	}
	n = min(n, len(items))

	chunkSize := (len(items) + n - 1) / n
	chunks := make([][]T, 0, n)
	for start := 0; start < len(items); start += chunkSize {
		chunks = append(chunks, items[start:min(start+chunkSize, len(items))])
	}

	return chunks
}
