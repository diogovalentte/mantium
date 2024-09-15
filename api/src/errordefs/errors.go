package errordefs

var (
	ErrLastReleasedChapterNotFound = &CustomError{Message: "last released chapter not found"}
	ErrMangaHasNoIDOrURL           = &CustomError{Message: "manga has no ID or URL"}
	ErrMultiMangaHasNoID           = &CustomError{Message: "multimanga has no ID"}
	ErrChapterHasNoChapterOrURL    = &CustomError{Message: "chapter has no chapter or URL"}
	ErrMangaNotFound               = &CustomError{Message: "manga not found in source"}
	ErrMangaNotFoundInMultiManga   = &CustomError{Message: "manga not found in multimanga"}
	ErrChapterNotFound             = &CustomError{Message: "chapter not found in source"}

	ErrMangaNotFoundDB               = &CustomError{Message: "manga not found in DB"}
	ErrMultiMangaNotFoundDB          = &CustomError{Message: "multimanga not found in DB"}
	ErrMangaAlreadyInDB              = &CustomError{Message: "manga already exists in DB"}
	ErrMultiMangaAlreadyInDB         = &CustomError{Message: "multimanga already exists in DB"}
	ErrChapterNotFoundDB             = &CustomError{Message: "chapter not found in DB"}
	ErrAttemptedToDeleteCurrentManga = &CustomError{Message: "attempted to delete the multimanga's current manga"}
)

// CustomError is a custom error
type CustomError struct {
	Message string
}

func (e *CustomError) Error() string {
	return e.Message
}
