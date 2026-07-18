package competitive

import (
	"context"
	"time"

	"github.com/google/uuid"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/session"
)

// readStore extends store with profile/leaderboard/season query surfaces.
// *Repository satisfies this; tests may stub a subset.
type readStore interface {
	store
	GetSeason(ctx context.Context, seasonID uuid.UUID) (*Season, error)
	ListSeasons(ctx context.Context, limit int) ([]Season, error)
	GetStanding(ctx context.Context, seasonID, userID uuid.UUID) (*Standing, error)
	LastRatingChange(ctx context.Context, seasonID, userID uuid.UUID) (*RatingChange, error)
	EnsureStandingWithSoftReset(ctx context.Context, season Season, userID uuid.UUID, now time.Time) (*Standing, error)
	EligibleStandingPosition(ctx context.Context, season Season, userID uuid.UUID) (*int, error)
	ListEligibleStandings(ctx context.Context, season Season, limit int, cursor *leaderboardCursor, maxScan int) ([]StandingRow, error)
	ListClosedTop500(ctx context.Context, seasonID uuid.UUID, limit int, cursor *leaderboardCursor) ([]StandingRow, error)
	ListRatingHistory(ctx context.Context, userID uuid.UUID, limit int, cursor *historyCursor) ([]HistoryRow, error)
	UserIsActive(ctx context.Context, userID uuid.UUID) (bool, error)
	RollOverIfDue(ctx context.Context, cfg RolloverConfig, now time.Time) (*RolloverOutcome, error)
}

// Ensure *Repository implements readStore.
var _ readStore = (*Repository)(nil)

// Service fields for reads are set via WithCache / WithRolloverConfig.
func (s *Service) asReadStore() readStore {
	if s == nil || s.repo == nil {
		return nil
	}
	if rs, ok := s.repo.(readStore); ok {
		return rs
	}
	return nil
}

// WithCache attaches a competitive page cache.
func (s *Service) WithCache(cache pageCache) *Service {
	if s == nil {
		return s
	}
	s.cache = cache
	return s
}

// WithRolloverConfig sets season rollover constants.
func (s *Service) WithRolloverConfig(cfg RolloverConfig) *Service {
	if s == nil {
		return s
	}
	s.rollover = normalizeRolloverConfig(cfg)
	return s
}

// InvalidateSeasonCache bumps the cache generation after rating or rollover.
func (s *Service) InvalidateSeasonCache(ctx context.Context, seasonID uuid.UUID) {
	if s == nil || s.cache == nil || seasonID == uuid.Nil {
		return
	}
	if err := s.cache.InvalidateSeason(ctx, seasonID); err != nil && s.logger != nil {
		s.logger.Warn("competitive cache invalidation failed", "error", err.Error())
	}
	s.metrics.ObserveCacheInvalidate("season")
}

