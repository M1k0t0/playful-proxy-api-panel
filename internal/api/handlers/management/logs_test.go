package management

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectLogStorageGroupsKnownLogFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "main.log"), "main")
	writeTestFile(t, filepath.Join(dir, "main.log.1"), "old")
	writeTestFile(t, filepath.Join(dir, "v1-chat-completions-2026-05-03T120000-abc.log"), "request")
	writeTestFile(t, filepath.Join(dir, "error-v1-chat-completions-2026-05-03T120001-def.log"), "error")
	writeTestFile(t, filepath.Join(dir, "request-body-123.tmp"), "tmp")
	writeTestFile(t, filepath.Join(dir, "notes.txt"), "ignore")

	summary, err := collectLogStorage(dir)
	if err != nil {
		t.Fatalf("collectLogStorage returned error: %v", err)
	}

	if summary.TotalFiles != 5 {
		t.Fatalf("TotalFiles = %d, want 5", summary.TotalFiles)
	}
	if summary.Application.Files != 2 {
		t.Fatalf("Application.Files = %d, want 2", summary.Application.Files)
	}
	if summary.Request.Files != 1 {
		t.Fatalf("Request.Files = %d, want 1", summary.Request.Files)
	}
	if summary.ErrorRequest.Files != 1 {
		t.Fatalf("ErrorRequest.Files = %d, want 1", summary.ErrorRequest.Files)
	}
	if summary.Temporary.Files != 1 {
		t.Fatalf("Temporary.Files = %d, want 1", summary.Temporary.Files)
	}
}

func TestClearLogDataClearsTargetWithoutTouchingOtherLogs(t *testing.T) {
	dir := t.TempDir()
	mainLog := filepath.Join(dir, "main.log")
	requestLog := filepath.Join(dir, "v1-chat-completions-2026-05-03T120000-abc.log")
	errorLog := filepath.Join(dir, "error-v1-chat-completions-2026-05-03T120001-def.log")
	writeTestFile(t, mainLog, "main")
	writeTestFile(t, requestLog, "request")
	writeTestFile(t, errorLog, "error")

	result, err := clearLogData(dir, logDataTargetRequest)
	if err != nil {
		t.Fatalf("clearLogData returned error: %v", err)
	}
	if result.Removed != 1 || result.Truncated != 0 {
		t.Fatalf("clear result = removed %d truncated %d, want removed 1 truncated 0", result.Removed, result.Truncated)
	}
	if _, errStat := os.Stat(requestLog); !os.IsNotExist(errStat) {
		t.Fatalf("request log still exists or stat failed with non-missing error: %v", errStat)
	}
	if _, errStat := os.Stat(mainLog); errStat != nil {
		t.Fatalf("main log should remain: %v", errStat)
	}
	if _, errStat := os.Stat(errorLog); errStat != nil {
		t.Fatalf("error log should remain: %v", errStat)
	}
}

func TestClearLogDataAllTruncatesActiveMainLog(t *testing.T) {
	dir := t.TempDir()
	mainLog := filepath.Join(dir, "main.log")
	rotatedLog := filepath.Join(dir, "main.log.1")
	writeTestFile(t, mainLog, "main")
	writeTestFile(t, rotatedLog, "old")
	writeTestFile(t, filepath.Join(dir, "v1-responses-2026-05-03T120000-abc.log"), "request")

	result, err := clearLogData(dir, logDataTargetAll)
	if err != nil {
		t.Fatalf("clearLogData returned error: %v", err)
	}
	if result.Truncated != 1 {
		t.Fatalf("Truncated = %d, want 1", result.Truncated)
	}
	info, errStat := os.Stat(mainLog)
	if errStat != nil {
		t.Fatalf("main log should remain: %v", errStat)
	}
	if info.Size() != 0 {
		t.Fatalf("main log size = %d, want 0", info.Size())
	}
	if _, errStat = os.Stat(rotatedLog); !os.IsNotExist(errStat) {
		t.Fatalf("rotated log still exists or stat failed with non-missing error: %v", errStat)
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
