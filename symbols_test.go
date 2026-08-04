package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestFile creates a temporary file with the given content and extension.
// Returns the full path. Cleanup is handled by the caller via t.Cleanup.
func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestGetAllSymbols_Go(t *testing.T) {
	dir := t.TempDir()
	content := `package main

import "fmt"

// PublicFunc is exported.
func PublicFunc(x int) int {
	return x * 2
}

type Config struct {
	Name string
}

type Reader interface {
	Read() error
}

const MaxRetries = 3

var GlobalState = "init"

func lowerFunc() {}
`
	writeTestFile(t, dir, "sample.go", content)

	symbols := GetAllSymbols(dir, 100)
	if len(symbols) == 0 {
		t.Fatal("expected symbols from Go file, got none")
	}

	// Collect names for assertions
	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}

	for _, want := range []string{"PublicFunc", "Config", "Reader", "MaxRetries", "GlobalState"} {
		if !names[want] {
			t.Errorf("expected symbol %q not found", want)
		}
	}

	// lowerFunc should NOT be present (lowercase, filtered out)
	if names["lowerFunc"] {
		t.Error("lowercase func lowerFunc should not be extracted")
	}

	// Check a function entry has correct line + signature
	for _, s := range symbols {
		if s.Name == "PublicFunc" {
			if s.Type != "function" {
				t.Errorf("PublicFunc type = %q, want function", s.Type)
			}
			if s.File != filepath.Join(dir, "sample.go") {
				t.Errorf("PublicFunc file = %q, want %q", s.File, filepath.Join(dir, "sample.go"))
			}
			if s.Line <= 0 {
				t.Error("PublicFunc line should be > 0")
			}
		}
	}
}

func TestGetAllSymbols_TypeScript(t *testing.T) {
	dir := t.TempDir()
	content := `import { ref } from 'vue';

export function fetchData(url: string) {
	return fetch(url);
}

export const API_URL = "https://api.example.com";

export class UserService {
	private client: HttpClient;
}

export interface User {
	id: number;
	name: string;
}

export type Status = "active" | "inactive";
`
	writeTestFile(t, dir, "service.ts", content)

	symbols := GetAllSymbols(dir, 100)
	if len(symbols) == 0 {
		t.Fatal("expected symbols from TS file, got none")
	}

	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}

	for _, want := range []string{"fetchData", "API_URL", "UserService", "User", "Status"} {
		if !names[want] {
			t.Errorf("expected TS symbol %q not found", want)
		}
	}
}

func TestGetAllSymbols_Vue(t *testing.T) {
	dir := t.TempDir()
	content := `<template>
  <MyComponent />
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

function handleClick() {
	console.log('clicked');
}

const count = ref(0);
const doubled = computed(() => count.value * 2);

const submitForm = (e) => {};
</script>
`
	writeTestFile(t, dir, "Comp.vue", content)

	symbols := GetAllSymbols(dir, 100)
	if len(symbols) == 0 {
		t.Fatal("expected symbols from Vue file, got none")
	}

	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}

	// Vue extraction focuses on script-section symbols; component name from template
	if !names["handleClick"] {
		t.Error("expected handleClick symbol")
	}
	if !names["MyComponent"] {
		t.Error("expected MyComponent symbol from template")
	}
}

func TestGetAllSymbols_MaxResults(t *testing.T) {
	dir := t.TempDir()
	// Create a file with many exported funcs
	var b strings.Builder
	b.WriteString("package main\n\n")
	for i := 0; i < 10; i++ {
		b.WriteString("func Func" + itoa(i) + "() {}\n")
	}
	writeTestFile(t, dir, "many.go", b.String())

	symbols := GetAllSymbols(dir, 3)
	if len(symbols) > 3 {
		t.Errorf("expected at most 3 symbols, got %d", len(symbols))
	}
}

func TestGetAllSymbols_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	symbols := GetAllSymbols(dir, 100)
	if symbols == nil {
		t.Error("expected non-nil slice for empty dir")
	}
	if len(symbols) != 0 {
		t.Errorf("expected 0 symbols for empty dir, got %d", len(symbols))
	}
}

