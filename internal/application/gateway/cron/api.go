package cron

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	domainsession "dreamcreator/internal/domain/session"
)

type ScheduleDTO struct {
	Kind      string `json:"kind"`
	At        string `json:"at,omitempty"`
	EveryMs   int64  `json:"everyMs,omitempty"`
	AnchorMs  int64  `json:"anchorMs,omitempty"`
	Expr      string `json:"expr,omitempty"`
	TZ        string `json:"tz,omitempty"`
	StaggerMs int64  `json:"staggerMs,omitempty"`
}

type PayloadDTO struct {
	Kind           string `json:"kind"`
	Text           string `json:"text,omitempty"`
	Message        string `json:"message,omitempty"`
	Model          string `json:"model,omitempty"`
	Thinking       string `json:"thinking,omitempty"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
	LightContext   bool   `json:"lightContext,omitempty"`
}

type FailureDestinationDTO struct {
	Mode      string `json:"mode,omitempty"`
	Channel   string `json:"channel,omitempty"`
	To        string `json:"to,omitempty"`
	AccountID string `json:"accountId,omitempty"`
}

type DeliveryDTO struct {
	Mode               string                 `json:"mode"`
	Channel            string                 `json:"channel,omitempty"`
	To                 string                 `json:"to,omitempty"`
	AccountID          string                 `json:"accountId,omitempty"`
	BestEffort         bool                   `json:"bestEffort,omitempty"`
	FailureDestination *FailureDestinationDTO `json:"failureDestination,omitempty"`
}

type CreateInput struct {
	ID             string       `json:"id,omitempty"`
	AssistantID    string       `json:"assistantId,omitempty"`
	Name           string       `json:"name"`
	Description    string       `json:"description,omitempty"`
	Enabled        bool         `json:"enabled"`
	DeleteAfterRun bool         `json:"deleteAfterRun,omitempty"`
	Schedule       ScheduleDTO  `json:"schedule"`
	SessionTarget  string       `json:"sessionTarget"`
	WakeMode       string       `json:"wakeMode"`
	Payload        PayloadDTO   `json:"payload"`
	Delivery       *DeliveryDTO `json:"delivery,omitempty"`
	SessionKey     string       `json:"sessionKey,omitempty"`
}

type PatchInput struct {
	AssistantID    *string      `json:"assistantId,omitempty"`
	Name           *string      `json:"name,omitempty"`
	Description    *string      `json:"description,omitempty"`
	Enabled        *bool        `json:"enabled,omitempty"`
	DeleteAfterRun *bool        `json:"deleteAfterRun,omitempty"`
	Schedule       *ScheduleDTO `json:"schedule,omitempty"`
	SessionTarget  *string      `json:"sessionTarget,omitempty"`
	WakeMode       *string      `json:"wakeMode,omitempty"`
	Payload        *PayloadDTO  `json:"payload,omitempty"`
	Delivery       *DeliveryDTO `json:"delivery,omitempty"`
	SessionKey     *string      `json:"sessionKey,omitempty"`
}

type UpdateInput struct {
	ID    string     `json:"id"`
	Patch PatchInput `json:"patch"`
}

type RemoveInput struct {
	ID string `json:"id"`
}

type RunInput struct {
	ID   string `json:"id"`
	Mode string `json:"mode"`
}

type RunsQuery struct {
	Scope            string   `json:"scope,omitempty"`
	ID               string   `json:"id,omitempty"`
	Statuses         []string `json:"statuses,omitempty"`
	DeliveryStatuses []string `json:"deliveryStatuses,omitempty"`
	Query            string   `json:"query,omitempty"`
	SortDir          string   `json:"sortDir,omitempty"`
	Limit            int      `json:"limit,omitempty"`
	Offset           int      `json:"offset,omitempty"`
}

type ListInput struct {
	IncludeDisabled bool   `json:"includeDisabled,omitempty"`
	Enabled         string `json:"enabled,omitempty"`
	Query           string `json:"query,omitempty"`
	SortBy          string `json:"sortBy,omitempty"`
	SortDir         string `json:"sortDir,omitempty"`
	Limit           int    `json:"limit,omitempty"`
	Offset          int    `json:"offset,omitempty"`
}

type WakeInput struct {
	Mode       string `json:"mode,omitempty"`
	Text       string `json:"text"`
	SessionKey string `json:"sessionKey,omitempty"`
}

type RunDetailInput struct {
	RunID       string `json:"runId"`
	EventsLimit int    `json:"eventsLimit,omitempty"`
}

type RunEventsInput struct {
	RunID   string `json:"runId"`
	SortDir string `json:"sortDir,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

func decodeStrictJSON(data []byte, v any) error {
	if len(data) == 0 {
		return errors.New("params are required")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	return nil
}

func validateSchedule(input ScheduleDTO) error {
	kind := strings.TrimSpace(strings.ToLower(input.Kind))
	switch kind {
	case "at":
		if strings.TrimSpace(input.At) == "" {
			return errors.New("schedule.at is required when schedule.kind=at")
		}
		if _, err := parseAtTime(input.At, strings.TrimSpace(input.TZ)); err != nil {
			return fmt.Errorf("invalid schedule.at: %w", err)
		}
	case "every":
		if input.EveryMs <= 0 {
			return errors.New("schedule.everyMs must be > 0 when schedule.kind=every")
		}
	case "cron":
		expr := strings.TrimSpace(input.Expr)
		if expr == "" {
			return errors.New("schedule.expr is required when schedule.kind=cron")
		}
		if _, err := parseCronExpr(expr); err != nil {
			return fmt.Errorf("invalid schedule.expr: %w", err)
		}
	default:
		return errors.New("schedule.kind must be one of: at, every, cron")
	}
	return nil
}

func validatePayload(input PayloadDTO) error {
	kind := strings.TrimSpace(strings.ToLower(input.Kind))
	switch kind {
	case "systemevent":
		if strings.TrimSpace(input.Text) == "" {
			return errors.New("payload.text is required when payload.kind=systemEvent (hint: use payload.text for systemEvent, payload.message for agentTurn)")
		}
	case "agentturn":
		if strings.TrimSpace(input.Message) == "" {
			return errors.New("payload.message is required when payload.kind=agentTurn (hint: use payload.message for agentTurn, not payload.text)")
		}
		if input.TimeoutSeconds < 0 {
			return errors.New("payload.timeoutSeconds must be >= 0")
		}
	default:
		return errors.New("payload.kind must be one of: systemEvent, agentTurn")
	}
	return nil
}

func validateFailureDestination(input *FailureDestinationDTO) error {
	if input == nil {
		return nil
	}
	mode := strings.TrimSpace(strings.ToLower(input.Mode))
	if mode == "" {
		mode = "announce"
	}
	switch mode {
	case "announce":
		channel := normalizeAnnounceChannel(input.Channel)
		if !isValidAnnounceChannel(channel) {
			return errors.New("delivery.failureDestination.channel must be one of: default, app, telegram when delivery.failureDestination.mode=announce")
		}
		return nil
	case "webhook":
		target := strings.ToLower(strings.TrimSpace(input.To))
		if target == "" {
			return errors.New("delivery.failureDestination.to is required when delivery.failureDestination.mode=webhook")
		}
		if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
			return errors.New("delivery.failureDestination.to must start with http:// or https:// when delivery.failureDestination.mode=webhook")
		}
		return nil
	default:
		return errors.New("delivery.failureDestination.mode must be one of: announce, webhook")
	}
}

func validateDelivery(input *DeliveryDTO) error {
	if input == nil {
		return nil
	}
	mode := strings.TrimSpace(strings.ToLower(input.Mode))
	switch mode {
	case "none":
		return nil
	case "announce":
		channel := normalizeAnnounceChannel(input.Channel)
		if !isValidAnnounceChannel(channel) {
			return errors.New("delivery.channel must be one of: default, app, telegram when delivery.mode=announce")
		}
		return validateFailureDestination(input.FailureDestination)
	case "webhook":
		if strings.TrimSpace(input.To) == "" {
			return errors.New("delivery.to is required when delivery.mode=webhook")
		}
		target := strings.ToLower(strings.TrimSpace(input.To))
		if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
			return errors.New("delivery.to must start with http:// or https:// when delivery.mode=webhook")
		}
		return validateFailureDestination(input.FailureDestination)
	default:
		return errors.New("delivery.mode must be one of: none, announce, webhook")
	}
}

