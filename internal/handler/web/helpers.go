package web

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func parseUUIDParam(r *http.Request, key string) (uuid.UUID, error) {
	v := chi.URLParam(r, key)
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid param %s: %w", key, err)
	}
	return id, nil
}
