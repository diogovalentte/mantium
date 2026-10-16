package errordefs

var (
	ErrMangaNotFound           = &CustomError{Message: "manga not found in source"}
	ErrMangaAttributesNotFound = &CustomError{Message: "one of the manga attributes were not found in source"}
	ErrMangaHasNoIDOrURL       = &CustomError{Message: "manga has no ID or URL"}

	ErrChapterNotFound           = &CustomError{Message: "chapter or one of its attributes not found in source"}
	ErrChapterAttributesNotFound = &CustomError{Message: "one of the chapter attributes were not found in source"}
	ErrChapterHasNoChapterOrURL  = &CustomError{Message: "chapter has no chapter or URL"}

	ErrMangaNotFoundInMultiManga            = &CustomError{Message: "manga not found in multimanga"}
	ErrMangaNotFoundDB                      = &CustomError{Message: "manga not found in DB"}
	ErrMultiMangaNotFoundDB                 = &CustomError{Message: "multimanga not found in DB"}
	ErrMangaAlreadyInDB                     = &CustomError{Message: "manga already exists in DB"}
	ErrMultiMangaAlreadyInDB                = &CustomError{Message: "multimanga already exists in DB"}
	ErrChapterNotFoundDB                    = &CustomError{Message: "chapter not found in DB"}
	ErrAttemptedToRemoveLastMultiMangaManga = &CustomError{Message: "attempted to remove the last manga from a multimanga"}
	ErrMultiMangaMangaListIsEmpty           = &CustomError{Message: "multimanga manga list is empty"}

	ErrNoLastReleasedChapter = &UserError{Message: "The manga has no last released chapter to mark as read"}
	ErrNoChapterToIgnore     = &UserError{Message: "The manga has no last released chapter to ignore"}
	ErrNoMangaLeftToFollow   = &UserError{Message: "Can't ignore this chapter: no other manga in the multimanga has a last released chapter that isn't ignored"}
)

// CustomError is a custom error
type CustomError struct {
	Message string
}

func (e *CustomError) Error() string {
	return e.Message
}

// UserError is an error caused by the user's input. Its Message is meant to be shown to the user as is.
type UserError struct {
	Message string
	Err     error
}

func (e *UserError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *UserError) Unwrap() error {
	return e.Err
}