func validateTargetAndPayload(sessionTarget string, payloadKind string) error {
	target := strings.TrimSpace(strings.ToLower(sessionTarget))
	payload := strings.TrimSpace(strings.ToLower(payloadKind))
	switch target {
	case "main":
		if payload != "systemevent" {
			return errors.New("sessionTarget=main requires payload.kind=systemEvent (hint: use payload.text with systemEvent)")
		}
	case "isolated":
		if payload != "agentturn" {
			return errors.New("sessionTarget=isolated requires payload.kind=agentTurn (hint: use payload.message with agentTurn)")
		}
	default:
		return errors.New("sessionTarget must be one of: main, isolated")
	}
	return nil
}

func normalizeWakeModeDTO(mode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", "next-heartbeat":
		return "next-heartbeat", nil
	case "now":
		return "now", nil
	default:
		return "", errors.New("wakeMode must be one of: next-heartbeat, now")
	}
}

func normalizeRunMode(mode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", "force":
		return "force", nil
	case "due":
		return "due", nil
	default:
		return "", errors.New("mode must be one of: due, force")
	}
}

func (input CreateInput) Validate() error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateSchedule(input.Schedule); err != nil {
		return err
	}
	if err := validatePayload(input.Payload); err != nil {
		return err
	}
	if err := validateTargetAndPayload(input.SessionTarget, input.Payload.Kind); err != nil {
		return err
	}
	if _, err := normalizeWakeModeDTO(input.WakeMode); err != nil {
		return err
	}
	if err := validateDelivery(input.Delivery); err != nil {
		return err
	}
	return nil
}

