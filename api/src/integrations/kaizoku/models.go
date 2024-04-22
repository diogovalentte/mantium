package kaizoku

type Manga struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	Source   string  `json:"source"`
	Interval string  `json:"interval"`
	Library  Library `json:"library"`
}

type Library struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}
