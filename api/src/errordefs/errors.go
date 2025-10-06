package errordefs

var (
	ErrLastReleasedChapterNotFound = &CustomError{Message: "last released chapter not found"}
	ErrMangaHasNoIDOrURL           = &CustomError{Message: "manga has no ID or URL"}
	ErrChapterHasNoChapterOrURL    = &CustomError{Message: "chapter has no chapter or URL"}
	ErrMangaNotFound               = &CustomError{Message: "manga not found in source"}
	ErrMangaNotFoundInMultiManga   = &CustomError{Message: "manga not found in multimanga"}
	ErrChapterNotFound             = &CustomError{Message: "chapter not found in source"}
	ErrChapterListNotFound         = &CustomError{Message: "chapter list not found in source"}
	ErrChapterURLNotFound          = &CustomError{Message: "chapter URL not found"}
	ErrMangaURLNotFound            = &CustomError{Message: "manga URL not found"}

	ErrMangaNotFoundDB                      = &CustomError{Message: "manga not found in DB"}
	ErrMultiMangaNotFoundDB                 = &CustomError{Message: "multimanga not found in DB"}
	ErrMangaAlreadyInDB                     = &CustomError{Message: "manga already exists in DB"}
	ErrMultiMangaAlreadyInDB                = &CustomError{Message: "multimanga already exists in DB"}
	ErrChapterNotFoundDB                    = &CustomError{Message: "chapter not found in DB"}
	ErrAttemptedToRemoveLastMultiMangaManga = &CustomError{Message: "attempted to remove the last manga from a multimanga"}
	ErrMultiMangaMangaListIsEmpty           = &CustomError{Message: "multimanga manga list is empty"}
)

// CustomError is a custom error
type CustomError struct {
	Message string
}

func (e *CustomError) Error() string {
	return e.Message
}
