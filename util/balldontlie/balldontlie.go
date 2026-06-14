package balldontlie

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/useryege/athena/util/ratelimit"
)

const (
	DefaultBaseURL           = "https://api.balldontlie.io"
	DefaultTimeout           = 30 * time.Second
	DefaultRateLimitRequests = 5
	DefaultRateLimitPeriod   = time.Minute

	errorBodyLimit = 4096
)

type Client interface {
	ListPlayers(ctx context.Context, options ListPlayersOptions) (*ListPlayersResponse, error)
	GetPlayer(ctx context.Context, id int) (*ATPPlayer, error)
	ListTournaments(ctx context.Context, options ListTournamentsOptions) (*ListTournamentsResponse, error)
	GetTournament(ctx context.Context, id int, options GetTournamentOptions) (*ATPTournament, error)
	ListRankings(ctx context.Context, options ListRankingsOptions) (*ListRankingsResponse, error)
	ListMatches(ctx context.Context, options ListMatchesOptions) (*ListMatchesResponse, error)
	GetMatch(ctx context.Context, id int) (*ATPMatch, error)
	ListATPRace(ctx context.Context, options ListATPRaceOptions) (*ListATPRaceResponse, error)
	ListMatchStats(ctx context.Context, options ListMatchStatsOptions) (*ListMatchStatsResponse, error)
	ListPlayerCareerStats(ctx context.Context, options ListPlayerCareerStatsOptions) (*ListPlayerCareerStatsResponse, error)
	GetHeadToHead(ctx context.Context, options HeadToHeadOptions) (*ATPHeadToHead, error)
	ListOdds(ctx context.Context, options ListOddsOptions) (*ListOddsResponse, error)
}

type Config struct {
	BaseURL     string
	APIKey      string
	Timeout     time.Duration
	RateLimiter ratelimit.Limiter
}

type clientImpl struct {
	config      Config
	http        *http.Client
	rateLimiter ratelimit.Limiter
}

var _ Client = (*clientImpl)(nil)

func NewClient(config Config) (Client, error) {
	config = config.withDefaults()
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, errors.New("balldontlie api key is required")
	}
	if _, err := url.ParseRequestURI(config.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid balldontlie base url: %w", err)
	}
	rateLimiter := config.RateLimiter
	if rateLimiter == nil {
		var err error
		rateLimiter, err = ratelimit.New(ratelimit.Config{
			Requests: DefaultRateLimitRequests,
			Per:      DefaultRateLimitPeriod,
			Burst:    1,
		})
		if err != nil {
			return nil, fmt.Errorf("create balldontlie rate limiter: %w", err)
		}
	}
	return &clientImpl{
		config:      config,
		http:        &http.Client{Timeout: config.Timeout},
		rateLimiter: rateLimiter,
	}, nil
}

func (c Config) WithDefaults() Config {
	return c.withDefaults()
}

func (c Config) withDefaults() Config {
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.APIKey = strings.TrimSpace(c.APIKey)
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}

