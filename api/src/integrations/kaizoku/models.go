package kaizoku

// Manga represents a manga in Kaizoku
type Manga struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	Source   string  `json:"source"`
	Interval string  `json:"interval"`
	Library  Library `json:"library"`
}

// Library represents a library in Kaizoku
type Library struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

// Queue represents a Bull queue
type Queue struct {
	Name   string `json:"name"`
	Counts struct {
		Active          int `json:"active"`
		Completed       int `json:"completed"`
		Delayed         int `json:"delayed"`
		Failed          int `json:"failed"`
		Paused          int `json:"paused"`
		Waiting         int `json:"waiting"`
		WaitingChildren int `json:"waiting-children"`
	} `json:"counts"`
	Jobs       []Job `json:"jobs"`
	Pagination struct {
		PageCount int `json:"pageCount"`
		Range     struct {
			Start int `json:"start"`
			End   int `json:"end"`
		} `json:"range"`
	} `json:"pagination"`
	ReadOnlyMode          bool `json:"readOnlyMode"`
	AllowRetries          bool `json:"allowRetries"`
	AllowCompletedRetries bool `json:"allowCompletedRetries"`
	IsPaused              bool `json:"isPaused"`
}

// Job represents a job in Bull queue
type Job struct {
	ID          int `json:"id"`
	Timestamp   int `json:"timestamp"`
	ProcessedOn int `json:"processedOn"`
	Progress    int `json:"progress"`
	Attempts    int `json:"attempts"`
	Delay       int `json:"delay"`
	Opts        struct {
		Attempts         int  `json:"attempts"`
		Delay            int  `json:"delay"`
		JobID            int  `json:"jobId"`
		RemoveOnComplete bool `json:"removeOnComplete"`
	}
	Data     map[string]interface{} `json:"data"`
	Name     string                 `json:"name"`
	IsFailed bool                   `json:"isFailed"`
}