// GetProfile returns the caller's active-season competitive profile.
func (s *Service) GetProfile(ctx context.Context, sess session.Context) (*ProfileResponse, error) {
	start := s.now()
	userID, err := requireRegisteredUser(sess)
	if err != nil {
		s.metrics.ObserveProfileRead("unauthorized", 0)
		return nil, err
	}
	rs := s.asReadStore()
	if rs == nil {
		s.metrics.ObserveProfileRead("error", s.now().Sub(start))
		return nil, ErrDependencyFailure
	}

	season, err := rs.ActiveSeason(ctx)
	if err != nil {
		s.metrics.ObserveProfileRead("error", s.now().Sub(start))
		return nil, err
	}
	if season == nil {
		s.metrics.ObserveProfileRead("no_season", s.now().Sub(start))
		return nil, ErrNoActiveSeason
	}

	version := int64(1)
	if s.cache != nil {
		if v, verr := s.cache.Version(ctx, season.ID); verr == nil {
			version = v
		}
		key := ProfileCacheKey(season.ID, userID, version)
		if cached, gerr := s.cache.GetProfile(ctx, key); gerr == nil && cached != nil {
			s.metrics.ObserveProfileRead("cache_hit", s.now().Sub(start))
			return cached, nil
		}
	}

	st, err := rs.EnsureStandingWithSoftReset(ctx, *season, userID, s.now())
	if err != nil {
		s.metrics.ObserveProfileRead("error", s.now().Sub(start))
		return nil, err
	}
	if st == nil {
		s.metrics.ObserveProfileRead("error", s.now().Sub(start))
		return nil, ErrDependencyFailure
	}

	var top500Pos *int
	if EligibleForTop500(st.Rating, st.PlacementsCompleted, st.MatchesPlayed, season.Top500MinMatches) {
		active, aerr := rs.UserIsActive(ctx, userID)
		if aerr != nil {
			s.metrics.ObserveProfileRead("error", s.now().Sub(start))
			return nil, aerr
		}
		if active {
			pos, perr := rs.EligibleStandingPosition(ctx, *season, userID)
			if perr != nil {
				s.metrics.ObserveProfileRead("error", s.now().Sub(start))
				return nil, perr
			}
			top500Pos = pos
		}
	}

	var lastChange *LastChangeDTO
	if change, cerr := rs.LastRatingChange(ctx, season.ID, userID); cerr != nil {
		s.metrics.ObserveProfileRead("error", s.now().Sub(start))
		return nil, cerr
	} else if change != nil {
		lastChange = &LastChangeDTO{MatchID: change.MatchID, Delta: change.TotalDelta}
	}

	resp := &ProfileResponse{
		Season:  SeasonSummary(*season),
		Profile: projectProfileBody(*st, top500Pos, lastChange),
	}

	if s.cache != nil {
		key := ProfileCacheKey(season.ID, userID, version)
		_ = s.cache.SetProfile(ctx, key, resp)
	}
	s.metrics.ObserveProfileRead("ok", s.now().Sub(start))
	return resp, nil
}

func projectProfileBody(st Standing, top500Pos *int, last *LastChangeDTO) ProfileBodyDTO {
	body := ProfileBodyDTO{
		PlacementsCompleted: st.PlacementsCompleted,
		PlacementsRequired:  PlacementsRequired,
		MatchesPlayed:       st.MatchesPlayed,
		Wins:                st.Wins,
		Losses:              st.Losses,
		Draws:               st.Draws,
		Abandons:            st.Abandons,
		LastChange:          last,
	}
	if st.PlacementsCompleted < PlacementsRequired {
		body.Placement = &PlacementProgressDTO{
			Completed: st.PlacementsCompleted,
			Required:  PlacementsRequired,
		}
		// Rating/rank hidden during placements.
		return body
	}
	rating := st.Rating
	peak := st.PeakRating
	progress := DivisionProgress(st.Rating, st.PlacementsCompleted)
	body.Rating = &rating
	body.PeakRating = &peak
	body.DivisionProgress = &progress
	body.StandardRank = StandardRankDetail(st.Rating)
	body.Rank = PresentationRank(st.Rating, st.PlacementsCompleted, top500Pos)
	if top500Pos != nil && IsWorldLegendPosition(*top500Pos) {
		body.Top500Position = top500Pos
	} else if top500Pos != nil {
		// Still expose position when eligible but outside top 500.
		body.Top500Position = top500Pos
	}
	return body
}