func (input UpdateInput) Validate() error {
	if strings.TrimSpace(input.ID) == "" {
		return errors.New("id is required")
	}
	patch := input.Patch
	if patch.Name != nil && strings.TrimSpace(*patch.Name) == "" {
		return errors.New("name cannot be empty")
	}
	if patch.Schedule != nil {
		if err := validateSchedule(*patch.Schedule); err != nil {
			return err
		}
	}
	if patch.Payload != nil {
		if err := validatePayload(*patch.Payload); err != nil {
			return err
		}
	}
	if patch.WakeMode != nil {
		if _, err := normalizeWakeModeDTO(*patch.WakeMode); err != nil {
			return err
		}
	}
	if err := validateDelivery(patch.Delivery); err != nil {
		return err
	}
	if patch.SessionTarget != nil || patch.Payload != nil {
		target := ""
		if patch.SessionTarget != nil {
			target = *patch.SessionTarget
		}
		payloadKind := ""
		if patch.Payload != nil {
			payloadKind = patch.Payload.Kind
		}
		if target != "" && payloadKind != "" {
			if err := validateTargetAndPayload(target, payloadKind); err != nil {
				return err
			}
		}
	}
	return nil
}

func (input RemoveInput) Validate() error {
	if strings.TrimSpace(input.ID) == "" {
		return errors.New("id is required")
	}
	return nil
}

func (input RunInput) Validate() error {
	if strings.TrimSpace(input.ID) == "" {
		return errors.New("id is required")
	}
	if _, err := normalizeRunMode(input.Mode); err != nil {
		return err
	}
	return nil
}

func (input RunsQuery) Validate() error {
	scope := strings.TrimSpace(strings.ToLower(input.Scope))
	if scope != "" && scope != "job" && scope != "all" {
		return errors.New("scope must be one of: job, all")
	}
	if scope == "job" && strings.TrimSpace(input.ID) == "" {
		return errors.New("id is required when scope=job")
	}
	if input.Limit < 0 {
		return errors.New("limit must be >= 0")
	}
	if input.Offset < 0 {
		return errors.New("offset must be >= 0")
	}
	sortDir := strings.TrimSpace(strings.ToLower(input.SortDir))
	if sortDir != "" && sortDir != "asc" && sortDir != "desc" {
		return errors.New("sortDir must be one of: asc, desc")
	}
	return nil
}

