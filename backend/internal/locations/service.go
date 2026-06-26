package locations

import (
	"context"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/session"
)

// Service implements location business logic.
type Service struct {
	repo     *Repository
	provider Provider
}

// NewService returns a new locations service.
func NewService(repo *Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
}

// GetLocationMedia returns the media for a location when the requester is allowed
// to view it. Coordinates and provider-hidden metadata are never returned.
func (s *Service) GetLocationMedia(ctx context.Context, sess *session.Context, locationID string) (*RoundMediaResponse, error) {
	id, err := uuid.Parse(locationID)
	if err != nil {
		return nil, ErrInvalidLocationID
	}

	location, err := s.repo.GetLocationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if location == nil {
		return nil, ErrLocationNotFound
	}

	accessRows, err := s.repo.ListMapAccessForLocation(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(accessRows) == 0 {
		return nil, ErrMediaAccessDenied
	}

	if err := s.authorizeAnyMedia(sess, accessRows); err != nil {
		return nil, err
	}

	url, err := s.provider.MediaURL(location.Provider, location.ProviderRef)
	if err != nil {
		return nil, err
	}

	media := RoundMedia{
		Type:        MediaType(location.Provider),
		URL:         url,
		Attribution: location.Attribution,
	}
	return &RoundMediaResponse{Media: media}, nil
}

func (s *Service) authorizeAnyMedia(sess *session.Context, accessRows []mapAccess) error {
	for i := range accessRows {
		if err := s.authorizeMedia(sess, &accessRows[i]); err == nil {
			return nil
		}
	}
	return ErrMediaAccessDenied
}

func (s *Service) authorizeMedia(sess *session.Context, access *mapAccess) error {
	if sess == nil {
		sess = &session.Context{Kind: session.KindAnonymous}
	}
	if access.Status != "active" || access.LocationStatus != "active" {
		return ErrMediaAccessDenied
	}
	if access.Visibility != "public" {
		return ErrMediaAccessDenied
	}

	switch access.AccessTier {
	case "free":
		return nil
	case "premium":
		if !sess.IsRegistered() {
			return ErrMediaAccessDenied
		}
		return nil
	case "admin":
		if !sess.IsAdmin() {
			return ErrMediaAccessDenied
		}
		return nil
	default:
		return ErrMediaAccessDenied
	}
}
