package users

// UserStatsResponse is the public stats response for a user.
type UserStatsResponse struct {
	Stats UserStats `json:"stats"`
}

// UserStats contains aggregate gameplay statistics.
type UserStats struct {
	GamesPlayed   int     `json:"games_played"`
	TotalScore    int     `json:"total_score"`
	AverageScore  float64 `json:"average_score"`
	BestScore     int     `json:"best_score"`
}
