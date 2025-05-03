package suwayomi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	req.SetBasicAuth(s.Username, s.Password)

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

func (s *Suwayomi) AddManga(manga *manga.Manga, enqueChapterDownloads bool) error {
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

	if enqueChapterDownloads && manga.Source != "comick" {
		chapters, err := s.GetChapters(sourceManga.ID)
		if err != nil {
			return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), util.AddErrorContext("error while getting chapters", err))
		}
		chapterIDs := make([]int, len(chapters))
		for i, chapter := range chapters {
			chapterIDs[i] = chapter.ID
		}
		err = s.EnqueueChapterDownloads(chapterIDs)
		if err != nil {
			return util.AddErrorContext(fmt.Sprintf(errorContext, manga.Name, manga.URL), util.AddErrorContext("error while enqueueing chapter downloads", err))
		}
	}

	return nil
}

func (s *Suwayomi) GetLibraryMangaID(m *manga.Manga) (int, error) {
	errorContext := "error while getting in-library manga ID for manga '%s'"

	query := `
query AllCategories {
	mangas(condition: {inLibrary: true, url: "%s"}) {
    nodes {
      realUrl
      id
    }
  }
}
	`

	URL, err := s.getSourceMangaURL(m)
	if err != nil {
		return 0, util.AddErrorContext(fmt.Sprintf(errorContext, m), err)
	}

	query = fmt.Sprintf(query, URL)

	payload := map[string]any{
		"query": query,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return 0, util.AddErrorContext(fmt.Sprintf(errorContext, m), util.AddErrorContext("error while marshalling payload", err))
	}

	var mangaResponse struct {
		Data struct {
			Mangas struct {
				Nodes []*APIManga `json:"nodes"`
			} `json:"mangas"`
		} `json:"data"`
	}
	_, err = s.baseRequest(bytes.NewBuffer(jsonData), &mangaResponse)
	if err != nil {
		return 0, util.AddErrorContext(fmt.Sprintf(errorContext, m), err)
	}

	if len(mangaResponse.Data.Mangas.Nodes) == 0 {
		return 0, util.AddErrorContext(fmt.Sprintf(errorContext, m), fmt.Errorf("manga not found in library"))
	} else if len(mangaResponse.Data.Mangas.Nodes) > 1 {
		return 0, util.AddErrorContext(fmt.Sprintf(errorContext, m), fmt.Errorf("multiple mangas found in library"))
	}

	return mangaResponse.Data.Mangas.Nodes[0].ID, nil
}

func (s *Suwayomi) GetChapters(mangaID int) ([]*APIChapter, error) {
	errorContext := "error while getting chapters for manga '%d'"

	query := `
query AllCategories {
  manga(id: %d) {
    chapters {
      nodes {
        isDownloaded
        realUrl
        id
      }
    }
  }
}
	`
	query = fmt.Sprintf(query, mangaID)

	payload := map[string]any{
		"query": query,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaID), util.AddErrorContext("error while marshalling payload", err))
	}

	var chaptersResponse struct {
		Data struct {
			Manga struct {
				Chapters struct {
					Nodes []*APIChapter `json:"nodes"`
				} `json:"chapters"`
			} `json:"manga"`
		} `json:"data"`
	}
	_, err = s.baseRequest(bytes.NewBuffer(jsonData), &chaptersResponse)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaID), err)
	}

	if len(chaptersResponse.Data.Manga.Chapters.Nodes) == 0 {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaID), fmt.Errorf("manga not found"))
	}

	return chaptersResponse.Data.Manga.Chapters.Nodes, nil
}

func (s *Suwayomi) GetChapter(mangaID int, chapterURL string) (*APIChapter, error) {
	errorContext := "error while getting chapter '%s' for manga '%d'"

	chapters, err := s.GetChapters(mangaID)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, chapterURL, mangaID), err)
	}

	for _, chapter := range chapters {
		if strings.Contains(chapter.RealURL, chapterURL) {
			return chapter, nil
		}
	}

	return nil, util.AddErrorContext(fmt.Sprintf(errorContext, chapterURL, mangaID), fmt.Errorf("chapter not found"))
}

func (s *Suwayomi) EnqueueChapterDownloads(chapterIDs []int) error {
	errorContext := "error while enqueueing chapter downloads for chapters '%s'"

	strSlice := make([]string, len(chapterIDs))
	for i, id := range chapterIDs {
		strSlice[i] = strconv.Itoa(id)
	}
	chapterIDsStr := strings.Join(strSlice, ",")

	query := `
mutation MyMutation($ids: [Int!] = [%s]) {
  enqueueChapterDownloads(input: {ids: $ids}) {
    clientMutationId
  }
}
	`
	query = fmt.Sprintf(query, chapterIDsStr)

	payload := map[string]any{
		"query": query,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, chapterIDsStr), util.AddErrorContext("error while marshalling payload", err))
	}

	_, err = s.baseRequest(bytes.NewBuffer(jsonData), nil)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(errorContext, chapterIDsStr), err)
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