type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"error,omitempty"`
	RawBody    string `json:"-"`
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return fmt.Sprintf("balldontlie request failed with status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("balldontlie request failed with status %d", e.StatusCode)
}

type PageOptions struct {
	Cursor  *int
	PerPage *int
}

type ListPlayersOptions struct {
	PageOptions
	PlayerIDs   []int
	Search      string
	FirstName   string
	LastName    string
	Country     string
	CountryCode string
}

type ListTournamentsOptions struct {
	PageOptions
	TournamentIDs []int
	Season        *int
	Surface       string
	Category      string
}

type GetTournamentOptions struct {
	Season *int
}

type ListRankingsOptions struct {
	PageOptions
	PlayerIDs []int
	Date      string
}

type ListMatchesOptions struct {
	PageOptions
	TournamentIDs []int
	PlayerIDs     []int
	Season        *int
	Round         string
	IsLive        *bool
}

type ListATPRaceOptions struct {
	PageOptions
	PlayerIDs []int
	Date      string
}

type ListMatchStatsOptions struct {
	PageOptions
	MatchIDs  []int
	PlayerIDs []int
	SetNumber *int
}

type ListPlayerCareerStatsOptions struct {
	PageOptions
	PlayerIDs []int
}

type HeadToHeadOptions struct {
	Player1ID int
	Player2ID int
}

type ListOddsOptions struct {
	PageOptions
	MatchIDs      []int
	TournamentIDs []int
	PlayerIDs     []int
	Season        *int
}

type Pagination struct {
	NextCursor *int `json:"next_cursor,omitempty"`
	PrevCursor *int `json:"prev_cursor,omitempty"`
	PerPage    int  `json:"per_page,omitempty"`
}

type ListPlayersResponse struct {
	Data []ATPPlayer `json:"data"`
	Meta Pagination  `json:"meta"`
}

type ListTournamentsResponse struct {
	Data []ATPTournament `json:"data"`
	Meta Pagination      `json:"meta"`
}

type ListRankingsResponse struct {
	Data []ATPRanking `json:"data"`
	Meta Pagination   `json:"meta"`
}

type ListMatchesResponse struct {
	Data []ATPMatch `json:"data"`
	Meta Pagination `json:"meta"`
}

type ListATPRaceResponse struct {
	Data []ATPRace  `json:"data"`
	Meta Pagination `json:"meta"`
}

type ListMatchStatsResponse struct {
	Data []ATPMatchStats `json:"data"`
	Meta Pagination      `json:"meta"`
}

type ListPlayerCareerStatsResponse struct {
	Data []ATPPlayerCareerStats `json:"data"`
	Meta Pagination             `json:"meta"`
}

type ListOddsResponse struct {
	Data []ATPBettingOdd `json:"data"`
	Meta Pagination      `json:"meta"`
}

type ATPPlayer struct {
	ID          int     `json:"id"`
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	FullName    *string `json:"full_name"`
	Country     *string `json:"country"`
	CountryCode *string `json:"country_code"`
	BirthPlace  *string `json:"birth_place"`
	Age         *int    `json:"age"`
	HeightCM    *int    `json:"height_cm"`
	WeightKG    *int    `json:"weight_kg"`
	Plays       *string `json:"plays"`
	TurnedPro   *int    `json:"turned_pro"`
}

type ATPTournament struct {
	ID            int     `json:"id"`
	Name          *string `json:"name"`
	Location      *string `json:"location"`
	Surface       *string `json:"surface"`
	Category      *string `json:"category"`
	Season        *int    `json:"season"`
	StartDate     *string `json:"start_date"`
	EndDate       *string `json:"end_date"`
	PrizeMoney    *int    `json:"prize_money"`
	PrizeCurrency *string `json:"prize_currency"`
	DrawSize      *int    `json:"draw_size"`
}

type ATPRanking struct {
	ID          int       `json:"id"`
	Player      ATPPlayer `json:"player"`
	Rank        int       `json:"rank"`
	Points      int       `json:"points"`
	Movement    *int      `json:"movement"`
	RankingDate string    `json:"ranking_date"`
}

type ATPRace struct {
	ID          int       `json:"id"`
	Player      ATPPlayer `json:"player"`
	RankingDate string    `json:"ranking_date"`
	Rank        int       `json:"rank"`
	Points      int       `json:"points"`
	Movement    int       `json:"movement"`
	IsQualified bool      `json:"is_qualified"`
}

type ATPMatch struct {
	ID               int           `json:"id"`
	Tournament       ATPTournament `json:"tournament"`
	Season           int           `json:"season"`
	Round            *string       `json:"round"`
	Player1          ATPPlayer     `json:"player1"`
	Player2          ATPPlayer     `json:"player2"`
	Winner           *ATPPlayer    `json:"winner"`
	Score            *string       `json:"score"`
	SetScores        []ATPSetScore `json:"set_scores"`
	Player1GameScore *string       `json:"player1_game_score"`
	Player2GameScore *string       `json:"player2_game_score"`
	Server           *int          `json:"server"`
	Duration         *string       `json:"duration"`
	NumberOfSets     *int          `json:"number_of_sets"`
	MatchStatus      *string       `json:"match_status"`
	IsLive           bool          `json:"is_live"`
}

type ATPSetScore struct {
	SetNumber       int  `json:"set_number"`
	Player1Games    int  `json:"player1_games"`
	Player2Games    int  `json:"player2_games"`
	Player1Tiebreak *int `json:"player1_tiebreak"`
	Player2Tiebreak *int `json:"player2_tiebreak"`
}

type ATPMatchStats struct {
	ID                       int                `json:"id"`
	Match                    ATPMatchStatsMatch `json:"match"`
	Player                   ATPPlayer          `json:"player"`
	SetNumber                *int               `json:"set_number"`
	ServeRating              *int               `json:"serve_rating"`
	Aces                     *int               `json:"aces"`
	DoubleFaults             *int               `json:"double_faults"`
	FirstServePct            *float64           `json:"first_serve_pct"`
	FirstServePointsWonPct   *float64           `json:"first_serve_points_won_pct"`
	SecondServePointsWonPct  *float64           `json:"second_serve_points_won_pct"`
	BreakPointsSavedPct      *float64           `json:"break_points_saved_pct"`
	ReturnRating             *int               `json:"return_rating"`
	FirstReturnWonPct        *float64           `json:"first_return_won_pct"`
	SecondReturnWonPct       *float64           `json:"second_return_won_pct"`
	BreakPointsConvertedPct  *float64           `json:"break_points_converted_pct"`
	TotalServicePointsWonPct *float64           `json:"total_service_points_won_pct"`
	TotalReturnPointsWonPct  *float64           `json:"total_return_points_won_pct"`
	TotalPointsWonPct        *float64           `json:"total_points_won_pct"`
}

type ATPMatchStatsMatch struct {
	ID           int     `json:"id"`
	TournamentID int     `json:"tournament_id"`
	Season       int     `json:"season"`
	Round        *string `json:"round"`
	Player1ID    int     `json:"player1_id"`
	Player2ID    int     `json:"player2_id"`
	WinnerID     *int    `json:"winner_id"`
}

type ATPPlayerCareerStats struct {
	Player           ATPPlayer `json:"player"`
	CareerTitles     *int      `json:"career_titles"`
	CareerPrizeMoney *float64  `json:"career_prize_money"`
	SinglesWins      *int      `json:"singles_wins"`
	SinglesLosses    *int      `json:"singles_losses"`
	YTDWins          *int      `json:"ytd_wins"`
	YTDLosses        *int      `json:"ytd_losses"`
	YTDTitles        *int      `json:"ytd_titles"`
}

type ATPHeadToHead struct {
	ID          int       `json:"id"`
	Player1     ATPPlayer `json:"player1"`
	Player2     ATPPlayer `json:"player2"`
	Player1Wins int       `json:"player1_wins"`
	Player2Wins int       `json:"player2_wins"`
}

type ATPBettingOdd struct {
	ID          int        `json:"id"`
	MatchID     int        `json:"match_id"`
	Vendor      string     `json:"vendor"`
	Player1     *ATPPlayer `json:"player1"`
	Player2     *ATPPlayer `json:"player2"`
	Player1Odds *int       `json:"player1_odds"`
	Player2Odds *int       `json:"player2_odds"`
	UpdatedAt   string     `json:"updated_at"`
}

func (c *clientImpl) ListPlayers(ctx context.Context, options ListPlayersOptions) (*ListPlayersResponse, error) {
	query := options.values()
	var out ListPlayersResponse
	if err := c.do(ctx, http.MethodGet, "/atp/v1/players", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clientImpl) GetPlayer(ctx context.Context, id int) (*ATPPlayer, error) {
	var out dataEnvelope[ATPPlayer]
	if err := c.do(ctx, http.MethodGet, "/atp/v1/players/"+url.PathEscape(strconv.Itoa(id)), nil, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *clientImpl) ListTournaments(ctx context.Context, options ListTournamentsOptions) (*ListTournamentsResponse, error) {
	query := options.values()
	var out ListTournamentsResponse
	if err := c.do(ctx, http.MethodGet, "/atp/v1/tournaments", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clientImpl) GetTournament(ctx context.Context, id int, options GetTournamentOptions) (*ATPTournament, error) {
	query := options.values()
	var out dataEnvelope[ATPTournament]
	if err := c.do(ctx, http.MethodGet, "/atp/v1/tournaments/"+url.PathEscape(strconv.Itoa(id)), query, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *clientImpl) ListRankings(ctx context.Context, options ListRankingsOptions) (*ListRankingsResponse, error) {
	query := options.values()
	var out ListRankingsResponse
	if err := c.do(ctx, http.MethodGet, "/atp/v1/rankings", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clientImpl) ListMatches(ctx context.Context, options ListMatchesOptions) (*ListMatchesResponse, error) {
	query := options.values()
	var out ListMatchesResponse
	if err := c.do(ctx, http.MethodGet, "/atp/v1/matches", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clientImpl) GetMatch(ctx context.Context, id int) (*ATPMatch, error) {
	var out dataEnvelope[ATPMatch]
	if err := c.do(ctx, http.MethodGet, "/atp/v1/matches/"+url.PathEscape(strconv.Itoa(id)), nil, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *clientImpl) ListATPRace(ctx context.Context, options ListATPRaceOptions) (*ListATPRaceResponse, error) {
	query := options.values()
	var out ListATPRaceResponse
	if err := c.do(ctx, http.MethodGet, "/atp/v1/atp_race", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clientImpl) ListMatchStats(ctx context.Context, options ListMatchStatsOptions) (*ListMatchStatsResponse, error) {
	query := options.values()
	var out ListMatchStatsResponse
	if err := c.do(ctx, http.MethodGet, "/atp/v1/match_stats", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clientImpl) ListPlayerCareerStats(ctx context.Context, options ListPlayerCareerStatsOptions) (*ListPlayerCareerStatsResponse, error) {
	query := options.values()
	var out ListPlayerCareerStatsResponse
	if err := c.do(ctx, http.MethodGet, "/atp/v1/player_career_stats", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clientImpl) GetHeadToHead(ctx context.Context, options HeadToHeadOptions) (*ATPHeadToHead, error) {
	query := options.values()
	var out dataEnvelope[ATPHeadToHead]
	if err := c.do(ctx, http.MethodGet, "/atp/v1/head_to_head", query, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *clientImpl) ListOdds(ctx context.Context, options ListOddsOptions) (*ListOddsResponse, error) {
	query := options.values()
	var out ListOddsResponse
	if err := c.do(ctx, http.MethodGet, "/atp/v1/odds", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clientImpl) do(ctx context.Context, method string, path string, query url.Values, out any) error {
	endpoint, err := c.buildURL(path, query)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create balldontlie request: %w", err)
	}
	req.Header.Set("Authorization", c.config.APIKey)

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("wait balldontlie rate limit: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send balldontlie request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeHTTPError(resp)
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode balldontlie response: %w", err)
	}
	return nil
}

func (c *clientImpl) buildURL(path string, query url.Values) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimRight(c.config.BaseURL, "/") + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse balldontlie request url: %w", err)
	}
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}
	return endpoint, nil
}

func decodeHTTPError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
	if err != nil {
		return fmt.Errorf("balldontlie request failed with status %s and unreadable body: %w", resp.Status, err)
	}
	trimmed := strings.TrimSpace(string(body))

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil {
		apiErr.StatusCode = resp.StatusCode
		apiErr.RawBody = trimmed
		if apiErr.Message != "" {
			return &apiErr
		}
	}

	if trimmed == "" {
		trimmed = resp.Status
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    trimmed,
		RawBody:    trimmed,
	}
}

type dataEnvelope[T any] struct {
	Data T `json:"data"`
}

func (o PageOptions) values() url.Values {
	q := make(url.Values)
	setIntPtr(q, "cursor", o.Cursor)
	setIntPtr(q, "per_page", o.PerPage)
	return q
}

func (o ListPlayersOptions) values() url.Values {
	q := o.PageOptions.values()
	addIntSlice(q, "player_ids[]", o.PlayerIDs)
	setString(q, "search", o.Search)
	setString(q, "first_name", o.FirstName)
	setString(q, "last_name", o.LastName)
	setString(q, "country", o.Country)
	setString(q, "country_code", o.CountryCode)
	return q
}

func (o ListTournamentsOptions) values() url.Values {
	q := o.PageOptions.values()
	addIntSlice(q, "tournament_ids", o.TournamentIDs)
	setIntPtr(q, "season", o.Season)
	setString(q, "surface", o.Surface)
	setString(q, "category", o.Category)
	return q
}

func (o GetTournamentOptions) values() url.Values {
	q := make(url.Values)
	setIntPtr(q, "season", o.Season)
	return q
}

func (o ListRankingsOptions) values() url.Values {
	q := o.PageOptions.values()
	addIntSlice(q, "player_ids[]", o.PlayerIDs)
	setString(q, "date", o.Date)
	return q
}

func (o ListMatchesOptions) values() url.Values {
	q := o.PageOptions.values()
	addIntSlice(q, "tournament_ids", o.TournamentIDs)
	addIntSlice(q, "player_ids[]", o.PlayerIDs)
	setIntPtr(q, "season", o.Season)
	setString(q, "round", o.Round)
	setBoolPtr(q, "is_live", o.IsLive)
	return q
}

func (o ListATPRaceOptions) values() url.Values {
	q := o.PageOptions.values()
	addIntSlice(q, "player_ids[]", o.PlayerIDs)
	setString(q, "date", o.Date)
	return q
}

func (o ListMatchStatsOptions) values() url.Values {
	q := o.PageOptions.values()
	addIntSlice(q, "match_ids", o.MatchIDs)
	addIntSlice(q, "player_ids[]", o.PlayerIDs)
	setIntPtr(q, "set_number", o.SetNumber)
	return q
}

func (o ListPlayerCareerStatsOptions) values() url.Values {
	q := o.PageOptions.values()
	addIntSlice(q, "player_ids[]", o.PlayerIDs)
	return q
}

func (o HeadToHeadOptions) values() url.Values {
	q := make(url.Values)
	q.Set("player1_id", strconv.Itoa(o.Player1ID))
	q.Set("player2_id", strconv.Itoa(o.Player2ID))
	return q
}

func (o ListOddsOptions) values() url.Values {
	q := o.PageOptions.values()
	addIntSlice(q, "match_ids", o.MatchIDs)
	addIntSlice(q, "tournament_ids", o.TournamentIDs)
	addIntSlice(q, "player_ids[]", o.PlayerIDs)
	setIntPtr(q, "season", o.Season)
	return q
}

func setString(q url.Values, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		q.Set(key, value)
	}
}

func setIntPtr(q url.Values, key string, value *int) {
	if value != nil {
		q.Set(key, strconv.Itoa(*value))
	}
}

func setBoolPtr(q url.Values, key string, value *bool) {
	if value != nil {
		q.Set(key, strconv.FormatBool(*value))
	}
}

func addIntSlice(q url.Values, key string, values []int) {
	for _, value := range values {
		q.Add(key, strconv.Itoa(value))
	}
}
