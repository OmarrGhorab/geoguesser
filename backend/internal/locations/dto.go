package locations

// MediaType values.
const (
	MediaTypeImage    = "image"
	MediaTypePanorama = "panorama"
)

// RoundMedia is a single location media payload without hidden coordinates.
type RoundMedia struct {
	Type        string  `json:"type"`
	URL         string  `json:"url"`
	Attribution *string `json:"attribution,omitempty"`
}

// RoundMediaResponse is the response for GET /locations/{locationId}/media.
type RoundMediaResponse struct {
	Media RoundMedia `json:"media"`
}