func TestGetAllSymbols_SkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	// Real source symbol
	writeTestFile(t, dir, "main.go", "package main\nfunc RealFunc() {}\n")
	// node_modules symbol that should be skipped
	writeTestFile(t, dir, "node_modules/dep/index.js", "export function JunkFunc() {}\n")

	symbols := GetAllSymbols(dir, 100)
	for _, s := range symbols {
		if strings.Contains(s.File, "node_modules") {
			t.Errorf("should not extract from node_modules: %s", s.File)
		}
		if s.Name == "JunkFunc" {
			t.Error("JunkFunc from node_modules should be skipped")
		}
	}
}

func TestSearchSymbols_Matching(t *testing.T) {
	dir := t.TempDir()
	content := `package main

func SearchSymbols() {}
func GetAllSymbols() {}
func normalizeName() {}
type Config struct{}
`
	writeTestFile(t, dir, "app.go", content)

	results := SearchSymbols("Symbol", dir, 100)
	if len(results) == 0 {
		t.Fatal("expected matching symbols for 'Symbol'")
	}

	// Should match both SearchSymbols and GetAllSymbols (name contains "Symbol")
	names := make(map[string]bool)
	for _, r := range results {
		names[r.Name] = true
	}
	if !names["SearchSymbols"] {
		t.Error("expected SearchSymbols to match")
	}
	if !names["GetAllSymbols"] {
		t.Error("expected GetAllSymbols to match")
	}
	// normalizeName starts lowercase, filtered out from extraction; should not match
	if names["normalizeName"] {
		t.Error("normalizeName should not be extracted")
	}
}

func TestSearchSymbols_NoMatch(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", "package main\nfunc Foo() {}\n")

	results := SearchSymbols("NonExistent", dir, 100)
	if results == nil {
		t.Error("expected non-nil slice for no match")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for non-matching query, got %d", len(results))
	}
}

func TestSearchSymbols_EmptyNameReturnsAll(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", "package main\nfunc Foo() {}\nfunc Bar() {}\n")

	results := SearchSymbols("", dir, 100)
	if len(results) < 2 {
		t.Errorf("expected at least 2 symbols with empty query, got %d", len(results))
	}
}

func TestSearchSymbols_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", "package main\nfunc CamelCase() {}\n")

	results := SearchSymbols("camel", dir, 100)
	if len(results) == 0 {
		t.Fatal("expected case-insensitive match for 'camel'")
	}
}

func TestExtractSymbolsFromFile_NonExistentFile(t *testing.T) {
	symbols := extractSymbolsFromFile("/nonexistent/path/file.go", ".go")
	if symbols != nil {
		t.Errorf("expected nil for non-existent file, got %v", symbols)
	}
}

func TestGetSymbolType(t *testing.T) {
	tests := []struct {
		keyword   string
		signature string
		want      string
	}{
		{"func", "func Foo() {}", "function"},
		{"method", "func (r R) Foo() {}", "function"},
		{"class", "export class Foo {}", "class"},
		{"struct", "type Foo struct{}", "class"},
		{"interface", "type Foo interface{}", "class"},
		{"const", "const X = 1", "const"},
		{"var", "var X = 1", "variable"},
		{"", "let x = 1", "variable"},
		{"", "export const x = 1", "variable"},
		{"", "random declaration", "symbol"},
	}

	for _, tc := range tests {
		got := GetSymbolType(tc.keyword, tc.signature)
		if got != tc.want {
			t.Errorf("GetSymbolType(%q, %q) = %q, want %q", tc.keyword, tc.signature, got, tc.want)
		}
	}
}

func TestNormalizeSymbolName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Foo", "Foo"},
		{" Foo ", "Foo"},
		{"Foo[", "Foo"},
		{"Foo(", "Foo"},
		{"  Bar  ", "Bar"},
		{"", ""},
	}

	for _, tc := range tests {
		got := normalizeSymbolName(tc.input)
		if got != tc.want {
			t.Errorf("normalizeSymbolName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestGetPatternsForExtension(t *testing.T) {
	// Each supported extension should return patterns
	for _, ext := range []string{".go", ".ts", ".tsx", ".js", ".vue"} {
		patterns := getPatternsForExtension(ext)
		if len(patterns) == 0 {
			t.Errorf("expected patterns for extension %q", ext)
		}
		for _, p := range patterns {
			if p.regex == nil {
				t.Errorf("nil regex in pattern for %q", ext)
			}
		}
	}

	// Unsupported extension should return nil
	patterns := getPatternsForExtension(".py")
	if patterns != nil {
		t.Errorf("expected nil patterns for unsupported extension .py, got %v", patterns)
	}
}

// itoa converts an int to string without importing strconv (keeps test deps minimal).
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