func (input ListInput) Validate() error {
	if input.Limit < 0 {
		return errors.New("limit must be >= 0")
	}
	if input.Offset < 0 {
		return errors.New("offset must be >= 0")
	}
	enabled := strings.TrimSpace(strings.ToLower(input.Enabled))
	if enabled != "" && enabled != "all" && enabled != "enabled" && enabled != "disabled" {
		return errors.New("enabled must be one of: all, enabled, disabled")
	}
	sortBy := strings.TrimSpace(strings.ToLower(input.SortBy))
	if sortBy != "" && sortBy != "nextrunatms" && sortBy != "updatedatms" && sortBy != "name" {
		return errors.New("sortBy must be one of: nextRunAtMs, updatedAtMs, name")
	}
	sortDir := strings.TrimSpace(strings.ToLower(input.SortDir))
	if sortDir != "" && sortDir != "asc" && sortDir != "desc" {
		return errors.New("sortDir must be one of: asc, desc")
	}
	return nil
}

func (input WakeInput) Validate() error {
	if strings.TrimSpace(input.Text) == "" {
		return errors.New("text is required")
	}
	_, err := normalizeWakeModeDTO(input.Mode)
	return err
}

func (input RunDetailInput) Validate() error {
	if strings.TrimSpace(input.RunID) == "" {
		return errors.New("runId is required")
	}
	if input.EventsLimit < 0 {
		return errors.New("eventsLimit must be >= 0")
	}
	return nil
}

func (input RunEventsInput) Validate() error {
	if strings.TrimSpace(input.RunID) == "" {
		return errors.New("runId is required")
	}
	if input.Limit < 0 {
		return errors.New("limit must be >= 0")
	}
	if input.Offset < 0 {
		return errors.New("offset must be >= 0")
	}
	sortDir := strings.TrimSpace(strings.ToLower(input.SortDir))
	if sortDir != "" && sortDir != "asc" && sortDir != "desc" {
		return errors.New("sortDir must be one of: asc, desc")
	}
	return nil
}

func DecodeCreateInput(params []byte) (CreateInput, error) {
	input := CreateInput{}
	if err := decodeStrictJSON(params, &input); err != nil {
		return CreateInput{}, err
	}
	if err := input.Validate(); err != nil {
		return CreateInput{}, err
	}
	return input, nil
}

func DecodeUpdateInput(params []byte) (UpdateInput, error) {
	input := UpdateInput{}
	if err := decodeStrictJSON(params, &input); err != nil {
		return UpdateInput{}, err
	}
	if err := input.Validate(); err != nil {
		return UpdateInput{}, err
	}
	return input, nil
}

func DecodeRemoveInput(params []byte) (RemoveInput, error) {
	input := RemoveInput{}
	if err := decodeStrictJSON(params, &input); err != nil {
		return RemoveInput{}, err
	}
	if err := input.Validate(); err != nil {
		return RemoveInput{}, err
	}
	return input, nil
}

func DecodeRunInput(params []byte) (RunInput, error) {
	input := RunInput{}
	if err := decodeStrictJSON(params, &input); err != nil {
		return RunInput{}, err
	}
	if err := input.Validate(); err != nil {
		return RunInput{}, err
	}
	return input, nil
}

func DecodeRunsQuery(params []byte) (RunsQuery, error) {
	if len(params) == 0 {
		return RunsQuery{}, nil
	}
	input := RunsQuery{}
	if err := decodeStrictJSON(params, &input); err != nil {
		return RunsQuery{}, err
	}
	if err := input.Validate(); err != nil {
		return RunsQuery{}, err
	}
	return input, nil
}

func DecodeListInput(params []byte) (ListInput, error) {
	if len(params) == 0 {
		return ListInput{}, nil
	}
	input := ListInput{}
	if err := decodeStrictJSON(params, &input); err != nil {
		return ListInput{}, err
	}
	if err := input.Validate(); err != nil {
		return ListInput{}, err
	}
	return input, nil
}

