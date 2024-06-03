package errordefs

var (
	ErrLastReleasedChapterNotFound    = &CustomError{Message: "last released chapter not found"}
	ErrMangaDoesntHaveIDAndURL        = &CustomError{Message: "manga doesn't have and ID or URL"}
	ErrChapterDoesntHaveChapterAndURL = &CustomError{Message: "chapter doesn't have a chapter or URL"}
	ErrMangaNotFound                  = &CustomError{Message: "manga not found in source"}
	ErrChapterNotFound                = &CustomError{Message: "chapter not found in source"}

	ErrMangaNotFoundDB   = &CustomError{Message: "manga not found in DB"}
	ErrMangaAlreadyInDB  = &CustomError{Message: "manga already exists in DB"}
	ErrChapterNotFoundDB = &CustomError{Message: "chapter not found in DB"}
)

// CustomError is a custom error
type CustomError struct {
	Message string
}

func (e *CustomError) Error() string {
	return e.Message
}
