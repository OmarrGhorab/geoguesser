package locations

import (
	"net/url"
	"regexp"
	"strings"
)

var panoramaIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,512}$`)

// Provider resolves a location's provider and provider_ref into a playable media URL.
type Provider interface {
	MediaURL(provider, providerRef string) (string, error)
}

// StaticProvider resolves provider_ref only when it is already a public
// HTTP(S) URL. Raw provider IDs such as panorama IDs stay server-side.
type StaticProvider struct{}

// MediaURL returns provider_ref as the media URL after URL validation.
func (StaticProvider) MediaURL(provider, providerRef string) (string, error) {
	parsed, err := url.ParseRequestURI(providerRef)
	if err != nil || parsed.Host == "" {
		return "", ErrMediaUnavailable
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return providerRef, nil
	default:
		return "", ErrMediaUnavailable
	}
}

// MediaType infers the media type from the provider name.
func MediaType(provider string) string {
	switch strings.ToLower(provider) {
	case "streetview", "google_street_view", "panorama", "mapillary":
		return MediaTypePanorama
	default:
		return MediaTypeImage
	}
}

// PanoramaID returns a client-playable opaque panorama identifier for known
// panorama providers. It never exposes coordinates or arbitrary provider data.
func PanoramaID(provider, providerRef string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "streetview", "google_street_view", "panorama":
		value := strings.TrimSpace(providerRef)
		if panoramaIDPattern.MatchString(value) {
			return value, true
		}
	}
	return "", false
}
