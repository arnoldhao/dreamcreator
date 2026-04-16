package llmrecord

import (
	"context"
	"errors"
	"testing"
	"time"

	settingsdto "dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/infrastructure/llm"
)

func TestServiceStartLLMCallDisabledStrategySkipsInsert(t *testing.T) {
	t.Parallel()

	repo := newStubRepository()
	service := NewService(repo, stubSettingsReader{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{
				Runtime: settingsdto.GatewayRuntimeSettings{
					CallRecords: settingsdto.GatewayCallRecordsSettings{
						SaveStrategy:  "off",
						RetentionDays: 7,
						AutoCleanup:   "off",
					},
				},
			},
		},
	})

	id, err := service.StartLLMCall(context.Background(), llm.CallRecordStart{
		ThreadID:       "thread-1",
		RequestPayload: `{"model":"gpt-5"}`,
	})
	if err != nil {
		t.Fatalf("start llm call: %v", err)
	}
	if id != "" {
		t.Fatalf("expected no persisted id when strategy=off, got %q", id)
	}
	if repo.insertCount != 0 {
		t.Fatalf("expected no insert when strategy=off, got %d", repo.insertCount)
	}
}

func TestServiceFinishLLMCallErrorsStrategyDeletesSuccessfulRecords(t *testing.T) {
	t.Parallel()

	repo := newStubRepository()
	repo.items["call-1"] = Record{
		ID:        "call-1",
		Status:    llm.CallRecordStatusStarted,
		StartedAt: time.Date(2026, 4, 17, 8, 0, 0, 0, time.UTC),
	}
	service := NewService(repo, stubSettingsReader{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{
				Runtime: settingsdto.GatewayRuntimeSettings{
					CallRecords: settingsdto.GatewayCallRecordsSettings{
						SaveStrategy:  "errors",
						RetentionDays: 30,
						AutoCleanup:   "off",
					},
				},
			},
		},
	})

	err := service.FinishLLMCall(context.Background(), llm.CallRecordFinish{
		ID:         "call-1",
		Status:     llm.CallRecordStatusCompleted,
		FinishedAt: time.Date(2026, 4, 17, 8, 0, 1, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("finish llm call: %v", err)
	}
	if repo.deleteCount != 1 {
		t.Fatalf("expected one delete for successful record, got %d", repo.deleteCount)
	}
	if _, ok := repo.items["call-1"]; ok {
		t.Fatal("expected successful record to be removed when strategy=errors")
	}
	if repo.updateCount != 0 {
		t.Fatalf("expected no update for deleted successful record, got %d", repo.updateCount)
	}
}

func TestServicePruneExpiredUsesConfiguredRetention(t *testing.T) {
	t.Parallel()

	repo := newStubRepository()
	now := time.Date(2026, 4, 17, 15, 30, 0, 0, time.UTC)
	service := NewService(repo, stubSettingsReader{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{
				Runtime: settingsdto.GatewayRuntimeSettings{
					CallRecords: settingsdto.GatewayCallRecordsSettings{
						SaveStrategy:  "all",
						RetentionDays: 14,
						AutoCleanup:   "hourly",
					},
				},
			},
		},
	})
	service.now = func() time.Time { return now }

	if _, err := service.PruneExpired(context.Background()); err != nil {
		t.Fatalf("prune expired records: %v", err)
	}
	expectedCutoff := now.AddDate(0, 0, -14)
	if !repo.lastDeleteBefore.Equal(expectedCutoff) {
		t.Fatalf("expected cutoff %s, got %s", expectedCutoff, repo.lastDeleteBefore)
	}
	if repo.deleteBeforeCount != 1 {
		t.Fatalf("expected one retention cleanup call, got %d", repo.deleteBeforeCount)
	}
}

type stubSettingsReader struct {
	settings settingsdto.Settings
	err      error
}

func (reader stubSettingsReader) GetSettings(_ context.Context) (settingsdto.Settings, error) {
	if reader.err != nil {
		return settingsdto.Settings{}, reader.err
	}
	return reader.settings, nil
}

type stubRepository struct {
	items             map[string]Record
	insertCount       int
	updateCount       int
	deleteCount       int
	deleteBeforeCount int
	deleteAllCount    int
	lastDeleteBefore  time.Time
}

func newStubRepository() *stubRepository {
	return &stubRepository{items: map[string]Record{}}
}

func (repo *stubRepository) Insert(_ context.Context, record Record) error {
	repo.insertCount++
	repo.items[record.ID] = record
	return nil
}

func (repo *stubRepository) Update(_ context.Context, record Record) error {
	repo.updateCount++
	repo.items[record.ID] = record
	return nil
}

func (repo *stubRepository) Get(_ context.Context, id string) (Record, error) {
	record, ok := repo.items[id]
	if !ok {
		return Record{}, errors.New("not found")
	}
	return record, nil
}

func (repo *stubRepository) List(_ context.Context, _ QueryFilter) ([]Record, error) {
	result := make([]Record, 0, len(repo.items))
	for _, item := range repo.items {
		result = append(result, item)
	}
	return result, nil
}

func (repo *stubRepository) Delete(_ context.Context, id string) error {
	repo.deleteCount++
	delete(repo.items, id)
	return nil
}

func (repo *stubRepository) DeleteStartedBefore(_ context.Context, cutoff time.Time) (int, error) {
	repo.deleteBeforeCount++
	repo.lastDeleteBefore = cutoff
	return 0, nil
}

func (repo *stubRepository) DeleteAll(_ context.Context) (int, error) {
	repo.deleteAllCount++
	count := len(repo.items)
	repo.items = map[string]Record{}
	return count, nil
}
