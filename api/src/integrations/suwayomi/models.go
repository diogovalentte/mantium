package suwayomi

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
	RealURL   string `json:"realURL"`
	Title     string `json:"title"`
}

type APIChapter struct {
	ID           int    `json:"id"`
	IsDownloaded bool   `json:"isDownloaded"`
	RealURL      string `json:"realURL"`
}
