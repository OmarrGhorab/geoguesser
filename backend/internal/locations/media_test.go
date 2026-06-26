package locations

import (
	"errors"
	"testing"

	"github.com/raven/geoguess/backend/internal/session"
)

func TestStaticProviderMediaURLRejectsUnsafeProviderRefs(t *testing.T) {
	provider := StaticProvider{}

	tests := []struct {
		name        string
		providerRef string
		wantURL     string
		wantErr     error
	}{
		{name: "https URL", providerRef: "https://media.example/location.jpg", wantURL: "https://media.example/location.jpg"},
		{name: "http URL", providerRef: "http://media.example/location.jpg", wantURL: "http://media.example/location.jpg"},
		{name: "panorama id", providerRef: "CAoSLEFGMVFpcE5fexample", wantErr: ErrMediaUnavailable},
		{name: "relative path", providerRef: "/media/location.jpg", wantErr: ErrMediaUnavailable},
		{name: "javascript URL", providerRef: "javascript:alert(1)", wantErr: ErrMediaUnavailable},
		{name: "empty ref", providerRef: "", wantErr: ErrMediaUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := provider.MediaURL("image", tt.providerRef)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantURL {
				t.Fatalf("url = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestAuthorizeMediaAccessTiers(t *testing.T) {
	service := Service{}
	userID := "0197a000-0000-7000-8000-000000000001"

	tests := []struct {
		name    string
		sess    *session.Context
		access  mapAccess
		wantErr error
	}{
		{
			name: "free allows anonymous",
			sess: &session.Context{Kind: session.KindAnonymous},
			access: mapAccess{
				Visibility:     "public",
				AccessTier:     "free",
				Status:         "active",
				LocationStatus: "active",
			},
		},
		{
			name: "nil session is anonymous",
			access: mapAccess{
				Visibility:     "public",
				AccessTier:     "free",
				Status:         "active",
				LocationStatus: "active",
			},
		},
		{
			name: "premium denies anonymous",
			sess: &session.Context{Kind: session.KindAnonymous},
			access: mapAccess{
				Visibility:     "public",
				AccessTier:     "premium",
				Status:         "active",
				LocationStatus: "active",
			},
			wantErr: ErrMediaAccessDenied,
		},
		{
			name: "premium allows registered",
			sess: &session.Context{Kind: session.KindUser, UserID: &userID},
			access: mapAccess{
				Visibility:     "public",
				AccessTier:     "premium",
				Status:         "active",
				LocationStatus: "active",
			},
		},
		{
			name: "admin denies non-admin",
			sess: &session.Context{Kind: session.KindUser, UserID: &userID},
			access: mapAccess{
				Visibility:     "public",
				AccessTier:     "admin",
				Status:         "active",
				LocationStatus: "active",
			},
			wantErr: ErrMediaAccessDenied,
		},
		{
			name: "admin allows admin",
			sess: &session.Context{Kind: session.KindUser, UserID: &userID, Role: "admin"},
			access: mapAccess{
				Visibility:     "public",
				AccessTier:     "admin",
				Status:         "active",
				LocationStatus: "active",
			},
		},
		{
			name: "private map denied",
			sess: &session.Context{Kind: session.KindUser, UserID: &userID, Role: "admin"},
			access: mapAccess{
				Visibility:     "private",
				AccessTier:     "free",
				Status:         "active",
				LocationStatus: "active",
			},
			wantErr: ErrMediaAccessDenied,
		},
		{
			name: "inactive location denied",
			sess: &session.Context{Kind: session.KindAnonymous},
			access: mapAccess{
				Visibility:     "public",
				AccessTier:     "free",
				Status:         "active",
				LocationStatus: "disabled",
			},
			wantErr: ErrMediaAccessDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.authorizeMedia(tt.sess, &tt.access)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
