package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dreamcreator/internal/domain/library"
)

type subtitleDownloadFileRepo struct {
	saved []library.LibraryFile
}

func (repo *subtitleDownloadFileRepo) List(_ context.Context) ([]library.LibraryFile, error) {
	return nil, nil
}

func (repo *subtitleDownloadFileRepo) ListByLibraryID(_ context.Context, _ string) ([]library.LibraryFile, error) {
	return nil, nil
}

func (repo *subtitleDownloadFileRepo) Get(_ context.Context, _ string) (library.LibraryFile, error) {
	return library.LibraryFile{}, library.ErrFileNotFound
}

func (repo *subtitleDownloadFileRepo) Save(_ context.Context, item library.LibraryFile) error {
	repo.saved = append(repo.saved, item)
	return nil
}

func (repo *subtitleDownloadFileRepo) Delete(_ context.Context, _ string) error {
	return nil
}

type subtitleDownloadDocumentRepo struct {
	saved []library.SubtitleDocument
}

func (repo *subtitleDownloadDocumentRepo) Get(_ context.Context, _ string) (library.SubtitleDocument, error) {
	return library.SubtitleDocument{}, library.ErrSubtitleDocumentNotFound
}

func (repo *subtitleDownloadDocumentRepo) GetByFileID(_ context.Context, _ string) (library.SubtitleDocument, error) {
	return library.SubtitleDocument{}, library.ErrSubtitleDocumentNotFound
}

func (repo *subtitleDownloadDocumentRepo) Save(_ context.Context, document library.SubtitleDocument) error {
	repo.saved = append(repo.saved, document)
	return nil
}

func (repo *subtitleDownloadDocumentRepo) DeleteByFileID(_ context.Context, _ string) error {
	return nil
}

func TestCreateDownloadedSubtitleFileStoresDBDocumentAndDeletesSource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempDir := t.TempDir()
	subtitlePath := filepath.Join(tempDir, "episode.en.srt")
	content := "1\n00:00:00,000 --> 00:00:01,000\nHello world.\n"
	if err := os.WriteFile(subtitlePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write subtitle: %v", err)
	}

	files := &subtitleDownloadFileRepo{}
	subtitles := &subtitleDownloadDocumentRepo{}
	service := &LibraryService{
		files:     files,
		subtitles: subtitles,
		nowFunc: func() time.Time {
			return time.Date(2026, 3, 24, 8, 0, 0, 0, time.UTC)
		},
	}

	fileItem, err := service.createDownloadedSubtitleFile(
		ctx,
		library.LibraryOperation{ID: "op-1", LibraryID: "lib-1"},
		subtitlePath,
		"Episode 01",
		time.Date(2026, 3, 24, 7, 30, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("create downloaded subtitle: %v", err)
	}

	if fileItem.Storage.Mode != "db_document" {
		t.Fatalf("expected db_document storage mode, got %q", fileItem.Storage.Mode)
	}
	if fileItem.Storage.LocalPath != "" {
		t.Fatalf("expected downloaded subtitle local path to be cleared, got %q", fileItem.Storage.LocalPath)
	}
	if fileItem.Storage.DocumentID == "" {
		t.Fatalf("expected downloaded subtitle document id to be populated")
	}
	if len(files.saved) != 1 {
		t.Fatalf("expected one file save, got %d", len(files.saved))
	}
	if len(subtitles.saved) != 1 {
		t.Fatalf("expected one subtitle document save, got %d", len(subtitles.saved))
	}
	if subtitles.saved[0].OriginalContent != content {
		t.Fatalf("expected subtitle document content to round-trip, got %q", subtitles.saved[0].OriginalContent)
	}
	if _, statErr := os.Stat(subtitlePath); !os.IsNotExist(statErr) {
		t.Fatalf("expected source subtitle file to be deleted, stat err=%v", statErr)
	}
}