// GetLeaderboard returns the active-season eligible leaderboard page.
func (s *Service) GetLeaderboard(ctx context.Context, sess session.Context, limit int, cursor string) (*LeaderboardResponse, error) {
	start := s.now()
	userID, err := requireRegisteredUser(sess)
	if err != nil {
		s.metrics.ObserveLeaderboardRead("unauthorized", 0)
		return nil, err
	}
	rs := s.asReadStore()
	if rs == nil {
		s.metrics.ObserveLeaderboardRead("error", s.now().Sub(start))
		return nil, ErrDependencyFailure
	}
	limit, err = normalizeLeaderboardLimit(limit)
	if err != nil {
		s.metrics.ObserveLeaderboardRead("validation", s.now().Sub(start))
		return nil, err
	}
	if err := validateCursorString(cursor); err != nil {
		s.metrics.ObserveLeaderboardRead("validation", s.now().Sub(start))
		return nil, err
	}
	var lbCursor *leaderboardCursor
	if cursor != "" {
		lbCursor, err = decodeLeaderboardCursor(cursor)
		if err != nil {
			s.metrics.ObserveLeaderboardRead("validation", s.now().Sub(start))
			return nil, ErrInvalidCursor
		}
	}

	season, err := rs.ActiveSeason(ctx)
	if err != nil {
		s.metrics.ObserveLeaderboardRead("error", s.now().Sub(start))
		return nil, err
	}
	if season == nil {
		s.metrics.ObserveLeaderboardRead("no_season", s.now().Sub(start))
		return nil, ErrNoActiveSeason
	}

	version := int64(1)
	if s.cache != nil {
		if v, verr := s.cache.Version(ctx, season.ID); verr == nil {
			version = v
		}
		key := LeaderboardCacheKey(season.ID, version, limit, cursor, userID.String())
		if cached, gerr := s.cache.GetLeaderboard(ctx, key); gerr == nil && cached != nil {
			s.metrics.ObserveLeaderboardRead("cache_hit", s.now().Sub(start))
			return cached, nil
		}
	}

	// Scan at most 501 eligible rows for top-500 correctness budget.
	maxScan := Top500Limit + 1
	rows, err := rs.ListEligibleStandings(ctx, *season, limit+1, lbCursor, maxScan)
	if err != nil {
		s.metrics.ObserveLeaderboardRead("error", s.now().Sub(start))
		return nil, err
	}
	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}

	// Absolute positions: for first page position = index+1; with cursor we recompute via
	// EligibleStandingPosition for the first row then increment. Simpler: when cursor is
	// empty positions are 1..n within the bounded scan; with cursor decode we compute
	// position of first row.
	entries, err := s.projectLeaderboardEntries(ctx, rs, *season, rows, lbCursor)
	if err != nil {
		s.metrics.ObserveLeaderboardRead("error", s.now().Sub(start))
		return nil, err
	}

	viewer, err := s.projectViewer(ctx, rs, *season, userID)
	if err != nil {
		s.metrics.ObserveLeaderboardRead("error", s.now().Sub(start))
		return nil, err
	}

	var nextCursor *string
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		enc := encodeLeaderboardCursor(last.Rating, last.Wins, last.RatingReachedAt, last.UserID)
		nextCursor = &enc
	}

	resp := &LeaderboardResponse{
		Season: SeasonSummary(*season),
		Data:   entries,
		Viewer: viewer,
		Page:   apphttp.PageInfo{Limit: limit, NextCursor: nextCursor},
	}
	if s.cache != nil {
		key := LeaderboardCacheKey(season.ID, version, limit, cursor, userID.String())
		_ = s.cache.SetLeaderboard(ctx, key, resp)
	}
	s.metrics.ObserveLeaderboardRead("ok", s.now().Sub(start))
	return resp, nil
}

func (s *Service) projectLeaderboardEntries(
	ctx context.Context,
	rs readStore,
	season Season,
	rows []StandingRow,
	cursor *leaderboardCursor,
) ([]LeaderboardEntryDTO, error) {
	if len(rows) == 0 {
		return []LeaderboardEntryDTO{}, nil
	}
	startPos := 1
	if cursor != nil {
		// Position of first row among eligible set.
		pos, err := rs.EligibleStandingPosition(ctx, season, rows[0].UserID)
		if err != nil {
			return nil, err
		}
		if pos != nil {
			startPos = *pos
		}
	}
	out := make([]LeaderboardEntryDTO, 0, len(rows))
	for i, row := range rows {
		position := startPos + i
		std := StandardRankDetail(row.Rating)
		wl := IsWorldLegendPosition(position)
		rank := *std
		if wl {
			rank = RankDetailDTO{Code: WorldLegendCode, NameKey: WorldLegendNameKey, Division: nil}
		}
		out = append(out, LeaderboardEntryDTO{
			Position:      position,
			Player:        PublicPlayerDTO{UserID: row.UserID, DisplayName: row.DisplayName},
			Rating:        row.Rating,
			Wins:          row.Wins,
			MatchesPlayed: row.MatchesPlayed,
			StandardRank:  *std,
			WorldLegend:   wl,
			Rank:          rank,
		})
	}
	return out, nil
}

