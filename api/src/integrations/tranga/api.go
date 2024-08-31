package tranga

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

func (t *Tranga) AddManga(manga *manga.Manga) error {
	errorContext := "(tranga) error while adding manga '%s' / '%s'"

	mangaConnector, err := t.getTrangaConnector(manga.Source)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), err)
	}

	trangaManga, err := t.SearchManga(manga, mangaConnector)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), err)
	}

	err = t.addManga(trangaManga.InternalID, mangaConnector)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), err)
	}

	_, err = t.GetMonitorJobBySiteURL(manga.URL)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), err)
	}

	return nil
}

func (t *Tranga) addManga(internalID, connector string) error {
	url := fmt.Sprintf("%s/Jobs/MonitorManga?translatedLanguage=en&connector=%s&internalId=%s&interval=%s", t.Address, connector, internalID, t.DefaultInterval)
	_, err := t.request(http.MethodPost, url, nil, nil)
	if err != nil {
		return err
	}

	return err
}

// SearchManga searches for a manga in Tranga. It will match by the URL.
func (t *Tranga) SearchManga(manga *manga.Manga, connector string) (*Manga, error) {
	mangaReqName := strings.ReplaceAll(manga.Name, " ", "%20")
	url := fmt.Sprintf("%s/Manga/FromConnector?connector=%s&title=%s", t.Address, connector, mangaReqName)
	var mangas []*Manga
	_, err := t.request(http.MethodGet, url, nil, &mangas)
	if err != nil {
		return nil, err
	}

	if len(mangas) == 0 {
		return nil, fmt.Errorf("no manga results from tranga")
	}
	for _, m := range mangas {
		if strings.Contains(m.WebSiteURL, manga.URL) {
			return m, nil
		}
	}

	return nil, fmt.Errorf("manga not found in Tranga")
}

func (t *Tranga) GetMonitorJobs() ([]*getMonitorJobsResponse, error) {
	errorContext := "error while getting monitor jobs"

	url := fmt.Sprintf("%s/Jobs/MonitorJobs", t.Address)
	var jobs []*getMonitorJobsResponse
	_, err := t.request(http.MethodGet, url, nil, &jobs)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return jobs, nil
}

type getMonitorJobsResponse struct {
	Manga   Manga  `json:"manga"`
	ID      string `json:"id"`
	JobType int    `json:"jobType"`
}

func (t *Tranga) GetMonitorJobBySiteURL(mangaSiteURL string) (*getMonitorJobsResponse, error) {
	errorContext := "error while getting monitor job by site URL for manga '%s'"

	jobs, err := t.GetMonitorJobs()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaSiteURL), err)
	}

	for _, job := range jobs {
		if strings.Contains(job.Manga.WebSiteURL, mangaSiteURL) {
			return job, nil
		}
	}

	return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaSiteURL), fmt.Errorf("monitor job not found in Tranga"))
}

func (t *Tranga) StartJob(manga *manga.Manga) error {
	errorContext := "(tranga) error while starting job for manga '%s' / '%s'"

	job, err := t.GetMonitorJobBySiteURL(manga.URL)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), err)
	}

	url := fmt.Sprintf("%s/Jobs/StartNow?jobId=%s", t.Address, job.ID)
	_, err = t.request(http.MethodPost, url, nil, nil)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), err)
	}

	return nil
}

func (t *Tranga) GetConnectors() ([]string, error) {
	errorContext := "error while getting manga connectors"

	url := fmt.Sprintf("%s/Connectors", t.Address)
	resp, err := t.request(http.MethodGet, url, nil, nil)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	defer resp.Body.Close()

	var connectors []string
	err = json.NewDecoder(resp.Body).Decode(&connectors)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return connectors, nil
}

// request is a helper function to make a request to the Comick API
func (t *Tranga) request(method, url string, reqBody io.Reader, retBody interface{}) (*http.Response, error) {
	errorContext := "error while making '%s' request"

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, method), err)
	}

	header := http.Header{}
	header.Set("Content-Length", "0")
	req.Header = header

	resp, err := t.c.Do(req)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, method), err)
	} else if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, method), fmt.Errorf("non-200 status code -> (%d). Body: %s", resp.StatusCode, string(body)))
	}

	if retBody != nil {
		body, _ := io.ReadAll(resp.Body)
		if err = json.NewDecoder(bytes.NewReader(body)).Decode(retBody); err != nil {
			return nil, util.AddErrorContext(fmt.Sprintf(errorContext, method), fmt.Errorf("error decoding request body response. Body: %s", string(body)))
		}
	}

	return resp, nil
}

func (t *Tranga) getTrangaConnector(source string) (string, error) {
	errorContext := "error while getting manga connector"

	var returnConnector string
	switch source {
	case "mangadex.org":
		returnConnector = "MangaDex"
	default:
		return "", util.AddErrorContext(errorContext, fmt.Errorf("%s connector is not implemented in Tranga", source))
	}

	connectors, err := t.GetConnectors()
	if err != nil {
		return "", util.AddErrorContext(errorContext, err)
	}

	for _, c := range connectors {
		if c == returnConnector {
			return returnConnector, nil
		}
	}

	return "", util.AddErrorContext(errorContext, fmt.Errorf("%s connector not found in Tranga instance", returnConnector))
}
