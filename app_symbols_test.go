package main

import (
	"testing"
)

func TestAppGetAllSymbols_EmptyDir(t *testing.T) {
	app := &App{}
	symbols := app.GetAllSymbols("", 100)
	if symbols == nil {
		t.Error("GetAllSymbols returned nil, want non-nil empty slice")
	}
	if len(symbols) != 0 {
		t.Errorf("GetAllSymbols returned %d symbols, want 0", len(symbols))
	}
}

func TestAppGetAllSymbols_WithFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", "package main\nfunc Hello() string { return \"world\" }\n")
	writeTestFile(t, dir, "util.go", "package main\nvar Version = \"1.0\"\n")

	app := &App{}
	symbols := app.GetAllSymbols(dir, 100)
	if symbols == nil {
		t.Fatal("GetAllSymbols returned nil")
	}
	if len(symbols) == 0 {
		t.Fatal("expected symbols, got 0")
	}
}

func TestAppGetAllSymbols_NilCtxDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", "package main\nfunc Foo() {}\n")

	app := &App{}
	symbols := app.GetAllSymbols(dir, 100)
	if symbols == nil {
		t.Error("GetAllSymbols returned nil with nil ctx")
	}
}

func TestAppSearchSymbols_EmptyDir(t *testing.T) {
	app := &App{}
	symbols := app.SearchSymbols("test", "", 100)
	if symbols == nil {
		t.Error("SearchSymbols returned nil, want non-nil empty slice")
	}
	if len(symbols) != 0 {
		t.Errorf("SearchSymbols returned %d symbols, want 0", len(symbols))
	}
}

func TestAppSearchSymbols_Matching(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", "package main\nfunc Hello() string { return \"world\" }\nfunc Goodbye() string { return \"later\" }\n")

	app := &App{}
	symbols := app.SearchSymbols("Hello", dir, 100)
	if symbols == nil {
		t.Fatal("SearchSymbols returned nil")
	}
	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
	if symbols[0].Name != "Hello" {
		t.Errorf("expected symbol name 'Hello', got %q", symbols[0].Name)
	}
}

func TestAppSearchSymbols_NoMatch(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", "package main\nfunc Foo() {}\n")

	app := &App{}
	symbols := app.SearchSymbols("NonExistent", dir, 100)
	if symbols == nil {
		t.Error("SearchSymbols returned nil, want non-nil empty slice")
	}
	if len(symbols) != 0 {
		t.Errorf("expected 0 symbols, got %d", len(symbols))
	}
}

func TestAppSearchSymbols_EmptyName(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", "package main\nfunc Foo() {}\nfunc Bar() {}\n")

	app := &App{}
	symbols := app.SearchSymbols("", dir, 100)
	if symbols == nil {
		t.Fatal("SearchSymbols returned nil")
	}
	if len(symbols) < 2 {
		t.Errorf("expected at least 2 symbols with empty name, got %d", len(symbols))
	}
}

func TestAppGetAllSymbols_NilResultGuard(t *testing.T) {
	app := &App{}
	symbols := app.GetAllSymbols("/nonexistent/path/that/does/not/exist/x/y/z", 100)
	if symbols == nil {
		t.Error("GetAllSymbols returned nil, want non-nil empty slice")
	}
}

func TestAppGetAllSymbols_MaxResults(t *testing.T) {
	dir := t.TempDir()
	code := "package main\n"
	for i := 0; i < 10; i++ {
		code += "func Func" + string(rune('A'+i)) + "() {}\n"
	}
	writeTestFile(t, dir, "main.go", code)

	app := &App{}
	symbols := app.GetAllSymbols(dir, 3)
	if len(symbols) > 3 {
		t.Errorf("expected at most 3 symbols, got %d", len(symbols))
	}
}