func (s *Service) projectViewer(ctx context.Context, rs readStore, season Season, userID uuid.UUID) (*ViewerEntryDTO, error) {
	st, err := rs.EnsureStandingWithSoftReset(ctx, season, userID, s.now())
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, nil
	}
	viewer := &ViewerEntryDTO{
		Eligible:            false,
		Wins:                st.Wins,
		MatchesPlayed:       st.MatchesPlayed,
		PlacementsCompleted: st.PlacementsCompleted,
		PlacementsRequired:  PlacementsRequired,
	}
	if st.PlacementsCompleted < PlacementsRequired {
		viewer.Placement = &PlacementProgressDTO{
			Completed: st.PlacementsCompleted,
			Required:  PlacementsRequired,
		}
		return viewer, nil
	}
	rating := st.Rating
	viewer.Rating = &rating
	viewer.StandardRank = StandardRankDetail(st.Rating)
	viewer.Rank = PresentationRank(st.Rating, st.PlacementsCompleted, nil)

	if EligibleForTop500(st.Rating, st.PlacementsCompleted, st.MatchesPlayed, season.Top500MinMatches) {
		active, aerr := rs.UserIsActive(ctx, userID)
		if aerr != nil {
			return nil, aerr
		}
		if active {
			pos, perr := rs.EligibleStandingPosition(ctx, season, userID)
			if perr != nil {
				return nil, perr
			}
			viewer.Eligible = true
			viewer.Position = pos
			viewer.WorldLegend = pos != nil && IsWorldLegendPosition(*pos)
			viewer.Rank = PresentationRank(st.Rating, st.PlacementsCompleted, pos)
		}
	}
	return viewer, nil
}

