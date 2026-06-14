package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

// quietApp returns an App whose logger discards output and runs at Info level,
// so benchmarks measure search work rather than logging I/O. Running above
// DebugLevel also exercises the guarded debug-logging fast path.
func quietApp() *App {
	app := NewApp()
	app.logger.SetOutput(io.Discard)
	app.logger.SetLevel(logrus.InfoLevel)
	return app
}

// setupBenchTree creates a directory tree of small text files for benchmarking
// the search pipeline. Roughly 1 in 20 files contains the search term.
func setupBenchTree(b *testing.B, numFiles int) string {
	b.Helper()
	tempDir := b.TempDir()

	for i := 0; i < numFiles; i++ {
		// Spread files across subdirectories so the walk has some depth.
		dir := filepath.Join(tempDir, fmt.Sprintf("dir_%d", i%50))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatalf("Failed to create dir: %v", err)
		}
		content := fmt.Sprintf("package main\n// file %d\nfunc helper%d() {}\n", i, i)
		if i%20 == 0 {
			content += "needle marker line\n"
		}
		filename := filepath.Join(dir, fmt.Sprintf("file_%d.go", i))
		if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
			b.Fatalf("Failed to create test file %d: %v", i, err)
		}
	}
	return tempDir
}

// BenchmarkSearchWithProgress measures the end-to-end search over a tree of
// many small files. This is the path where per-file syscall overhead (stat,
// abs, open) dominates, so it reflects the collection/worker metadata reuse.
func BenchmarkSearchWithProgress(b *testing.B) {
	app := quietApp()

	tempDir := setupBenchTree(b, 2000)
	req := SearchRequest{
		Directory:     tempDir,
		Query:         "needle",
		SearchSubdirs: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := app.SearchWithProgress(req)
		if err != nil {
			b.Fatalf("SearchWithProgress failed: %v", err)
		}
		if len(results) == 0 {
			b.Fatal("expected matches, got none")
		}
	}
}

// BenchmarkCollectFilesToProcess isolates the directory-walk + filtering phase,
// where the absolute-path and size metadata are gathered once per file.
func BenchmarkCollectFilesToProcess(b *testing.B) {
	app := quietApp()

	tempDir := setupBenchTree(b, 2000)
	req := SearchRequest{
		Directory:     tempDir,
		Query:         "needle",
		SearchSubdirs: true,
	}
	validated, err := app.validateAndSetDefaults(req)
	if err != nil {
		b.Fatalf("validateAndSetDefaults failed: %v", err)
	}
	pattern, err := app.compileSearchPattern(validated)
	if err != nil {
		b.Fatalf("compileSearchPattern failed: %v", err)
	}
	absDir, _ := filepath.Abs(validated.Directory)
	baseDir := filepath.Clean(absDir) + string(filepath.Separator)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := app.collectFilesToProcess(validated, pattern, baseDir)
		if err != nil {
			b.Fatalf("collectFilesToProcess failed: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("expected collected files, got none")
		}
	}
}
