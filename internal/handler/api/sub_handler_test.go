package api_test

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/handler/api"
	"github.com/freeway-vpn/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubUserUC struct {
	u   *domain.User
	err error
}

func (s *stubUserUC) Register(ctx context.Context, telegramID int64, username string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (s *stubUserUC) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (s *stubUserUC) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.u, nil
}
func (s *stubUserUC) GetBySubToken(ctx context.Context, token string) (*domain.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.u != nil && s.u.SubToken == token {
		return s.u, nil
	}
	return nil, domain.ErrUserNotFound
}
func (s *stubUserUC) List(ctx context.Context, query string, page, pageSize int) ([]*domain.User, int64, error) {
	return nil, 0, nil
}

type stubConfigUC struct {
	out string
	err error
}

func (s *stubConfigUC) GenerateSubscription(ctx context.Context, userUUID uuid.UUID) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.out, nil
}

func TestSubHandler_GetSubscription_OK(t *testing.T) {
	t.Parallel()
	uid := uuid.New()
	u := &domain.User{ID: uid, SubToken: "tok-abc", TelegramID: 1}
	h := api.NewSubHandler(&stubUserUC{u: u}, &stubConfigUC{out: base64.StdEncoding.EncodeToString([]byte("vless://test"))}, "FreeWay", nil)

	r := chi.NewRouter()
	r.Get("/sub/{token}", h.GetSubscription)

	req := httptest.NewRequest(http.MethodGet, "/sub/tok-abc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	require.NotEmpty(t, body)
	require.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestSubHandler_GetSubscription_NotFound(t *testing.T) {
	t.Parallel()
	h := api.NewSubHandler(&stubUserUC{err: domain.ErrUserNotFound}, &stubConfigUC{}, "FreeWay", nil)

	r := chi.NewRouter()
	r.Get("/sub/{token}", h.GetSubscription)

	req := httptest.NewRequest(http.MethodGet, "/sub/bad", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

var _ usecase.UserUseCase = (*stubUserUC)(nil)
var _ usecase.ConfigUseCase = (*stubConfigUC)(nil)
