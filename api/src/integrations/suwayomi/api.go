package suwayomi

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

func (s *Suwayomi) baseRequest(reqBody io.Reader, target any) (*http.Response, error) {
	errorContext := "error while making request"

	req, err := http.NewRequest("POST", s.Address+"/api/graphql", reqBody)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.c.Do(req)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, util.AddErrorContext(errorContext, fmt.Errorf("non-200 status code -> (%d). Body: %s", resp.StatusCode, string(body)))
	}

	if target != nil {
		err = json.NewDecoder(resp.Body).Decode(target)
		if err != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, util.AddErrorContext(errorContext, fmt.Errorf("error while decoding response: '%s'. Body: %s", err.Error(), string(body)))
		}
	}

	return resp, nil
}

func (s *Suwayomi) fetchSources() ([]*Extension, error) {
	errorContext := "error while fetching sources"
	query := `
mutation {
  fetchExtensions(input: {}) {
    extensions {
      source {
        edges {
          node {
            id,displayName
          }
        }
      }
    }
  }
}
	`
	payload := map[string]any{
		"query": query,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, util.AddErrorContext("error while marshalling payload", err))
	}

	var fetchSourcesResponse FetchSourcesResponse
	_, err = s.baseRequest(bytes.NewBuffer(jsonData), &fetchSourcesResponse)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return fetchSourcesResponse.Data.FetchExtensions.Extensions, nil
}

type FetchSourcesResponse struct {
	Data struct {
		FetchExtensions struct {
			Extensions []*Extension `json:"extensions"`
		} `json:"fetchExtensions"`
	} `json:"data"`
}

type Extension struct {
	Source *Source `json:"source"`
}

type Source struct {
	Edges []*struct {
		Node *struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"node"`
	} `json:"edges"`
}

func (s *Suwayomi) fetchSourceID(sourceName string) (string, error) {
	errorContext := "error while fetching source ID for source '%s'"
	sources, err := s.fetchSources()
	if err != nil {
		return "", util.AddErrorContext(fmt.Sprintf(errorContext, sourceName), err)
	}

	for _, source := range sources {
		for _, edge := range source.Source.Edges {
			if edge.Node.DisplayName == sourceName {
				return edge.Node.ID, nil
			}
		}
	}

	return "", util.AddErrorContext(fmt.Sprintf(errorContext, sourceName), fmt.Errorf("source not found/installed"))
}

func (s *Suwayomi) fetchSourceManga(sourceID string, m *manga.Manga, page int) (*APIManga, error) {
	errorContext := "error while fetching manga from source"

	query := `
		mutation FetchSourceManga($input: FetchSourceMangaInput!) {
		  fetchSourceManga(input: $input) {
			mangas {
			  url,id,inLibrary
			}
		  }
		}
	`

	variables := map[string]any{
		"input": map[string]any{
			"query":  m.Name,
			"source": sourceID,
			"type":   "SEARCH",
			"page":   page,
		},
	}

	payload := map[string]any{
		"query":         query,
		"variables":     variables,
		"operationName": "FetchSourceManga",
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, util.AddErrorContext("error while marshalling payload", err))
	}

	var fetchMangasResponse FetchMangasResponse
	_, err = s.baseRequest(bytes.NewBuffer(jsonData), &fetchMangasResponse)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	mangaURL, err := s.getSourceMangaURL(m)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	for _, manga := range fetchMangasResponse.Data.FetchSourceManga.Mangas {
		if manga.URL == mangaURL {
			return manga, nil
		}
	}

	return nil, util.AddErrorContext(errorContext, fmt.Errorf("manga not found"))
}

type FetchMangasResponse struct {
	Data struct {
		FetchSourceManga struct {
			HasNextPage bool        `json:"hasNextPage"`
			Mangas      []*APIManga `json:"mangas"`
		} `json:"fetchSourceManga"`
	} `json:"data"`
}

type APIManga struct {
	ID        int    `json:"id"`
	InLibrary bool   `json:"inLibrary"`
	URL       string `json:"url"`
}

func (s *Suwayomi) addManga(mangaID int) error {
	errorContext := "error while adding manga with ID '%d'"

	query := `
		mutation UpdateManga($input: UpdateMangaInput!) {
		  updateManga(input: $input) {
			manga {
			  inLibrary
			}
		  }
		}
	`

	variables := map[string]any{
		"input": map[string]any{
			"id": mangaID,
			"patch": map[string]any{
				"inLibrary": true,
			},
		},
	}

	payload := map[string]any{
		"query":         query,
		"variables":     variables,
		"operationName": "UpdateManga",
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, mangaID), util.AddErrorContext("error while marshalling payload", err))
	}

	_, err = s.baseRequest(bytes.NewBuffer(jsonData), nil)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, mangaID), err)
	}

	return nil
}

func (s *Suwayomi) AddManga(manga *manga.Manga) error {
	errorContext := "(suwayomi) error while adding manga '%s' / '%s'"

	source, err := s.translateSuwayomiSource(manga.Source)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), err)
	}
	sourceID, err := s.fetchSourceID(source)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), err)
	}

	sourceManga, err := s.fetchSourceManga(sourceID, manga, 1)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), err)
	}

	if sourceManga.InLibrary {
		return nil
	}

	err = s.addManga(sourceManga.ID)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), err)
	}

	return nil
}

func (s *Suwayomi) translateSuwayomiSource(sourceName string) (string, error) {
	errorContext := "error while translating Mantium source '%s' to Suwayomi source"

	switch sourceName {
	case "comick":
		return "Comick (ALL)", nil
	case "mangadex":
		return "MangaDex (EN)", nil
	case "mangaplus":
		return "MANGA Plus by SHUEISHA (EN)", nil
	case "mangahub":
		return "MangaHub (EN)", nil
	}

	return "", util.AddErrorContext(fmt.Sprintf(errorContext, sourceName), fmt.Errorf("source not found"))
}

func (s *Suwayomi) getSourceMangaURL(manga *manga.Manga) (string, error) {
	errorContext := "error while getting source manga URL for manga '%s' / '%s'"

	switch manga.Source {
	case "comick":
		URLParts := strings.Split(manga.URL, "/")
		return fmt.Sprintf("/comic/%s#", URLParts[len(URLParts)-1]), nil
	case "mangadex":
		URLParts := strings.Split(manga.URL, "/")
		return fmt.Sprintf("/manga/%s", URLParts[len(URLParts)-1]), nil
	case "mangaplus":
		URLParts := strings.Split(manga.URL, "/")
		return fmt.Sprintf("#/titles/%s", URLParts[len(URLParts)-1]), nil
	case "mangahub":
		URLParts := strings.Split(manga.URL, "/")
		return fmt.Sprintf("/manga/%s", URLParts[len(URLParts)-1]), nil
	}

	return "", util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), fmt.Errorf("source not found"))
}
