package parties_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/parties"
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

func TestHandlerCreateRequiresAuth(t *testing.T) {
	store := newMemStore()
	h := parties.NewHandler(parties.NewService(store, &memFriends{}, nil), nil)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/parties", strings.NewReader(`{"format":"duo"}`), &session.Context{Kind: session.KindGuest})
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "k1")
	h.CreateParty(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerCreateStrictJSON(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	store.seedUser(leader, "L")
	h := parties.NewHandler(parties.NewService(store, &memFriends{}, nil), nil)
	sess := registered(leader)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/parties", strings.NewReader(`{"format":"duo","extra":true}`), &sess)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "k1")
	h.CreateParty(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerCreateSuccessAndCurrent(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	store.seedUser(leader, "Leader")
	h := parties.NewHandler(parties.NewService(store, &memFriends{}, nil), nil)
	sess := registered(leader)

	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/parties", strings.NewReader(`{"format":"duo"}`), &sess)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "create-1")
	h.CreateParty(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var created parties.PartyResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Party.Format != "duo" {
		t.Fatalf("format = %s", created.Party.Format)
	}

	rec2 := httptest.NewRecorder()
	req2 := requestWithSession(http.MethodGet, "/parties/current", nil, &sess)
	h.GetCurrentParty(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("current status = %d", rec2.Code)
	}
	var cur parties.CurrentPartyResponse
	if err := json.NewDecoder(rec2.Body).Decode(&cur); err != nil {
		t.Fatalf("decode current: %v", err)
	}
	if cur.Party == nil || cur.Party.ID != created.Party.ID {
		t.Fatalf("current = %+v", cur)
	}
}

func TestHandlerFriendUnavailablePrivacy(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	stranger := uuid.New()
	store.seedUser(leader, "L")
	store.seedUser(stranger, "S")
	svc := parties.NewService(store, &memFriends{}, nil)
	created, err := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "c1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	h := parties.NewHandler(svc, nil)
	sess := registered(leader)
	body := `{"user_id":"` + stranger.String() + `"}`
	// Use router for path params
	router := chi.NewRouter()
	h.RegisterRoutes(router)
	req := requestWithSession(http.MethodPost, "/parties/"+created.Party.ID.String()+"/invites", strings.NewReader(body), &sess)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "inv-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "friend_unavailable") {
		t.Fatalf("expected friend_unavailable, body=%s", rec.Body.String())
	}
	if strings.Contains(strings.ToLower(rec.Body.String()), "block") {
		t.Fatalf("must not mention block: %s", rec.Body.String())
	}
}

func TestHandlerIdempotencyRequired(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	store.seedUser(leader, "L")
	h := parties.NewHandler(parties.NewService(store, &memFriends{}, nil), nil)
	sess := registered(leader)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/parties", strings.NewReader(`{"format":"duo"}`), &sess)
	req.Header.Set("Content-Type", "application/json")
	// no Idempotency-Key
	h.CreateParty(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRegisterRoutes(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	store.seedUser(leader, "L")
	policy := &memFriends{}
	svc := parties.NewService(store, policy, nil)
	h := parties.NewHandler(svc, nil)
	router := chi.NewRouter()
	// Session middleware for all routes
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := registered(leader)
			ctx := context.WithValue(r.Context(), struct{ name string }{"session"}, &s)
			// properly inject via SessionLoader
			appmiddleware.SessionLoader(staticSessionResolver{session: &s}, "a", "g")(next).ServeHTTP(w, r.WithContext(ctx))
		})
	})
	h.RegisterRoutes(router)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/parties", strings.NewReader(`{"format":"squad"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "route-1")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/parties/current", nil)
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("current status = %d", rec2.Code)
	}
}

func TestHandlerRateLimitObserver(t *testing.T) {
	store := newMemStore()
	h := parties.NewHandler(parties.NewService(store, &memFriends{}, nil), nil)
	// Should not panic with nil metrics
	req := httptest.NewRequest(http.MethodPost, "/parties", nil)
	h.RecordRateLimited(req)
}

func TestHandlerInvalidJSON(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	store.seedUser(leader, "L")
	h := parties.NewHandler(parties.NewService(store, &memFriends{}, nil), nil)
	sess := registered(leader)
	rec := httptest.NewRecorder()
	req := requestWithSession(http.MethodPost, "/parties", strings.NewReader(`{`), &sess)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "k")
	h.CreateParty(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}
