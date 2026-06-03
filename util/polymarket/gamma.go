package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultGammaBaseURL = "https://gamma-api.polymarket.com"
	DefaultTimeout      = 30 * time.Second

	errorBodyLimit = 16 * 1024
)

// GammaClient is a typed Polymarket Gamma API client.
type GammaClient interface {
	// Markets
	ListMarkets(ctx context.Context, options ListMarketsOptions) ([]Market, error)
	ListMarketsKeyset(ctx context.Context, options ListMarketsKeysetOptions) (*MarketKeysetResponse, error)
	GetMarketByID(ctx context.Context, id int64, options GetMarketOptions) (*Market, error)
	GetMarketBySlug(ctx context.Context, slug string, options GetMarketOptions) (*Market, error)
	GetMarketTagsByID(ctx context.Context, id int64) ([]Tag, error)

	// Events
	ListEvents(ctx context.Context, options ListEventsOptions) ([]Event, error)
	ListEventsKeyset(ctx context.Context, options ListEventsKeysetOptions) (*EventKeysetResponse, error)
	GetEventByID(ctx context.Context, id int64, options GetEventOptions) (*Event, error)
	GetEventBySlug(ctx context.Context, slug string, options GetEventOptions) (*Event, error)
	GetEventTags(ctx context.Context, id int64) ([]Tag, error)

	// Tags
	ListTags(ctx context.Context, options ListTagsOptions) ([]Tag, error)
	GetTagByID(ctx context.Context, id string, options GetTagOptions) (*Tag, error)
	GetTagBySlug(ctx context.Context, slug string, options GetTagOptions) (*Tag, error)
	GetRelatedTagsByTagID(ctx context.Context, id string, options RelatedTagsOptions) ([]RelatedTagRelationship, error)
	GetRelatedTagsByTagSlug(ctx context.Context, slug string, options RelatedTagsOptions) ([]RelatedTagRelationship, error)
	GetTagsRelatedToTagID(ctx context.Context, id string, options RelatedTagsOptions) ([]Tag, error)
	GetTagsRelatedToTagSlug(ctx context.Context, slug string, options RelatedTagsOptions) ([]Tag, error)

	// Comments
	ListComments(ctx context.Context, options ListCommentsOptions) ([]Comment, error)
	GetCommentByID(ctx context.Context, id string, options GetCommentOptions) ([]Comment, error)
	GetCommentsByUserAddress(ctx context.Context, userAddress string, options ListOptions) ([]Comment, error)

	// Series
	ListSeries(ctx context.Context, options ListSeriesOptions) ([]Series, error)
	GetSeriesByID(ctx context.Context, id int64, options GetSeriesOptions) (*Series, error)

	// Profiles
	GetPublicProfile(ctx context.Context, address string) (*PublicProfile, error)

	// Search
	PublicSearch(ctx context.Context, options PublicSearchOptions) (*PublicSearchResponse, error)

	// Sports
	GetSportsMetadata(ctx context.Context) ([]SportsMetadata, error)
	GetSportsMarketTypes(ctx context.Context) (*SportsMarketTypesResponse, error)
	ListTeams(ctx context.Context, options ListTeamsOptions) ([]Team, error)
}

type GammaConfig struct {
	GammaBaseURL string
	Timeout      time.Duration
}

func (c GammaConfig) WithDefaults() GammaConfig {
	return c.withDefaults()
}

