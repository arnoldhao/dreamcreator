package usage

import (
	"context"
	"time"
)

const (
	CategoryTokens       = "tokens"
	CategoryContextToken = "context_tokens"
	CategoryTTS          = "tts"

	RequestSourceDialogue = "dialogue"
	RequestSourceRelay    = "relay"
	RequestSourceOneShot  = "one-shot"
	RequestSourceUnknown  = "unknown"

	CostBasisEstimated  = "estimated"
	CostBasisReconciled = "reconciled"
)

type UsageEvent struct {
	ID                string
	RequestID         string
	StepID            string
	ProviderID        string
	ModelName         string
	Category          string
	Channel           string
	RequestSource     string
	UsageStatus       string
	InputTokens       int
	OutputTokens      int
	TotalTokens       int
	CachedInputTokens int
	ReasoningTokens   int
	AudioInputTokens  int
	AudioOutputTokens int
	RawUsageJSON      string
	OccurredAt        time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type LedgerEntry struct {
	ID                    string
	EventID               string
	Category              string
	ProviderID            string
	ModelName             string
	Channel               string
	RequestID             string
	StepID                string
	RequestSource         string
	CostBasis             string
	PricingVersionID      string
	Units                 int
	InputTokens           int
	OutputTokens          int
	CachedInputTokens     int
	ReasoningTokens       int
	InputCostMicros       int64
	OutputCostMicros      int64
	CachedInputCostMicros int64
	ReasoningCostMicros   int64
	RequestCostMicros     int64
	CostMicros            int64
	RawUsageJSON          string
	OccurredAt            time.Time
	CreatedAt             time.Time
}

type QueryFilter struct {
	StartAt       time.Time
	EndAt         time.Time
	ProviderID    string
	ModelName     string
	Channel       string
	Category      string
	RequestSource string
	CostBasis     string
}

type PricingVersion struct {
	ID                    string     `json:"id"`
	ProviderID            string     `json:"providerId"`
	ModelName             string     `json:"modelName"`
	Currency              string     `json:"currency"`
	InputPerMillion       float64    `json:"inputPerMillion"`
	OutputPerMillion      float64    `json:"outputPerMillion"`
	CachedInputPerMillion float64    `json:"cachedInputPerMillion"`
	ReasoningPerMillion   float64    `json:"reasoningPerMillion"`
	AudioInputPerMillion  float64    `json:"audioInputPerMillion"`
	AudioOutputPerMillion float64    `json:"audioOutputPerMillion"`
	PerRequest            float64    `json:"perRequest"`
	Source                string     `json:"source"`
	EffectiveFrom         time.Time  `json:"effectiveFrom"`
	EffectiveTo           *time.Time `json:"effectiveTo,omitempty"`
	IsActive              bool       `json:"isActive"`
	UpdatedBy             string     `json:"updatedBy,omitempty"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type PricingVersionFilter struct {
	ProviderID string
	ModelName  string
	Source     string
	ActiveOnly bool
}

type Repository interface {
	UpsertEvent(ctx context.Context, event UsageEvent) (UsageEvent, error)
	UpsertLedger(ctx context.Context, entry LedgerEntry) error
	ListLedger(ctx context.Context, filter QueryFilter) ([]LedgerEntry, error)
	ResolvePricingVersion(ctx context.Context, providerID string, modelName string, at time.Time) (PricingVersion, bool, error)
	ListPricingVersions(ctx context.Context, filter PricingVersionFilter) ([]PricingVersion, error)
	UpsertPricingVersion(ctx context.Context, version PricingVersion) (PricingVersion, error)
	DeletePricingVersion(ctx context.Context, id string) error
	ActivatePricingVersion(ctx context.Context, id string) error
}

type UsageStatusRequest struct {
	Window                string   `json:"window,omitempty"`
	StartAt               string   `json:"startAt,omitempty"`
	EndAt                 string   `json:"endAt,omitempty"`
	GroupBy               []string `json:"groupBy,omitempty"`
	ProviderID            string   `json:"providerId,omitempty"`
	ModelName             string   `json:"modelName,omitempty"`
	Channel               string   `json:"channel,omitempty"`
	Category              string   `json:"category,omitempty"`
	RequestSource         string   `json:"requestSource,omitempty"`
	CostBasis             string   `json:"costBasis,omitempty"`
	TimezoneOffsetMinutes int      `json:"timezoneOffsetMinutes,omitempty"`
}

type UsageTotals struct {
	Requests          int   `json:"requests"`
	Units             int   `json:"units"`
	InputTokens       int   `json:"inputTokens"`
	OutputTokens      int   `json:"outputTokens"`
	CachedInputTokens int   `json:"cachedInputTokens"`
	ReasoningTokens   int   `json:"reasoningTokens"`
	CostMicros        int64 `json:"costMicros"`
}

type UsageBucket struct {
	Key               string `json:"key"`
	BucketStart       string `json:"bucketStart,omitempty"`
	BucketEnd         string `json:"bucketEnd,omitempty"`
	ProviderID        string `json:"providerId,omitempty"`
	ModelName         string `json:"modelName,omitempty"`
	Channel           string `json:"channel,omitempty"`
	Category          string `json:"category,omitempty"`
	RequestSource     string `json:"requestSource,omitempty"`
	CostBasis         string `json:"costBasis,omitempty"`
	Requests          int    `json:"requests"`
	Units             int    `json:"units"`
	InputTokens       int    `json:"inputTokens"`
	OutputTokens      int    `json:"outputTokens"`
	CachedInputTokens int    `json:"cachedInputTokens"`
	ReasoningTokens   int    `json:"reasoningTokens"`
	CostMicros        int64  `json:"costMicros"`
}

type UsageStatusResponse struct {
	Window  string        `json:"window,omitempty"`
	Totals  UsageTotals   `json:"totals"`
	Buckets []UsageBucket `json:"buckets"`
}

type UsageCostRequest struct {
	Window                string   `json:"window,omitempty"`
	StartAt               string   `json:"startAt,omitempty"`
	EndAt                 string   `json:"endAt,omitempty"`
	GroupBy               []string `json:"groupBy,omitempty"`
	ProviderID            string   `json:"providerId,omitempty"`
	ModelName             string   `json:"modelName,omitempty"`
	Channel               string   `json:"channel,omitempty"`
	Category              string   `json:"category,omitempty"`
	RequestSource         string   `json:"requestSource,omitempty"`
	CostBasis             string   `json:"costBasis,omitempty"`
	TimezoneOffsetMinutes int      `json:"timezoneOffsetMinutes,omitempty"`
}

type UsageCostLine struct {
	BucketStart   string `json:"bucketStart,omitempty"`
	BucketEnd     string `json:"bucketEnd,omitempty"`
	ProviderID    string `json:"providerId,omitempty"`
	ModelName     string `json:"modelName,omitempty"`
	Channel       string `json:"channel,omitempty"`
	Category      string `json:"category,omitempty"`
	RequestSource string `json:"requestSource,omitempty"`
	CostBasis     string `json:"costBasis,omitempty"`
	Requests      int    `json:"requests"`
	CostMicros    int64  `json:"costMicros"`
}

type UsageCostResponse struct {
	Window          string          `json:"window,omitempty"`
	TotalCostMicros int64           `json:"totalCostMicros"`
	Lines           []UsageCostLine `json:"lines"`
}

type PricingListRequest struct {
	ProviderID string `json:"providerId,omitempty"`
	ModelName  string `json:"modelName,omitempty"`
	Source     string `json:"source,omitempty"`
	ActiveOnly bool   `json:"activeOnly,omitempty"`
}

type PricingListResponse struct {
	Items []PricingVersion `json:"items"`
}

type PricingUpsertRequest struct {
	ID                    string  `json:"id,omitempty"`
	ProviderID            string  `json:"providerId"`
	ModelName             string  `json:"modelName"`
	Currency              string  `json:"currency,omitempty"`
	InputPerMillion       float64 `json:"inputPerMillion"`
	OutputPerMillion      float64 `json:"outputPerMillion"`
	CachedInputPerMillion float64 `json:"cachedInputPerMillion,omitempty"`
	ReasoningPerMillion   float64 `json:"reasoningPerMillion,omitempty"`
	AudioInputPerMillion  float64 `json:"audioInputPerMillion,omitempty"`
	AudioOutputPerMillion float64 `json:"audioOutputPerMillion,omitempty"`
	PerRequest            float64 `json:"perRequest,omitempty"`
	Source                string  `json:"source,omitempty"`
	EffectiveFrom         string  `json:"effectiveFrom"`
	EffectiveTo           string  `json:"effectiveTo,omitempty"`
	IsActive              bool    `json:"isActive,omitempty"`
	UpdatedBy             string  `json:"updatedBy,omitempty"`
}

type PricingDeleteRequest struct {
	ID string `json:"id"`
}

type PricingActivateRequest struct {
	ID string `json:"id"`
}
