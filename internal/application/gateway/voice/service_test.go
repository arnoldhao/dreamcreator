package voice

import (
	"context"
	"os"
	"testing"
	"time"
)

type memoryConfigRepo struct {
	config VoiceConfig
}

func (repo *memoryConfigRepo) Get(_ context.Context) (VoiceConfig, error) {
	if repo.config.Version == 0 {
		repo.config = DefaultConfig()
	}
	return repo.config, nil
}

func (repo *memoryConfigRepo) Save(_ context.Context, config VoiceConfig) error {
	repo.config = config
	return nil
}

type memoryJobRepo struct{}

func (repo *memoryJobRepo) Save(_ context.Context, _ TTSJob) error {
	return nil
}

func TestVoiceWakeVersionIncrement(t *testing.T) {
	configRepo := &memoryConfigRepo{}
	service := NewService(configRepo, &memoryJobRepo{}, nil, nil, nil)

	initial, err := service.VoiceWakeGet(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if initial.Version != 1 {
		t.Fatalf("expected version 1, got %d", initial.Version)
	}
	updated, err := service.VoiceWakeSet(context.Background(), VoiceWakeSetRequest{Triggers: []string{"hello"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}
}

func TestTalkLocksVoiceOnConvert(t *testing.T) {
	configRepo := &memoryConfigRepo{}
	service := NewService(configRepo, &memoryJobRepo{}, nil, nil, nil)
	service.now = func() time.Time { return time.Date(2026, 2, 19, 12, 0, 0, 0, time.UTC) }

	service.TalkMode(TalkModeRequest{Enabled: true, Phase: "listening"})
	resp, err := service.Convert(context.Background(), TTSConvertRequest{
		Text:   "hello",
		Format: "wav",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Artifact.Path != "" {
		_ = os.Remove(resp.Artifact.Path)
	}
	state := service.TalkMode(TalkModeRequest{Enabled: true})
	if !state.VoiceLocked {
		t.Fatalf("expected voice locked to be true")
	}
}