// GetClosedLeaderboard returns a frozen closed-season top-500 leaderboard.
// Active season IDs are served via the live eligible leaderboard shape.
func (s *Service) GetClosedLeaderboard(ctx context.Context, sess session.Context, seasonID uuid.UUID, limit int, cursor string) (*LeaderboardResponse, error) {
	start := s.now()
	userID, err := requireRegisteredUser(sess)
	if err != nil {
		s.metrics.ObserveLeaderboardRead("unauthorized", 0)
		return nil, err
	}
	rs := s.asReadStore()
	if rs == nil {
		s.metrics.ObserveLeaderboardRead("error", s.now().Sub(start))
		return nil, ErrDependencyFailure
	}
	if seasonID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	limit, err = normalizeLeaderboardLimit(limit)
	if err != nil {
		return nil, err
	}
	if err := validateCursorString(cursor); err != nil {
		return nil, err
	}
	var lbCursor *leaderboardCursor
	if cursor != "" {
		lbCursor, err = decodeLeaderboardCursor(cursor)
		if err != nil {
			return nil, ErrInvalidCursor
		}
	}

	season, err := rs.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	if season.Status == SeasonStatusActive {
		// Semantically redirect to the live leaderboard.
		return s.GetLeaderboard(ctx, sess, limit, cursor)
	}
	if season.Status != SeasonStatusClosed {
		return nil, ErrSeasonNotFound
	}

	version := int64(1)
	if s.cache != nil {
		if v, verr := s.cache.Version(ctx, season.ID); verr == nil {
			version = v
		}
		key := ClosedLeaderboardCacheKey(season.ID, version, limit, cursor, userID.String())
		if cached, gerr := s.cache.GetLeaderboard(ctx, key); gerr == nil && cached != nil {
			s.metrics.ObserveLeaderboardRead("cache_hit", s.now().Sub(start))
			return cached, nil
		}
	}

	rows, err := rs.ListClosedTop500(ctx, season.ID, limit+1, lbCursor)
	if err != nil {
		s.metrics.ObserveLeaderboardRead("error", s.now().Sub(start))
		return nil, err
	}
	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}

	entries := make([]LeaderboardEntryDTO, 0, len(rows))
	for _, row := range rows {
		position := 0
		if row.FinalPosition != nil {
			position = *row.FinalPosition
		}
		std := StandardRankDetail(row.Rating)
		wl := IsWorldLegendPosition(position)
		rank := *std
		if row.EndingRankCode != nil && *row.EndingRankCode == WorldLegendCode {
			wl = true
			rank = RankDetailDTO{Code: WorldLegendCode, NameKey: WorldLegendNameKey, Division: nil}
		} else if wl {
			rank = RankDetailDTO{Code: WorldLegendCode, NameKey: WorldLegendNameKey, Division: nil}
		}
		entries = append(entries, LeaderboardEntryDTO{
			Position:      position,
			Player:        PublicPlayerDTO{UserID: row.UserID, DisplayName: row.DisplayName},
			Rating:        row.Rating,
			Wins:          row.Wins,
			MatchesPlayed: row.MatchesPlayed,
			StandardRank:  *std,
			WorldLegend:   wl,
			Rank:          rank,
		})
	}

	// Viewer from frozen standing when present.
	var viewer *ViewerEntryDTO
	if st, serr := rs.GetStanding(ctx, season.ID, userID); serr != nil {
		return nil, serr
	} else if st != nil {
		viewer = &ViewerEntryDTO{
			Position:            st.FinalPosition,
			Eligible:            st.FinalPosition != nil,
			Wins:                st.Wins,
			MatchesPlayed:       st.MatchesPlayed,
			PlacementsCompleted: st.PlacementsCompleted,
			PlacementsRequired:  PlacementsRequired,
		}
		if st.RankVisible() {
			rating := st.Rating
			viewer.Rating = &rating
			viewer.StandardRank = StandardRankDetail(st.Rating)
			viewer.Rank = RankDetailFromCode(ptrString(st.EndingRankCode))
			if st.EndingRankCode != nil && *st.EndingRankCode == WorldLegendCode {
				viewer.WorldLegend = true
			}
			if viewer.Rank == nil {
				viewer.Rank = PresentationRank(st.Rating, st.PlacementsCompleted, st.FinalPosition)
			}
		} else {
			viewer.Placement = &PlacementProgressDTO{Completed: st.PlacementsCompleted, Required: PlacementsRequired}
		}
	}

	var nextCursor *string
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		enc := encodeLeaderboardCursor(last.Rating, last.Wins, last.RatingReachedAt, last.UserID)
		nextCursor = &enc
	}

	resp := &LeaderboardResponse{
		Season: SeasonSummary(*season),
		Data:   entries,
		Viewer: viewer,
		Page:   apphttp.PageInfo{Limit: limit, NextCursor: nextCursor},
	}
	if s.cache != nil {
		key := ClosedLeaderboardCacheKey(season.ID, version, limit, cursor, userID.String())
		_ = s.cache.SetLeaderboard(ctx, key, resp)
	}
	s.metrics.ObserveLeaderboardRead("ok", s.now().Sub(start))
	return resp, nil
}

// GetHistory returns the caller's rating history newest first.
func (s *Service) GetHistory(ctx context.Context, sess session.Context, limit int, cursor string) (*HistoryResponse, error) {
	userID, err := requireRegisteredUser(sess)
	if err != nil {
		return nil, err
	}
	rs := s.asReadStore()
	if rs == nil {
		return nil, ErrDependencyFailure
	}
	limit, err = normalizeHistoryLimit(limit)
	if err != nil {
		return nil, err
	}
	if err := validateCursorString(cursor); err != nil {
		return nil, err
	}
	var hCursor *historyCursor
	if cursor != "" {
		hCursor, err = decodeHistoryCursor(cursor)
		if err != nil {
			return nil, ErrInvalidCursor
		}
	}
	rows, err := rs.ListRatingHistory(ctx, userID, limit+1, hCursor)
	if err != nil {
		return nil, err
	}
	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}
	entries := make([]HistoryEntryDTO, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, projectHistoryEntry(row))
	}
	var nextCursor *string
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		enc := encodeHistoryCursor(last.CreatedAt, last.ID)
		nextCursor = &enc
	}
	return &HistoryResponse{
		Data: entries,
		Page: apphttp.PageInfo{Limit: limit, NextCursor: nextCursor},
	}, nil
}

