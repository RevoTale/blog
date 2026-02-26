package appcore

import (
	"errors"

	"blog/internal/notes"
)

var errNotesServiceUnavailable = errors.New("notes service unavailable")

type Context struct {
	service *notes.Service
}

func NewContext(service *notes.Service) *Context {
	return &Context{service: service}
}

func IsNotFoundError(err error) bool {
	return errors.Is(err, notes.ErrNotFound)
}
