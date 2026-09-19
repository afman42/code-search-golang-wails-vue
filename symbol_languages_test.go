package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// mockDirEntry implements fs.DirEntry for shouldSkipDir tests.
type mockDirEntry struct{ name string }

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return true }
func (m mockDirEntry) Type() fs.FileMode          { return fs.ModeDir }
func (m mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestGetAllSymbols_Python(t *testing.T) {
	dir := t.TempDir()
	content := `import os

class MyClass(Base):
    pass

def my_function(x, y):
    return x + y

async def async_fetch(url):
    pass

def _private_helper():
    pass

def __init__(self):
    pass

MAX_RETRIES = 5
MAX_RETRIES_WITH_TYPE: int = 10
helper_var = 1
`
	writeTestFile(t, dir, "module.py", content)

	symbols := GetAllSymbols(dir, 100)
	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}
	for _, want := range []string{"MyClass", "my_function", "async_fetch", "__init__", "MAX_RETRIES"} {
		if !names[want] {
			t.Errorf("expected Python symbol %q not found, got %v", want, names)
		}
	}
	if names["_private_helper"] {
		t.Error("_private_helper should be skipped (single underscore)")
	}
	if names["helper_var"] {
		t.Error("lowercase var should not be extracted")
	}
}

func TestGetAllSymbols_Rust(t *testing.T) {
	dir := t.TempDir()
	content := `pub struct MyStruct<T> {
    field: i32,
}

pub enum MyEnum {
    A,
    B,
}

pub trait MyTrait {
    fn do_something(&self);
}

pub fn my_function(x: i32) -> i32 { x }

impl MyStruct {
    fn helper() {}
}

pub const MAX_COUNT: usize = 10;
type MyAlias = MyStruct;
`
	writeTestFile(t, dir, "lib.rs", content)

	symbols := GetAllSymbols(dir, 100)
	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}
	for _, want := range []string{"MyStruct", "MyEnum", "MyTrait", "my_function", "MAX_COUNT", "MyAlias"} {
		if !names[want] {
			t.Errorf("expected Rust symbol %q not found", want)
		}
	}
}

func TestGetAllSymbols_Java(t *testing.T) {
	dir := t.TempDir()
	content := `package com.example;

public class MyClass {
    private int field;

    public MyClass() {}

    public void myMethod(String arg) {}

    public static final int CONSTANT = 1;
}

public interface MyInterface {
    void doWork();
}

public enum MyEnum { A, B }

public record MyRecord(String name, int value) {}
`
	writeTestFile(t, dir, "MyClass.java", content)

	symbols := GetAllSymbols(dir, 100)
	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}
	for _, want := range []string{"MyClass", "MyInterface", "MyEnum", "MyRecord", "myMethod"} {
		if !names[want] {
			t.Errorf("expected Java symbol %q not found, got %v", want, names)
		}
	}
}

func TestGetAllSymbols_CSharp(t *testing.T) {
	dir := t.TempDir()
	content := `using System;

public class MyClass {
    public MyClass() {}
    public void MyMethod(int x) {}
    public string MyProperty { get; set; }
}

public interface IMyInterface {
    void DoWork();
}

public struct MyStruct { public int X; }

public enum MyEnum { A, B }

public record MyRecord(string Name);
`
	writeTestFile(t, dir, "MyClass.cs", content)

	symbols := GetAllSymbols(dir, 100)
	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}
	for _, want := range []string{"MyClass", "IMyInterface", "MyStruct", "MyEnum", "MyRecord", "MyMethod", "MyProperty"} {
		if !names[want] {
			t.Errorf("expected C# symbol %q not found, names=%v", want, names)
		}
	}
}

func TestGetAllSymbols_Ruby(t *testing.T) {
	dir := t.TempDir()
	content := "class MyClass < Base\nend\n\ndef my_method(arg)\nend\n\ndef self.class_method\nend\n\ndef my_predicate?\nend\n\ndef save!\nend\n\nmodule MyModule\nend\n\nattr_accessor :my_attr\nMY_CONST = 1\n"
	writeTestFile(t, dir, "app.rb", content)

	symbols := GetAllSymbols(dir, 100)
	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}
	for _, want := range []string{"MyClass", "my_method", "class_method", "my_predicate?", "save!", "MyModule", "my_attr", "MY_CONST"} {
		if !names[want] {
			t.Errorf("expected Ruby symbol %q not found, got %v", want, names)
		}
	}
}

func TestPython_UnderscoreHandling(t *testing.T) {
	dir := t.TempDir()
	content := "def _helper():\n    pass\n\ndef __dunder__(self):\n    pass\n\ndef __private_mangled():\n    pass\n"
	writeTestFile(t, dir, "underscore.py", content)
	symbols := GetAllSymbols(dir, 100)
	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}
	if names["_helper"] {
		t.Error("_helper should be skipped")
	}
	if !names["__dunder__"] {
		t.Error("__dunder__ should be kept (dunder)")
	}
	// __private_mangled starts with __ but not ends with __, per spec it is dunder only if both
	// Here it starts __ and no trailing __, so single prefix counts as private -> should be skipped
	// Our implementation checks both prefix and suffix for dunder, so this should be skipped
	if names["__private_mangled"] {
		// This is actually considered dunder? Check: HasPrefix __ && HasSuffix __ => false, so skipped
		// So we expect it to be skipped
		t.Error("__private_mangled should be skipped (not dunder)")
	}
	// Ensure file path preserved
	for _, s := range symbols {
		if s.Name == "__dunder__" && !strings.HasSuffix(s.File, filepath.Join(dir, "underscore.py")) {
			t.Errorf("file path mismatch for dunder: %q", s.File)
		}
	}
}

func TestShouldSkipDirForSymbolScan(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"node_modules", true},
		{".git", true},
		{"vendor", true},
		{"build", true},
		{"dist", true},
		{"bin", true},
		{"__pycache__", true},
		{".venv", true},
		{"venv", true},
		{".mypy_cache", true},
		{".pytest_cache", true},
		{"target", true},
		{".gradle", true},
		{"obj", true},
		// case-insensitivity
		{"Node_Modules", true},
		{"TARGET", true},
		// negatives
		{"src", false},
		{"my_build", false},
		{"dist2", false},
		{"", false},
	}
	for _, tc := range cases {
		got := shouldSkipDirForSymbolScan(mockDirEntry{name: tc.name})
		if got != tc.want {
			t.Errorf("shouldSkipDirForSymbolScan(%q)=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestShouldSkipDirForSymbolScan_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", "package main\nfunc RealFunc() {}\n")
	writeTestFile(t, dir, "__pycache__/cached.py", "def CachedFunc():\n    pass\n")
	writeTestFile(t, dir, "target/debug.rs", "fn CachedRust() {}\n")
	symbols := GetAllSymbols(dir, 100)
	for _, s := range symbols {
		if s.Name == "CachedFunc" || s.Name == "CachedRust" {
			t.Errorf("should skip build output dir, got %v from %v", s.Name, s.File)
		}
	}
	if len(symbols) == 0 {
		t.Fatal("expected at least RealFunc")
	}
}
