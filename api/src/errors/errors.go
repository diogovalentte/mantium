package errors

var ErrLastReleasedChapterNotFound = &CustomError{Message: "last released chapter not found"}

// CustomError is a custom error
type CustomError struct {
	Message string
}

func (e *CustomError) Error() string {
	return e.Message
}