func (c GammaConfig) withDefaults() GammaConfig {
	c.GammaBaseURL = strings.TrimSpace(c.GammaBaseURL)
	if c.GammaBaseURL == "" {
		c.GammaBaseURL = DefaultGammaBaseURL
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}

type gammaClientImpl struct {
	config GammaConfig
	http   *http.Client
}

var _ GammaClient = (*gammaClientImpl)(nil)

func NewGammaClient(config GammaConfig) (GammaClient, error) {
	config = config.withDefaults()
	if _, err := url.ParseRequestURI(config.GammaBaseURL); err != nil {
		return nil, fmt.Errorf("invalid polymarket gamma base url: %w", err)
	}
	return &gammaClientImpl{
		config: config,
		http:   &http.Client{Timeout: config.Timeout},
	}, nil
}

type APIError struct {
	StatusCode int    `json:"-"`
	Service    string `json:"-"`
	Type       string `json:"type,omitempty"`
	Message    string `json:"error,omitempty"`
	RawBody    string `json:"-"`
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	service := strings.TrimSpace(e.Service)
	if service == "" {
		service = "gamma"
	}
	if e.Type != "" && e.Message != "" {
		return fmt.Sprintf("polymarket %s request failed (%d): %s: %s", service, e.StatusCode, e.Type, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("polymarket %s request failed (%d): %s", service, e.StatusCode, e.Message)
	}
	if e.RawBody != "" {
		return fmt.Sprintf("polymarket %s request failed (%d): %s", service, e.StatusCode, e.RawBody)
	}
	return fmt.Sprintf("polymarket %s request failed with status %d", service, e.StatusCode)
}

// Query option structs.

type ListOptions struct {
	Limit     *int
	Offset    *int
	Order     string
	Ascending *bool
}

type ListMarketsOptions struct {
	ListOptions
	ID                  []int64
	Slug                []string
	ClobTokenIDs        []string
	ConditionIDs        []string
	MarketMakerAddress  []string
	LiquidityNumMin     *float64
	LiquidityNumMax     *float64
	VolumeNumMin        *float64
	VolumeNumMax        *float64
	StartDateMin        *time.Time
	StartDateMax        *time.Time
	EndDateMin          *time.Time
	EndDateMax          *time.Time
	TagID               *int64
	RelatedTags         *bool
	CYOM                *bool
	UMAResolutionStatus string
	GameID              string
	SportsMarketTypes   []string
	RewardsMinSize      *float64
	QuestionIDs         []string
	IncludeTag          *bool
	Closed              *bool
}

type ListMarketsKeysetOptions struct {
	Limit               *int
	Order               string
	Ascending           *bool
	AfterCursor         string
	ID                  []int64
	Slug                []string
	Closed              *bool
	Decimalized         *bool
	ClobTokenIDs        []string
	ConditionIDs        []string
	QuestionIDs         []string
	MarketMakerAddress  []string
	LiquidityNumMin     *float64
	LiquidityNumMax     *float64
	VolumeNumMin        *float64
	VolumeNumMax        *float64
	StartDateMin        *time.Time
	StartDateMax        *time.Time
	EndDateMin          *time.Time
	EndDateMax          *time.Time
	TagID               []int64
	RelatedTags         *bool
	TagMatch            string
	CYOM                *bool
	RFQEnabled          *bool
	UMAResolutionStatus string
	GameID              string
	SportsMarketTypes   []string
	IncludeTag          *bool
	Locale              string
}

type GetMarketOptions struct {
	IncludeTag *bool
}

type ListEventsOptions struct {
	ListOptions
	ID              []int64
	TagID           *int64
	ExcludeTagID    []int64
	Slug            []string
	TagSlug         string
	RelatedTags     *bool
	Active          *bool
	Archived        *bool
	Featured        *bool
	CYOM            *bool
	IncludeChat     *bool
	IncludeTemplate *bool
	Recurrence      string
	Closed          *bool
	LiquidityMin    *float64
	LiquidityMax    *float64
	VolumeMin       *float64
	VolumeMax       *float64
	StartDateMin    *time.Time
	StartDateMax    *time.Time
	EndDateMin      *time.Time
	EndDateMax      *time.Time
}

type ListEventsKeysetOptions struct {
	Limit            *int
	Order            string
	Ascending        *bool
	AfterCursor      string
	ID               []int64
	Slug             []string
	Closed           *bool
	Live             *bool
	Featured         *bool
	CYOM             *bool
	TitleSearch      string
	LiquidityMin     *float64
	LiquidityMax     *float64
	VolumeMin        *float64
	VolumeMax        *float64
	StartDateMin     *time.Time
	StartDateMax     *time.Time
	EndDateMin       *time.Time
	EndDateMax       *time.Time
	StartTimeMin     *time.Time
	StartTimeMax     *time.Time
	TagID            []int64
	TagSlug          string
	ExcludeTagID     []int64
	RelatedTags      *bool
	TagMatch         string
	SeriesID         []int64
	GameID           []int64
	EventDate        *time.Time
	EventWeek        *int
	FeaturedOrder    *bool
	Recurrence       string
	CreatedBy        []string
	ParentEventID    *int64
	IncludeChildren  *bool
	PartnerSlug      string
	IncludeChat      *bool
	IncludeTemplate  *bool
	IncludeBestLines *bool
	Locale           string
}

type GetEventOptions struct {
	IncludeChat     *bool
	IncludeTemplate *bool
}

type ListTagsOptions struct {
	ListOptions
	IncludeTemplate *bool
	IsCarousel      *bool
}

type GetTagOptions struct {
	IncludeTemplate *bool
}

type RelatedTagsOptions struct {
	OmitEmpty string
	Status    string
}

type ListCommentsOptions struct {
	ListOptions
	ParentEntityType string
	ParentEntityID   *int64
	GetPositions     *bool
	HoldersOnly      *bool
}

type GetCommentOptions struct {
	GetPositions *bool
}

type ListSeriesOptions struct {
	ListOptions
	Slug             []string
	CategoriesIDs    []int64
	CategoriesLabels []string
	Closed           *bool
	IncludeChat      *bool
	Recurrence       string
	ExcludeEvents    *bool
}

type GetSeriesOptions struct {
	IncludeChat *bool
}

type PublicSearchOptions struct {
	Q                 string
	Cache             *bool
	EventsStatus      string
	LimitPerType      *int
	Page              *int
	EventsTag         []string
	KeepClosedMarkets *int
	Sort              string
	Ascending         *bool
	SearchTags        *bool
	SearchProfiles    *bool
	Recurrence        string
	ExcludeTagID      []int64
	Optimized         *bool
}

type ListTeamsOptions struct {
	ListOptions
	League       []string
	Name         []string
	Abbreviation []string
}

// API models.

type Market struct {
	ID                    string          `json:"id"`
	Question              *string         `json:"question,omitempty"`
	SportsMarketType      *string         `json:"sportsMarketType,omitempty"`
	GroupItemTitle        *string         `json:"groupItemTitle,omitempty"`
	ConditionID           *string         `json:"conditionId,omitempty"`
	Slug                  *string         `json:"slug,omitempty"`
	ResolutionSource      *string         `json:"resolutionSource,omitempty"`
	EndDate               *time.Time      `json:"endDate,omitempty"`
	StartDate             *time.Time      `json:"startDate,omitempty"`
	Image                 *string         `json:"image,omitempty"`
	Icon                  *string         `json:"icon,omitempty"`
	Description           *string         `json:"description,omitempty"`
	Outcomes              *string         `json:"outcomes,omitempty"`
	OutcomePrices         *string         `json:"outcomePrices,omitempty"`
	Volume                *string         `json:"volume,omitempty"`
	Active                *bool           `json:"active,omitempty"`
	Closed                *bool           `json:"closed,omitempty"`
	MarketMakerAddress    *string         `json:"marketMakerAddress,omitempty"`
	CreatedAt             *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt             *time.Time      `json:"updatedAt,omitempty"`
	Archived              *bool           `json:"archived,omitempty"`
	Restricted            *bool           `json:"restricted,omitempty"`
	QuestionID            *string         `json:"questionID,omitempty"`
	EnableOrderBook       *bool           `json:"enableOrderBook,omitempty"`
	OrderPriceMinTickSize *float64        `json:"orderPriceMinTickSize,omitempty"`
	OrderMinSize          *float64        `json:"orderMinSize,omitempty"`
	VolumeNum             *float64        `json:"volumeNum,omitempty"`
	LiquidityNum          *float64        `json:"liquidityNum,omitempty"`
	EndDateIso            *string         `json:"endDateIso,omitempty"`
	StartDateIso          *string         `json:"startDateIso,omitempty"`
	Volume24hr            *float64        `json:"volume24hr,omitempty"`
	Volume1wk             *float64        `json:"volume1wk,omitempty"`
	Volume1mo             *float64        `json:"volume1mo,omitempty"`
	Volume1yr             *float64        `json:"volume1yr,omitempty"`
	ClobTokenIDs          *string         `json:"clobTokenIds,omitempty"`
	NegRisk               *bool           `json:"negRisk,omitempty"`
	Spread                *float64        `json:"spread,omitempty"`
	LastTradePrice        *float64        `json:"lastTradePrice,omitempty"`
	BestBid               *float64        `json:"bestBid,omitempty"`
	BestAsk               *float64        `json:"bestAsk,omitempty"`
	Events                []Event         `json:"events,omitempty"`
	Tags                  []Tag           `json:"tags,omitempty"`
	Raw                   json.RawMessage `json:"-"`
}

type Event struct {
	ID                string            `json:"id"`
	Ticker            *string           `json:"ticker,omitempty"`
	Slug              *string           `json:"slug,omitempty"`
	Title             *string           `json:"title,omitempty"`
	Description       *string           `json:"description,omitempty"`
	ResolutionSource  *string           `json:"resolutionSource,omitempty"`
	StartDate         *time.Time        `json:"startDate,omitempty"`
	CreationDate      *time.Time        `json:"creationDate,omitempty"`
	EndDate           *time.Time        `json:"endDate,omitempty"`
	Image             *string           `json:"image,omitempty"`
	Icon              *string           `json:"icon,omitempty"`
	Active            *bool             `json:"active,omitempty"`
	Closed            *bool             `json:"closed,omitempty"`
	Archived          *bool             `json:"archived,omitempty"`
	Featured          *bool             `json:"featured,omitempty"`
	Restricted        *bool             `json:"restricted,omitempty"`
	Liquidity         *float64          `json:"liquidity,omitempty"`
	Volume            *float64          `json:"volume,omitempty"`
	OpenInterest      *float64          `json:"openInterest,omitempty"`
	Category          *string           `json:"category,omitempty"`
	Live              *bool             `json:"live,omitempty"`
	Ended             *bool             `json:"ended,omitempty"`
	Score             *string           `json:"score,omitempty"`
	Period            *string           `json:"period,omitempty"`
	Elapsed           *string           `json:"elapsed,omitempty"`
	FinishedTimestamp *string           `json:"finishedTimestamp,omitempty"`
	GameID            *int64            `json:"gameId,omitempty"`
	EventDate         *string           `json:"eventDate,omitempty"`
	StartTime         *time.Time        `json:"startTime,omitempty"`
	GameStatus        *string           `json:"gameStatus,omitempty"`
	Sport             json.RawMessage   `json:"sport,omitempty"`
	Teams             []json.RawMessage `json:"teams,omitempty"`
	CreatedAt         *time.Time        `json:"createdAt,omitempty"`
	UpdatedAt         *time.Time        `json:"updatedAt,omitempty"`
	CommentCount      *int64            `json:"commentCount,omitempty"`
	Markets           []Market          `json:"markets,omitempty"`
	Tags              []Tag             `json:"tags,omitempty"`
	Raw               json.RawMessage   `json:"-"`
}

type MarketKeysetResponse struct {
	Markets    []Market `json:"markets"`
	NextCursor *string  `json:"next_cursor,omitempty"`
}

type EventKeysetResponse struct {
	Events     []Event `json:"events"`
	NextCursor *string `json:"next_cursor,omitempty"`
}

type Tag struct {
	ID                  string     `json:"id"`
	Label               *string    `json:"label,omitempty"`
	Slug                *string    `json:"slug,omitempty"`
	ForceShow           *bool      `json:"forceShow,omitempty"`
	ForceHide           *bool      `json:"forceHide,omitempty"`
	PublishedAt         *string    `json:"publishedAt,omitempty"`
	CreatedAt           *time.Time `json:"createdAt,omitempty"`
	UpdatedAt           *time.Time `json:"updatedAt,omitempty"`
	RequiresTranslation *bool      `json:"requiresTranslation,omitempty"`
	ActiveEventsCount   *int64     `json:"activeEventsCount,omitempty"`
	IsCarousel          *bool      `json:"isCarousel,omitempty"`
}

type RelatedTagRelationship struct {
	ID         string     `json:"id"`
	TagID      *string    `json:"tagID,omitempty"`
	RelatedID  *string    `json:"relatedTagID,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
	Tag        *Tag       `json:"tag,omitempty"`
	RelatedTag *Tag       `json:"relatedTag,omitempty"`
}

type PublicSearchResponse struct {
	Events     []Event           `json:"events,omitempty"`
	Tags       []Tag             `json:"tags,omitempty"`
	Pagination *SearchPagination `json:"pagination,omitempty"`
}

type SearchPagination struct {
	HasMore      *bool `json:"hasMore,omitempty"`
	TotalResults *int  `json:"totalResults,omitempty"`
}

type SportsMetadata struct {
	ID         *int64     `json:"id,omitempty"`
	Sport      *string    `json:"sport,omitempty"`
	Image      *string    `json:"image,omitempty"`
	Resolution *string    `json:"resolution,omitempty"`
	Ordering   *string    `json:"ordering,omitempty"`
	Tags       *string    `json:"tags,omitempty"`
	Series     *string    `json:"series,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
}

type SportsMarketTypesResponse struct {
	Schema      *string  `json:"$schema,omitempty"`
	MarketTypes []string `json:"marketTypes,omitempty"`
}

type Team struct {
	ID           *int64     `json:"id,omitempty"`
	Name         *string    `json:"name,omitempty"`
	League       *string    `json:"league,omitempty"`
	Record       *string    `json:"record,omitempty"`
	Logo         *string    `json:"logo,omitempty"`
	Abbreviation *string    `json:"abbreviation,omitempty"`
	Alias        *string    `json:"alias,omitempty"`
	ProviderID   *int64     `json:"providerId,omitempty"`
	Color        *string    `json:"color,omitempty"`
	CreatedAt    *time.Time `json:"createdAt,omitempty"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
}

type Comment struct {
	ID               string            `json:"id"`
	Body             *string           `json:"body,omitempty"`
	ParentEntityType *string           `json:"parentEntityType,omitempty"`
	ParentEntityID   *int64            `json:"parentEntityID,omitempty"`
	ParentCommentID  *string           `json:"parentCommentID,omitempty"`
	UserAddress      *string           `json:"userAddress,omitempty"`
	ReplyAddress     *string           `json:"replyAddress,omitempty"`
	CreatedAt        *time.Time        `json:"createdAt,omitempty"`
	UpdatedAt        *time.Time        `json:"updatedAt,omitempty"`
	Profile          json.RawMessage   `json:"profile,omitempty"`
	Reactions        []json.RawMessage `json:"reactions,omitempty"`
	ReportCount      *int64            `json:"reportCount,omitempty"`
	ReactionCount    *int64            `json:"reactionCount,omitempty"`
}

type PublicProfile struct {
	CreatedAt             *time.Time        `json:"createdAt,omitempty"`
	ProxyWallet           *string           `json:"proxyWallet,omitempty"`
	ProfileImage          *string           `json:"profileImage,omitempty"`
	DisplayUsernamePublic *bool             `json:"displayUsernamePublic,omitempty"`
	Bio                   *string           `json:"bio,omitempty"`
	Pseudonym             *string           `json:"pseudonym,omitempty"`
	Name                  *string           `json:"name,omitempty"`
	Users                 []json.RawMessage `json:"users,omitempty"`
	XUsername             *string           `json:"xUsername,omitempty"`
	VerifiedBadge         *bool             `json:"verifiedBadge,omitempty"`
}

type Series struct {
	ID           string            `json:"id"`
	Ticker       *string           `json:"ticker,omitempty"`
	Slug         *string           `json:"slug,omitempty"`
	Title        *string           `json:"title,omitempty"`
	Subtitle     *string           `json:"subtitle,omitempty"`
	SeriesType   *string           `json:"seriesType,omitempty"`
	Recurrence   *string           `json:"recurrence,omitempty"`
	Description  *string           `json:"description,omitempty"`
	Image        *string           `json:"image,omitempty"`
	Icon         *string           `json:"icon,omitempty"`
	Layout       *string           `json:"layout,omitempty"`
	Active       *bool             `json:"active,omitempty"`
	Closed       *bool             `json:"closed,omitempty"`
	Archived     *bool             `json:"archived,omitempty"`
	Featured     *bool             `json:"featured,omitempty"`
	CreatedAt    *time.Time        `json:"createdAt,omitempty"`
	UpdatedAt    *time.Time        `json:"updatedAt,omitempty"`
	CommentCount *int64            `json:"commentCount,omitempty"`
	Events       []json.RawMessage `json:"events,omitempty"`
	Collections  []json.RawMessage `json:"collections,omitempty"`
	Categories   []json.RawMessage `json:"categories,omitempty"`
	Tags         []json.RawMessage `json:"tags,omitempty"`
	Chats        []json.RawMessage `json:"chats,omitempty"`
	Raw          json.RawMessage   `json:"-"`
}

func (c *gammaClientImpl) ListMarkets(ctx context.Context, options ListMarketsOptions) ([]Market, error) {
	query := options.values()
	var out []Market
	if err := c.do(ctx, http.MethodGet, "/markets", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) ListMarketsKeyset(ctx context.Context, options ListMarketsKeysetOptions) (*MarketKeysetResponse, error) {
	query := options.values()
	var out MarketKeysetResponse
	if err := c.do(ctx, http.MethodGet, "/markets/keyset", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) GetMarketByID(ctx context.Context, id int64, options GetMarketOptions) (*Market, error) {
	query := options.values()
	var out Market
	if err := c.do(ctx, http.MethodGet, "/markets/"+pathEscapeInt(id), query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) GetMarketBySlug(ctx context.Context, slug string, options GetMarketOptions) (*Market, error) {
	query := options.values()
	var out Market
	if err := c.do(ctx, http.MethodGet, "/markets/slug/"+url.PathEscape(strings.TrimSpace(slug)), query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) GetMarketTagsByID(ctx context.Context, id int64) ([]Tag, error) {
	var out []Tag
	if err := c.do(ctx, http.MethodGet, "/markets/"+pathEscapeInt(id)+"/tags", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) ListEvents(ctx context.Context, options ListEventsOptions) ([]Event, error) {
	query := options.values()
	var out []Event
	if err := c.do(ctx, http.MethodGet, "/events", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) ListEventsKeyset(ctx context.Context, options ListEventsKeysetOptions) (*EventKeysetResponse, error) {
	query := options.values()
	var out EventKeysetResponse
	if err := c.do(ctx, http.MethodGet, "/events/keyset", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) GetEventByID(ctx context.Context, id int64, options GetEventOptions) (*Event, error) {
	query := options.values()
	var out Event
	if err := c.do(ctx, http.MethodGet, "/events/"+pathEscapeInt(id), query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) GetEventBySlug(ctx context.Context, slug string, options GetEventOptions) (*Event, error) {
	query := options.values()
	var out Event
	if err := c.do(ctx, http.MethodGet, "/events/slug/"+url.PathEscape(strings.TrimSpace(slug)), query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) GetEventTags(ctx context.Context, id int64) ([]Tag, error) {
	var out []Tag
	if err := c.do(ctx, http.MethodGet, "/events/"+pathEscapeInt(id)+"/tags", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) ListTags(ctx context.Context, options ListTagsOptions) ([]Tag, error) {
	query := options.values()
	var out []Tag
	if err := c.do(ctx, http.MethodGet, "/tags", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) GetTagByID(ctx context.Context, id string, options GetTagOptions) (*Tag, error) {
	query := options.values()
	var out Tag
	if err := c.do(ctx, http.MethodGet, "/tags/"+url.PathEscape(strings.TrimSpace(id)), query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) GetTagBySlug(ctx context.Context, slug string, options GetTagOptions) (*Tag, error) {
	query := options.values()
	var out Tag
	if err := c.do(ctx, http.MethodGet, "/tags/slug/"+url.PathEscape(strings.TrimSpace(slug)), query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) GetRelatedTagsByTagID(ctx context.Context, id string, options RelatedTagsOptions) ([]RelatedTagRelationship, error) {
	query := options.values()
	var out []RelatedTagRelationship
	if err := c.do(ctx, http.MethodGet, "/tags/"+url.PathEscape(strings.TrimSpace(id))+"/related-tags", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) GetRelatedTagsByTagSlug(ctx context.Context, slug string, options RelatedTagsOptions) ([]RelatedTagRelationship, error) {
	query := options.values()
	var out []RelatedTagRelationship
	if err := c.do(ctx, http.MethodGet, "/tags/slug/"+url.PathEscape(strings.TrimSpace(slug))+"/related-tags", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) GetTagsRelatedToTagID(ctx context.Context, id string, options RelatedTagsOptions) ([]Tag, error) {
	query := options.values()
	var out []Tag
	if err := c.do(ctx, http.MethodGet, "/tags/"+url.PathEscape(strings.TrimSpace(id))+"/related-tags/tags", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) GetTagsRelatedToTagSlug(ctx context.Context, slug string, options RelatedTagsOptions) ([]Tag, error) {
	query := options.values()
	var out []Tag
	if err := c.do(ctx, http.MethodGet, "/tags/slug/"+url.PathEscape(strings.TrimSpace(slug))+"/related-tags/tags", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) ListComments(ctx context.Context, options ListCommentsOptions) ([]Comment, error) {
	query := options.values()
	var out []Comment
	if err := c.do(ctx, http.MethodGet, "/comments", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) GetCommentByID(ctx context.Context, id string, options GetCommentOptions) ([]Comment, error) {
	query := options.values()
	var out []Comment
	if err := c.do(ctx, http.MethodGet, "/comments/"+url.PathEscape(strings.TrimSpace(id)), query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) GetCommentsByUserAddress(ctx context.Context, userAddress string, options ListOptions) ([]Comment, error) {
	query := options.values()
	var out []Comment
	if err := c.do(ctx, http.MethodGet, "/comments/user_address/"+url.PathEscape(strings.TrimSpace(userAddress)), query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) ListSeries(ctx context.Context, options ListSeriesOptions) ([]Series, error) {
	query := options.values()
	var out []Series
	if err := c.do(ctx, http.MethodGet, "/series", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) GetSeriesByID(ctx context.Context, id int64, options GetSeriesOptions) (*Series, error) {
	query := options.values()
	var out Series
	if err := c.do(ctx, http.MethodGet, "/series/"+pathEscapeInt(id), query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) GetPublicProfile(ctx context.Context, address string) (*PublicProfile, error) {
	query := make(url.Values)
	setString(query, "address", address)
	var out PublicProfile
	if err := c.do(ctx, http.MethodGet, "/public-profile", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) PublicSearch(ctx context.Context, options PublicSearchOptions) (*PublicSearchResponse, error) {
	query := options.values()
	var out PublicSearchResponse
	if err := c.do(ctx, http.MethodGet, "/public-search", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) GetSportsMetadata(ctx context.Context) ([]SportsMetadata, error) {
	var out []SportsMetadata
	if err := c.do(ctx, http.MethodGet, "/sports", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) GetSportsMarketTypes(ctx context.Context) (*SportsMarketTypesResponse, error) {
	var out SportsMarketTypesResponse
	if err := c.do(ctx, http.MethodGet, "/sports/market-types", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gammaClientImpl) ListTeams(ctx context.Context, options ListTeamsOptions) ([]Team, error) {
	query := options.values()
	var out []Team
	if err := c.do(ctx, http.MethodGet, "/teams", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gammaClientImpl) do(ctx context.Context, method, path string, query url.Values, out any) error {
	endpoint, err := c.buildURL(path, query)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create polymarket gamma request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send polymarket gamma request: %w", err)
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
		return fmt.Errorf("failed to decode polymarket gamma response: %w", err)
	}
	return nil
}

func (c *gammaClientImpl) buildURL(path string, query url.Values) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimRight(c.config.GammaBaseURL, "/") + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse polymarket gamma request url: %w", err)
	}
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}
	return endpoint, nil
}

func decodeHTTPError(resp *http.Response) error {
	return decodeHTTPErrorWithService(resp, "gamma")
}

func decodeHTTPErrorWithService(resp *http.Response, service string) error {
	service = strings.TrimSpace(service)
	if service == "" {
		service = "gamma"
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
	if err != nil {
		return fmt.Errorf("polymarket %s request failed with status %s and unreadable body: %w", service, resp.Status, err)
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil {
		apiErr.StatusCode = resp.StatusCode
		apiErr.Service = service
		apiErr.RawBody = strings.TrimSpace(string(body))
		if apiErr.Type != "" || apiErr.Message != "" {
			return &apiErr
		}
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = resp.Status
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		Service:    service,
		Message:    trimmed,
		RawBody:    trimmed,
	}
}

func (o ListOptions) values() url.Values {
	q := make(url.Values)
	setIntPtr(q, "limit", o.Limit)
	setIntPtr(q, "offset", o.Offset)
	setString(q, "order", o.Order)
	setBoolPtr(q, "ascending", o.Ascending)
	return q
}

func (o ListMarketsOptions) values() url.Values {
	q := o.ListOptions.values()
	addInt64Slice(q, "id", o.ID)
	addStringSlice(q, "slug", o.Slug)
	addStringSlice(q, "clob_token_ids", o.ClobTokenIDs)
	addStringSlice(q, "condition_ids", o.ConditionIDs)
	addStringSlice(q, "market_maker_address", o.MarketMakerAddress)
	setFloat64Ptr(q, "liquidity_num_min", o.LiquidityNumMin)
	setFloat64Ptr(q, "liquidity_num_max", o.LiquidityNumMax)
	setFloat64Ptr(q, "volume_num_min", o.VolumeNumMin)
	setFloat64Ptr(q, "volume_num_max", o.VolumeNumMax)
	setTimePtr(q, "start_date_min", o.StartDateMin)
	setTimePtr(q, "start_date_max", o.StartDateMax)
	setTimePtr(q, "end_date_min", o.EndDateMin)
	setTimePtr(q, "end_date_max", o.EndDateMax)
	setInt64Ptr(q, "tag_id", o.TagID)
	setBoolPtr(q, "related_tags", o.RelatedTags)
	setBoolPtr(q, "cyom", o.CYOM)
	setString(q, "uma_resolution_status", o.UMAResolutionStatus)
	setString(q, "game_id", o.GameID)
	addStringSlice(q, "sports_market_types", o.SportsMarketTypes)
	setFloat64Ptr(q, "rewards_min_size", o.RewardsMinSize)
	addStringSlice(q, "question_ids", o.QuestionIDs)
	setBoolPtr(q, "include_tag", o.IncludeTag)
	setBoolPtr(q, "closed", o.Closed)
	return q
}

func (o ListMarketsKeysetOptions) values() url.Values {
	q := make(url.Values)
	setIntPtr(q, "limit", o.Limit)
	setString(q, "order", o.Order)
	setBoolPtr(q, "ascending", o.Ascending)
	setString(q, "after_cursor", o.AfterCursor)
	addInt64Slice(q, "id", o.ID)
	addStringSlice(q, "slug", o.Slug)
	setBoolPtr(q, "closed", o.Closed)
	setBoolPtr(q, "decimalized", o.Decimalized)
	addStringSlice(q, "clob_token_ids", o.ClobTokenIDs)
	addStringSlice(q, "condition_ids", o.ConditionIDs)
	addStringSlice(q, "question_ids", o.QuestionIDs)
	addStringSlice(q, "market_maker_address", o.MarketMakerAddress)
	setFloat64Ptr(q, "liquidity_num_min", o.LiquidityNumMin)
	setFloat64Ptr(q, "liquidity_num_max", o.LiquidityNumMax)
	setFloat64Ptr(q, "volume_num_min", o.VolumeNumMin)
	setFloat64Ptr(q, "volume_num_max", o.VolumeNumMax)
	setTimePtr(q, "start_date_min", o.StartDateMin)
	setTimePtr(q, "start_date_max", o.StartDateMax)
	setTimePtr(q, "end_date_min", o.EndDateMin)
	setTimePtr(q, "end_date_max", o.EndDateMax)
	addInt64Slice(q, "tag_id", o.TagID)
	setBoolPtr(q, "related_tags", o.RelatedTags)
	setString(q, "tag_match", o.TagMatch)
	setBoolPtr(q, "cyom", o.CYOM)
	setBoolPtr(q, "rfq_enabled", o.RFQEnabled)
	setString(q, "uma_resolution_status", o.UMAResolutionStatus)
	setString(q, "game_id", o.GameID)
	addStringSlice(q, "sports_market_types", o.SportsMarketTypes)
	setBoolPtr(q, "include_tag", o.IncludeTag)
	setString(q, "locale", o.Locale)
	return q
}

func (o GetMarketOptions) values() url.Values {
	q := make(url.Values)
	setBoolPtr(q, "include_tag", o.IncludeTag)
	return q
}

func (o ListEventsOptions) values() url.Values {
	q := o.ListOptions.values()
	addInt64Slice(q, "id", o.ID)
	setInt64Ptr(q, "tag_id", o.TagID)
	addInt64Slice(q, "exclude_tag_id", o.ExcludeTagID)
	addStringSlice(q, "slug", o.Slug)
	setString(q, "tag_slug", o.TagSlug)
	setBoolPtr(q, "related_tags", o.RelatedTags)
	setBoolPtr(q, "active", o.Active)
	setBoolPtr(q, "archived", o.Archived)
	setBoolPtr(q, "featured", o.Featured)
	setBoolPtr(q, "cyom", o.CYOM)
	setBoolPtr(q, "include_chat", o.IncludeChat)
	setBoolPtr(q, "include_template", o.IncludeTemplate)
	setString(q, "recurrence", o.Recurrence)
	setBoolPtr(q, "closed", o.Closed)
	setFloat64Ptr(q, "liquidity_min", o.LiquidityMin)
	setFloat64Ptr(q, "liquidity_max", o.LiquidityMax)
	setFloat64Ptr(q, "volume_min", o.VolumeMin)
	setFloat64Ptr(q, "volume_max", o.VolumeMax)
	setTimePtr(q, "start_date_min", o.StartDateMin)
	setTimePtr(q, "start_date_max", o.StartDateMax)
	setTimePtr(q, "end_date_min", o.EndDateMin)
	setTimePtr(q, "end_date_max", o.EndDateMax)
	return q
}

func (o ListEventsKeysetOptions) values() url.Values {
	q := make(url.Values)
	setIntPtr(q, "limit", o.Limit)
	setString(q, "order", o.Order)
	setBoolPtr(q, "ascending", o.Ascending)
	setString(q, "after_cursor", o.AfterCursor)
	addInt64Slice(q, "id", o.ID)
	addStringSlice(q, "slug", o.Slug)
	setBoolPtr(q, "closed", o.Closed)
	setBoolPtr(q, "live", o.Live)
	setBoolPtr(q, "featured", o.Featured)
	setBoolPtr(q, "cyom", o.CYOM)
	setString(q, "title_search", o.TitleSearch)
	setFloat64Ptr(q, "liquidity_min", o.LiquidityMin)
	setFloat64Ptr(q, "liquidity_max", o.LiquidityMax)
	setFloat64Ptr(q, "volume_min", o.VolumeMin)
	setFloat64Ptr(q, "volume_max", o.VolumeMax)
	setTimePtr(q, "start_date_min", o.StartDateMin)
	setTimePtr(q, "start_date_max", o.StartDateMax)
	setTimePtr(q, "end_date_min", o.EndDateMin)
	setTimePtr(q, "end_date_max", o.EndDateMax)
	setTimePtr(q, "start_time_min", o.StartTimeMin)
	setTimePtr(q, "start_time_max", o.StartTimeMax)
	addInt64Slice(q, "tag_id", o.TagID)
	setString(q, "tag_slug", o.TagSlug)
	addInt64Slice(q, "exclude_tag_id", o.ExcludeTagID)
	setBoolPtr(q, "related_tags", o.RelatedTags)
	setString(q, "tag_match", o.TagMatch)
	addInt64Slice(q, "series_id", o.SeriesID)
	addInt64Slice(q, "game_id", o.GameID)
	setTimePtr(q, "event_date", o.EventDate)
	setIntPtr(q, "event_week", o.EventWeek)
	setBoolPtr(q, "featured_order", o.FeaturedOrder)
	setString(q, "recurrence", o.Recurrence)
	addStringSlice(q, "created_by", o.CreatedBy)
	setInt64Ptr(q, "parent_event_id", o.ParentEventID)
	setBoolPtr(q, "include_children", o.IncludeChildren)
	setString(q, "partner_slug", o.PartnerSlug)
	setBoolPtr(q, "include_chat", o.IncludeChat)
	setBoolPtr(q, "include_template", o.IncludeTemplate)
	setBoolPtr(q, "include_best_lines", o.IncludeBestLines)
	setString(q, "locale", o.Locale)
	return q
}

func (o GetEventOptions) values() url.Values {
	q := make(url.Values)
	setBoolPtr(q, "include_chat", o.IncludeChat)
	setBoolPtr(q, "include_template", o.IncludeTemplate)
	return q
}

func (o ListTagsOptions) values() url.Values {
	q := o.ListOptions.values()
	setBoolPtr(q, "include_template", o.IncludeTemplate)
	setBoolPtr(q, "is_carousel", o.IsCarousel)
	return q
}

func (o GetTagOptions) values() url.Values {
	q := make(url.Values)
	setBoolPtr(q, "include_template", o.IncludeTemplate)
	return q
}

func (o RelatedTagsOptions) values() url.Values {
	q := make(url.Values)
	setString(q, "omit_empty", o.OmitEmpty)
	setString(q, "status", o.Status)
	return q
}

func (o ListCommentsOptions) values() url.Values {
	q := o.ListOptions.values()
	setString(q, "parent_entity_type", o.ParentEntityType)
	setInt64Ptr(q, "parent_entity_id", o.ParentEntityID)
	setBoolPtr(q, "get_positions", o.GetPositions)
	setBoolPtr(q, "holders_only", o.HoldersOnly)
	return q
}

func (o GetCommentOptions) values() url.Values {
	q := make(url.Values)
	setBoolPtr(q, "get_positions", o.GetPositions)
	return q
}

func (o ListSeriesOptions) values() url.Values {
	q := o.ListOptions.values()
	addStringSlice(q, "slug", o.Slug)
	addInt64Slice(q, "categories_ids", o.CategoriesIDs)
	addStringSlice(q, "categories_labels", o.CategoriesLabels)
	setBoolPtr(q, "closed", o.Closed)
	setBoolPtr(q, "include_chat", o.IncludeChat)
	setString(q, "recurrence", o.Recurrence)
	setBoolPtr(q, "exclude_events", o.ExcludeEvents)
	return q
}

func (o GetSeriesOptions) values() url.Values {
	q := make(url.Values)
	setBoolPtr(q, "include_chat", o.IncludeChat)
	return q
}

func (o PublicSearchOptions) values() url.Values {
	q := make(url.Values)
	setString(q, "q", o.Q)
	setBoolPtr(q, "cache", o.Cache)
	setString(q, "events_status", o.EventsStatus)
	setIntPtr(q, "limit_per_type", o.LimitPerType)
	setIntPtr(q, "page", o.Page)
	addStringSlice(q, "events_tag", o.EventsTag)
	setIntPtr(q, "keep_closed_markets", o.KeepClosedMarkets)
	setString(q, "sort", o.Sort)
	setBoolPtr(q, "ascending", o.Ascending)
	setBoolPtr(q, "search_tags", o.SearchTags)
	setBoolPtr(q, "search_profiles", o.SearchProfiles)
	setString(q, "recurrence", o.Recurrence)
	addInt64Slice(q, "exclude_tag_id", o.ExcludeTagID)
	setBoolPtr(q, "optimized", o.Optimized)
	return q
}

func (o ListTeamsOptions) values() url.Values {
	q := o.ListOptions.values()
	addStringSlice(q, "league", o.League)
	addStringSlice(q, "name", o.Name)
	addStringSlice(q, "abbreviation", o.Abbreviation)
	return q
}

func pathEscapeInt(v int64) string {
	return url.PathEscape(strconv.FormatInt(v, 10))
}

func setString(query url.Values, key, value string) {
	if strings.TrimSpace(value) != "" {
		query.Set(key, strings.TrimSpace(value))
	}
}

func setBoolPtr(query url.Values, key string, value *bool) {
	if value != nil {
		query.Set(key, strconv.FormatBool(*value))
	}
}

func setIntPtr(query url.Values, key string, value *int) {
	if value != nil {
		query.Set(key, strconv.Itoa(*value))
	}
}

func setInt64Ptr(query url.Values, key string, value *int64) {
	if value != nil {
		query.Set(key, strconv.FormatInt(*value, 10))
	}
}

func setFloat64Ptr(query url.Values, key string, value *float64) {
	if value != nil {
		query.Set(key, strconv.FormatFloat(*value, 'f', -1, 64))
	}
}

func setTimePtr(query url.Values, key string, value *time.Time) {
	if value != nil {
		query.Set(key, value.UTC().Format(time.RFC3339))
	}
}

func addStringSlice(query url.Values, key string, values []string) {
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			query.Add(key, trimmed)
		}
	}
}

func addInt64Slice(query url.Values, key string, values []int64) {
	for _, v := range values {
		query.Add(key, strconv.FormatInt(v, 10))
	}
}
