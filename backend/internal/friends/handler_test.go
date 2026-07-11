package friends_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/friends"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/session"
)

type staticSessionResolver struct {
	session *session.Context
}

func (r staticSessionResolver) ResolveSession(context.Context, string) (*session.Context, error) {
	if r.session == nil {
		return &session.Context{Kind: session.KindAnonymous}, nil
	}
	return r.session, nil
}

func (staticSessionResolver) ResolveGuestSession(context.Context, string) (string, error) {
	return "", nil
}

func requestWithSession(method, target string, body io.Reader, sess *session.Context) *http.Request {
	req := httptest.NewRequest(method, target, body)
	recorder := httptest.NewRecorder()
	appmiddleware.SessionLoader(staticSessionResolver{session: sess}, "access_token", "guest_session")(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		req = r
	})).ServeHTTP(recorder, req)
	return req
}

func TestHandlerCreateRequestRequiresAuth(t *testing.T) {
	h := friends.NewHandler(friends.NewService(&fakeStore{}, nil), nil)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/friends/requests", strings.NewReader(`{"user_id":"`+uuid.New().String()+`"}`), &session.Context{Kind: session.KindGuest})
	req.Header.Set("Content-Type", "application/json")
	h.CreateRequest(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerCreateRequestInvalidJSON(t *testing.T) {
	actor := uuid.New()
	sess := registered(actor)
	h := friends.NewHandler(friends.NewService(&fakeStore{}, nil), nil)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/friends/requests", strings.NewReader(`{`), &sess)
	req.Header.Set("Content-Type", "application/json")
	h.CreateRequest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid json status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerCreateRequestSuccess(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	reqID := uuid.New()
	now := time.Now().UTC()
	store := &fakeStore{
		createFn: func(context.Context, uuid.UUID, uuid.UUID) (*friends.Friendship, error) {
			return &friends.Friendship{ID: reqID, Status: friends.StatusPending, CreatedAt: now}, nil
		},
		profiles: map[uuid.UUID]*friends.PublicProfile{
			target: {UserID: target, DisplayName: "Target"},
		},
	}
	h := friends.NewHandler(friends.NewService(store, nil), nil)
	sess := registered(actor)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/friends/requests", strings.NewReader(`{"user_id":"`+target.String()+`"}`), &sess)
	req.Header.Set("Content-Type", "application/json")
	h.CreateRequest(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload friends.RequestResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Data.RequestID != reqID {
		t.Fatalf("request id = %s", payload.Data.RequestID)
	}
	body := rec.Body.String()
	if strings.Contains(strings.ToLower(body), "email") || strings.Contains(body, "password") || strings.Contains(body, "blocked_by") {
		t.Fatalf("privacy leak in body: %s", body)
	}
}

func TestHandlerDeclineNoContent(t *testing.T) {
	actor := uuid.New()
	reqID := uuid.New()
	store := &fakeStore{
		declineFn: func(context.Context, uuid.UUID, uuid.UUID) error { return nil },
	}
	h := friends.NewHandler(friends.NewService(store, nil), nil)
	sess := registered(actor)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/friends/requests/"+reqID.String()+"/decline", nil, &sess)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("requestId", reqID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	h.DeclineRequest(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlerListFriendsPaginationValidation(t *testing.T) {
	actor := uuid.New()
	sess := registered(actor)
	h := friends.NewHandler(friends.NewService(&fakeStore{}, nil), nil)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodGet, "/friends?limit=999", nil, &sess)
	h.ListFriends(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerConflictMapping(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	store := &fakeStore{
		createFn: func(context.Context, uuid.UUID, uuid.UUID) (*friends.Friendship, error) {
			return nil, friends.ErrAlreadyFriends
		},
	}
	h := friends.NewHandler(friends.NewService(store, nil), nil)
	sess := registered(actor)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/friends/requests", strings.NewReader(`{"user_id":"`+target.String()+`"}`), &sess)
	req.Header.Set("Content-Type", "application/json")
	h.CreateRequest(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerNotFoundMapping(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	store := &fakeStore{
		createFn: func(context.Context, uuid.UUID, uuid.UUID) (*friends.Friendship, error) {
			return nil, friends.ErrTargetNotFound
		},
	}
	h := friends.NewHandler(friends.NewService(store, nil), nil)
	sess := registered(actor)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/friends/requests", strings.NewReader(`{"user_id":"`+target.String()+`"}`), &sess)
	req.Header.Set("Content-Type", "application/json")
	h.CreateRequest(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerBlockUnblockNoContent(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	h := friends.NewHandler(friends.NewService(&fakeStore{}, nil), nil)
	sess := registered(actor)

	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/friends/"+target.String()+"/block", nil, &sess)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", target.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	h.BlockUser(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("block status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = requestWithSession(http.MethodDelete, "/friends/"+target.String()+"/block", nil, &sess)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("userId", target.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	h.UnblockUser(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unblock status = %d", rec.Code)
	}
}