func projectHistoryEntry(row HistoryRow) HistoryEntryDTO {
	dto := HistoryEntryDTO{
		ID:        row.ID,
		MatchID:   row.MatchID,
		SeasonID:  row.SeasonID,
		Playlist:  row.Playlist,
		Format:    row.Format,
		Outcome:   row.Outcome,
		CreatedAt: row.CreatedAt.UTC(),
	}
	// Hide numeric rating fields when rank codes are null (still placing).
	// History after placements reveal always stores rank codes when visible.
	if row.OldRankCode == nil && row.NewRankCode == nil {
		// Still may be post-placement if both null unexpectedly; expose when either rank present.
		// Prefer revealing numbers when the change itself had visible ranks at write time.
		// For placement games both codes are null — hide deltas numbers is optional; contract says
		// old/new rating with rank transition. During placements, hide rating fields.
		return dto
	}
	oldR, newR := row.OldRating, row.NewRating
	base, pen, total := row.BaseDelta, row.AbandonPenalty, row.TotalDelta
	dto.OldRating = &oldR
	dto.NewRating = &newR
	dto.BaseDelta = &base
	dto.AbandonPenalty = &pen
	dto.TotalDelta = &total
	if row.OldRankCode != nil {
		dto.OldRank = &RankDTO{Code: *row.OldRankCode}
	}
	if row.NewRankCode != nil {
		dto.NewRank = &RankDTO{Code: *row.NewRankCode}
	}
	return dto
}

// ListSeasons returns active and closed season summaries with the caller's frozen facts.
func (s *Service) ListSeasons(ctx context.Context, sess session.Context) (*SeasonsResponse, error) {
	userID, err := requireRegisteredUser(sess)
	if err != nil {
		return nil, err
	}
	rs := s.asReadStore()
	if rs == nil {
		return nil, ErrDependencyFailure
	}
	seasons, err := rs.ListSeasons(ctx, 50)
	if err != nil {
		return nil, err
	}
	out := make([]SeasonListEntryDTO, 0, len(seasons))
	for _, season := range seasons {
		entry := SeasonListEntryDTO{Season: SeasonSummary(season)}
		if st, serr := rs.GetStanding(ctx, season.ID, userID); serr != nil {
			return nil, serr
		} else if st != nil {
			entry.FinalPosition = st.FinalPosition
			entry.EndingRank = RankDetailFromCode(ptrString(st.EndingRankCode))
			entry.PeakRank = RankDetailFromCode(ptrString(st.PeakRankCode))
		}
		out = append(out, entry)
	}
	return &SeasonsResponse{Data: out}, nil
}

// RollOverIfDue runs season closure when the active season has ended.
func (s *Service) RollOverIfDue(ctx context.Context) (*RolloverOutcome, error) {
	rs := s.asReadStore()
	if rs == nil {
		return nil, ErrDependencyFailure
	}
	start := s.now()
	out, err := rs.RollOverIfDue(ctx, s.rollover, start)
	duration := s.now().Sub(start)
	if err != nil {
		s.metrics.ObserveRollover("error", 0, duration)
		return nil, err
	}
	if out == nil {
		s.metrics.ObserveRollover("skipped", 0, duration)
		return &RolloverOutcome{Skipped: true}, nil
	}
	switch {
	case out.Applied:
		seq := 0
		if out.NewSeason != nil {
			seq = out.NewSeason.Sequence
		}
		s.metrics.ObserveRollover("applied", seq, duration)
		if out.ClosedSeason != nil {
			s.InvalidateSeasonCache(ctx, out.ClosedSeason.ID)
		}
		if out.NewSeason != nil {
			s.InvalidateSeasonCache(ctx, out.NewSeason.ID)
		}
	case out.Replay:
		seq := 0
		if out.NewSeason != nil {
			seq = out.NewSeason.Sequence
		}
		s.metrics.ObserveRollover("replay", seq, duration)
	default:
		s.metrics.ObserveRollover("skipped", 0, duration)
	}
	return out, nil
}

func requireRegisteredUser(sess session.Context) (uuid.UUID, error) {
	if !sess.IsRegistered() || sess.UserID == nil {
		return uuid.Nil, ErrUnauthorized
	}
	id, err := uuid.Parse(*sess.UserID)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrUnauthorized
	}
	return id, nil
}

func ptrString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