func DecodeWakeInput(params []byte) (WakeInput, error) {
	input := WakeInput{}
	if err := decodeStrictJSON(params, &input); err != nil {
		return WakeInput{}, err
	}
	if err := input.Validate(); err != nil {
		return WakeInput{}, err
	}
	return input, nil
}

func DecodeRunDetailInput(params []byte) (RunDetailInput, error) {
	input := RunDetailInput{}
	if err := decodeStrictJSON(params, &input); err != nil {
		return RunDetailInput{}, err
	}
	if err := input.Validate(); err != nil {
		return RunDetailInput{}, err
	}
	return input, nil
}

func DecodeRunEventsInput(params []byte) (RunEventsInput, error) {
	input := RunEventsInput{}
	if err := decodeStrictJSON(params, &input); err != nil {
		return RunEventsInput{}, err
	}
	if err := input.Validate(); err != nil {
		return RunEventsInput{}, err
	}
	return input, nil
}

func BuildCronJobFromCreate(input CreateInput, now time.Time) CronJob {
	id := strings.TrimSpace(input.ID)
	wakeMode, _ := normalizeWakeModeDTO(input.WakeMode)
	var delivery *CronDelivery
	if input.Delivery != nil {
		delivery = &CronDelivery{
			Mode:       strings.TrimSpace(input.Delivery.Mode),
			Channel:    strings.TrimSpace(input.Delivery.Channel),
			To:         strings.TrimSpace(input.Delivery.To),
			AccountID:  strings.TrimSpace(input.Delivery.AccountID),
			BestEffort: input.Delivery.BestEffort,
		}
		if input.Delivery.FailureDestination != nil {
			delivery.FailureDestination = &CronFailureDestination{
				Mode:      strings.TrimSpace(input.Delivery.FailureDestination.Mode),
				Channel:   strings.TrimSpace(input.Delivery.FailureDestination.Channel),
				To:        strings.TrimSpace(input.Delivery.FailureDestination.To),
				AccountID: strings.TrimSpace(input.Delivery.FailureDestination.AccountID),
			}
		}
	}
	return CronJob{
		ID:             id,
		JobID:          id,
		Name:           strings.TrimSpace(input.Name),
		Description:    strings.TrimSpace(input.Description),
		AssistantID:    "",
		Enabled:        input.Enabled,
		DeleteAfterRun: input.DeleteAfterRun,
		Schedule: CronSchedule{
			Kind:      strings.TrimSpace(input.Schedule.Kind),
			At:        strings.TrimSpace(input.Schedule.At),
			EveryMs:   input.Schedule.EveryMs,
			AnchorMs:  input.Schedule.AnchorMs,
			Expr:      strings.TrimSpace(input.Schedule.Expr),
			TZ:        strings.TrimSpace(input.Schedule.TZ),
			StaggerMs: input.Schedule.StaggerMs,
		},
		SessionTarget: strings.TrimSpace(input.SessionTarget),
		WakeMode:      wakeMode,
		PayloadSpec: CronPayload{
			Kind:           strings.TrimSpace(input.Payload.Kind),
			Text:           strings.TrimSpace(input.Payload.Text),
			Message:        strings.TrimSpace(input.Payload.Message),
			Model:          strings.TrimSpace(input.Payload.Model),
			Thinking:       strings.TrimSpace(input.Payload.Thinking),
			TimeoutSeconds: input.Payload.TimeoutSeconds,
			LightContext:   input.Payload.LightContext,
		},
		Delivery:   delivery,
		SessionKey: strings.TrimSpace(input.SessionKey),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func ApplyPatch(job CronJob, patch PatchInput) CronJob {
	if patch.Name != nil {
		job.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.Description != nil {
		job.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.Enabled != nil {
		job.Enabled = *patch.Enabled
	}
	if patch.DeleteAfterRun != nil {
		job.DeleteAfterRun = *patch.DeleteAfterRun
	}
	if patch.SessionTarget != nil {
		job.SessionTarget = strings.TrimSpace(*patch.SessionTarget)
	}
	if patch.WakeMode != nil {
		wakeMode, _ := normalizeWakeModeDTO(*patch.WakeMode)
		job.WakeMode = wakeMode
	}
	if patch.SessionKey != nil {
		job.SessionKey = strings.TrimSpace(*patch.SessionKey)
	}
	if patch.Schedule != nil {
		job.Schedule = CronSchedule{
			Kind:      strings.TrimSpace(patch.Schedule.Kind),
			At:        strings.TrimSpace(patch.Schedule.At),
			EveryMs:   patch.Schedule.EveryMs,
			AnchorMs:  patch.Schedule.AnchorMs,
			Expr:      strings.TrimSpace(patch.Schedule.Expr),
			TZ:        strings.TrimSpace(patch.Schedule.TZ),
			StaggerMs: patch.Schedule.StaggerMs,
		}
	}
	if patch.Payload != nil {
		job.PayloadSpec = CronPayload{
			Kind:           strings.TrimSpace(patch.Payload.Kind),
			Text:           strings.TrimSpace(patch.Payload.Text),
			Message:        strings.TrimSpace(patch.Payload.Message),
			Model:          strings.TrimSpace(patch.Payload.Model),
			Thinking:       strings.TrimSpace(patch.Payload.Thinking),
			TimeoutSeconds: patch.Payload.TimeoutSeconds,
			LightContext:   patch.Payload.LightContext,
		}
	}
	if patch.Delivery != nil {
		job.Delivery = &CronDelivery{
			Mode:       strings.TrimSpace(patch.Delivery.Mode),
			Channel:    strings.TrimSpace(patch.Delivery.Channel),
			To:         strings.TrimSpace(patch.Delivery.To),
			AccountID:  strings.TrimSpace(patch.Delivery.AccountID),
			BestEffort: patch.Delivery.BestEffort,
		}
		if patch.Delivery.FailureDestination != nil {
			job.Delivery.FailureDestination = &CronFailureDestination{
				Mode:      strings.TrimSpace(patch.Delivery.FailureDestination.Mode),
				Channel:   strings.TrimSpace(patch.Delivery.FailureDestination.Channel),
				To:        strings.TrimSpace(patch.Delivery.FailureDestination.To),
				AccountID: strings.TrimSpace(patch.Delivery.FailureDestination.AccountID),
			}
		}
	}
	return job
}

type JobResponse struct {
	ID             string        `json:"id"`
	AssistantID    string        `json:"assistantId"`
	Name           string        `json:"name"`
	Description    string        `json:"description,omitempty"`
	Enabled        bool          `json:"enabled"`
	DeleteAfterRun bool          `json:"deleteAfterRun,omitempty"`
	Schedule       CronSchedule  `json:"schedule"`
	SessionTarget  string        `json:"sessionTarget"`
	WakeMode       string        `json:"wakeMode"`
	Payload        CronPayload   `json:"payload"`
	Delivery       *CronDelivery `json:"delivery,omitempty"`
	SourceChannel  string        `json:"sourceChannel,omitempty"`
	State          CronJobState  `json:"state"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}

func ToJobResponse(job CronJob) JobResponse {
	return JobResponse{
		ID:             strings.TrimSpace(job.JobID),
		AssistantID:    strings.TrimSpace(job.AssistantID),
		Name:           strings.TrimSpace(job.Name),
		Description:    strings.TrimSpace(job.Description),
		Enabled:        job.Enabled,
		DeleteAfterRun: job.DeleteAfterRun,
		Schedule:       job.Schedule,
		SessionTarget:  job.SessionTarget,
		WakeMode:       job.WakeMode,
		Payload:        job.PayloadSpec,
		Delivery:       job.Delivery,
		SourceChannel:  resolveCronSourceChannel(strings.TrimSpace(job.SessionKey)),
		State:          job.State,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
	}
}

func resolveCronSourceChannel(sessionKey string) string {
	parts, _, err := domainsession.NormalizeSessionKey(strings.TrimSpace(sessionKey))
	if err != nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(parts.Channel)) {
	case "app", "aui":
		return "app"
	default:
		return strings.ToLower(strings.TrimSpace(parts.Channel))
	}
}
