package kaizoku

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources"
	"github.com/diogovalentte/mantium/api/src/util"
)

func (k *Kaizoku) baseRequest(method, url string, body io.Reader) (*http.Response, error) {
	errorContext := "error while making '%s' request"

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, method), err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := k.c.Do(req)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, method), err)
	}

	return resp, nil
}

func validateResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("non-200 status code -> (%d). Body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (k *Kaizoku) GetSources() ([]string, error) {
	errorContext := "(kaizoku) error while getting manga sources"

	url := fmt.Sprintf("%s/api/trpc/manga.sources", k.Address)
	resp, err := k.baseRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	var mangas getMangaSources
	err = json.NewDecoder(resp.Body).Decode(&mangas)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return mangas.Result.Data.JSON, nil
}

func (k *Kaizoku) GetMangas() ([]*Manga, error) {
	errorContext := "(kaizoku) error while getting mangas"

	url := fmt.Sprintf("%s/api/trpc/manga.query", k.Address)
	resp, err := k.baseRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	var mangas getMangasResponse
	err = json.NewDecoder(resp.Body).Decode(&mangas)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return mangas.Result.Data.JSON, nil
}

func (k *Kaizoku) GetManga(mangaName string) (*Manga, error) {
	errorContext := "(kaizoku) error while getting manga with name '%s'"

	mangas, err := k.GetMangas()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaName), err)
	}

	for _, m := range mangas {
		if m.Title == mangaName {
			return m, nil
		}
	}

	return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaName), fmt.Errorf("manga not found in Kaizoku"))
}

func (k *Kaizoku) AddManga(manga *manga.Manga, tryOtherSources bool) error {
	errorContext := "(kaizoku) error while adding manga '%s' / '%s'"

	var lastError error
	var errors []error
	for source := range sources.GetSources() {
		lastError = k.addMangaToKaizoku(manga)
		if lastError != nil {
			errors = append(errors, fmt.Errorf("error with source '%s': %s", manga.Source, lastError))
			if tryOtherSources {
				manga.Source = source
				continue
			}
		}
		break
	}
	if lastError != nil {
		if len(errors) == 1 {
			return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), errors[0])
		}
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), fmt.Errorf("error with all sources: %s", errors))
	}

	return nil
}

func (k *Kaizoku) addMangaToKaizoku(manga *manga.Manga) error {
	mangaTitle := manga.Name
	mangaInterval := k.DefaultInterval
	mangaSource, err := k.getKaizokuSource(manga.Source)
	if err != nil {
		return err
	}
	reqBody := fmt.Sprintf(`{"0":{"json":{"title":"%s","source":"%s","interval":"%s"}}}`, mangaTitle, mangaSource, mangaInterval)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error while marshalling request body: %s", err)
	}

	url := fmt.Sprintf("%s/api/trpc/manga.add?batch=1", k.Address)
	resp, err := k.baseRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		if util.ErrorContains(err, fmt.Sprintf("Cannot find the %s.", mangaTitle)) {
			return fmt.Errorf("cannot find the manga. Maybe there is no Anilist page for this manga (Kaizoku can't add mangas that don't have one): Kaizoku API error: %s", err.Error())
		}
		return nil
	}

	return nil
}

func (k *Kaizoku) RemoveManga(mangaID int, removeFiles bool) error {
	errorContext := "(kaizoku) error while removing manga with id '%d' (removeFiles: %v)"

	reqBody := fmt.Sprintf(`{"0":{"json":{"id": %d, "shouldRemoveFiles": %v}}}`, mangaID, removeFiles)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return util.AddErrorContext(util.AddErrorContext(fmt.Sprintf(errorContext, mangaID, removeFiles), fmt.Errorf("error while marshalling request body")).Error(), err)
	}

	url := fmt.Sprintf("%s/api/trpc/manga.remove?batch=1", k.Address)
	resp, err := k.baseRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, mangaID, removeFiles), err)
	}
	defer resp.Body.Close()

	// It returns 500 when the manga is removed with success (weird)
	if resp.StatusCode != http.StatusInternalServerError && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return util.AddErrorContext(fmt.Sprintf(errorContext, mangaID, removeFiles), fmt.Errorf("non-200/500 status code -> (%d). Body: %s", resp.StatusCode, string(body)))
	}

	return nil
}

func (k *Kaizoku) CheckOutOfSyncChapters() error {
	errorContext := "(kaizoku) error while checking out of sync chapters"

	reqBody := `{"0":{"json":{"id": null}}}`

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return util.AddErrorContext(util.AddErrorContext(errorContext, fmt.Errorf("error while marshalling request body")).Error(), err)
	}

	url := fmt.Sprintf("%s/api/trpc/manga.checkOutOfSyncChapters?batch=1", k.Address)
	resp, err := k.baseRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return util.AddErrorContext(errorContext, err)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		if util.ErrorContains(err, "There is another active job running. Please wait until it finishes") {
			return util.AddErrorContext(errorContext, fmt.Errorf("there is another active job running"))
		}
		return util.AddErrorContext(errorContext, err)
	}

	return nil
}

func (k *Kaizoku) FixOutOfSyncChapters() error {
	errorContext := "(kaizoku) error while fixing out of sync chapters"

	reqBody := `{"0":{"json":{"id": null}}}`

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return util.AddErrorContext(util.AddErrorContext(errorContext, fmt.Errorf("error while marshalling request body")).Error(), err)
	}

	url := fmt.Sprintf("%s/api/trpc/manga.fixOutOfSyncChapters?batch=1", k.Address)
	resp, err := k.baseRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return util.AddErrorContext(errorContext, err)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		if util.ErrorContains(err, "There is another active job running. Please wait until it finishes") {
			return util.AddErrorContext(errorContext, fmt.Errorf("there is another active job running"))
		}
		return util.AddErrorContext(errorContext, err)
	}

	return nil
}

type getMangaSources struct {
	Result struct {
		Data struct {
			JSON []string `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type getMangasResponse struct {
	Result struct {
		Data struct {
			JSON []*Manga `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

func (k *Kaizoku) getKaizokuSource(source string) (string, error) {
	errorContext := "error while getting Kaizoku source"
	kaizokuSources, err := k.GetSources()
	if err != nil {
		return "", util.AddErrorContext(errorContext, err)
	}

	var returnSource string
	switch source {
	case "mangadex.org":
		returnSource = "MangaDex"
	case "comick.xyz":
		returnSource = "ComicK"
	case "comick.io":
		returnSource = "ComicK"
	case "rawkuma.com":
		returnSource = "RawKuma"
	case "mangahub.io":
		return "", util.AddErrorContext(errorContext, fmt.Errorf("MangaHub source is not implemented"))
	case "mangaplus.shueisha.co.jp":
		return "", util.AddErrorContext(errorContext, fmt.Errorf("Manga Plus source is not implemented"))
	case "mangaupdates":
		return "", util.AddErrorContext(errorContext, fmt.Errorf("MangaUpdates source is not implemented"))
	default:
		return "", util.AddErrorContext(errorContext, fmt.Errorf("unknown source"))
	}

	for _, s := range kaizokuSources {
		if s == returnSource {
			return returnSource, nil
		}
	}

	return "", util.AddErrorContext(errorContext, fmt.Errorf("source not found in Kaizoku"))
}
