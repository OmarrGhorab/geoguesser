// Package home composes the bounded read model needed by the authenticated
// home screen. It owns no underlying gameplay, profile, or challenge state.
package home

import (
	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/challenges"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/profiles"
)

// Response is the coherent initial snapshot for an authenticated home screen.
type Response struct {
	Viewer          ViewerDTO                            `json:"viewer"`
	Stats           profiles.StatsDTO                    `json:"stats"`
	DailyChallenge  challenges.ChallengeMetadataResponse `json:"daily_challenge"`
	RecommendedMaps []maps.MapDTO                        `json:"recommended_maps"`
}

// ViewerDTO is the minimum public-safe identity projection required by home.
type ViewerDTO struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	CountryCode *string   `json:"country_code,omitempty"`
}

func viewerDTO(profile *profiles.PublicProfileSummary) ViewerDTO {
	return ViewerDTO{
		UserID:      profile.UserID,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		CountryCode: profile.CountryCode,
	}
}

func statsDTO(stats *profiles.StatsSummary) profiles.StatsDTO {
	if stats == nil {
		return profiles.StatsDTO{}
	}
	return profiles.StatsDTO{
		GamesPlayed:  stats.GamesPlayed,
		TotalScore:   stats.TotalScore,
		AverageScore: stats.AverageScore,
		BestScore:    stats.BestScore,
		LastPlayedAt: stats.LastPlayedAt,
	}
}
