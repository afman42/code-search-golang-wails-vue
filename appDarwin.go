//go:build darwin

// Package main implements the backend functionality for the code search application.
// It provides functions for searching through code files, validating directories,
// and interacting with the system's file manager.
package main

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// terminalEmulators lists terminal emulators to try on macOS, in order of
// preference. The first one found in PATH is used to wrap terminal editors.
var terminalEmulators = []struct {
	name string
	args []string
}{
	{"Terminal.app", []string{"-e"}},
	{"iTerm.app", []string{"-e"}},
	{"wezterm", []string{"start", "--"}},
	{"alacritty", []string{"-e"}},
}

// wrapTerminalEditor wraps a terminal editor command in a terminal emulator.
// It returns the emulator command and args needed to run the editor inside it.
func wrapTerminalEditor(editor string, args []string) (string, []string) {
	emu, _ := detectTerminalEmulator()
	if emu.name == "" {
		// Fallback: run the editor directly (may not work without TTY)
		return editor, args
	}
	wrappedArgs := append(emu.args, editor)
	wrappedArgs = append(wrappedArgs, args...)
	return emu.name, wrappedArgs
}

// detectTerminalEmulator finds the first available terminal emulator from
// the fallback list. Returns the emulator entry and true if found.
func detectTerminalEmulator() (struct {
	name string
	args []string
}, bool) {
	for _, emu := range terminalEmulators {
		if _, err := exec.LookPath(emu.name); err == nil {
			return emu, true
		}
	}
	return struct {
		name string
		args []string
	}{}, false
}

// ShowInFolder reveals the given file in Finder using `open -R`, which selects
// and highlights the file in its containing folder.
func (a *App) ShowInFolder(filePath string) error {
	a.logDebug("Opening file location in folder", logrus.Fields{
		"filePath": filePath,
	})

	// The shared helper validates traversal and that the parent exists, and
	// returns the cleaned ABSOLUTE parent directory. Re-derive the absolute
	// file path from it — passing the raw (possibly relative) input to
	// `open -R` would reveal the wrong location relative to the app's cwd.
	absDir, err := a.validatePathForShowInFolder(filePath)
	if err != nil {
		return err
	}
	absPath := filepath.Join(absDir, filepath.Base(filePath))

	if err := runCommand("open", []string{"-R", absPath}); err != nil {
		a.logError("Failed to open folder", err, logrus.Fields{
			"filePath": absPath,
		})
		return err
	}

	a.logDebug("Successfully opened folder", logrus.Fields{
		"filePath": absPath,
	})
	return nil
}

// openInEditor is a helper function to open a file in a specific editor.
// If the editor is marked as needing a terminal (terminal=true), it wraps
// the command in a detected terminal emulator.
func (a *App) openInEditor(filePath string, editor string, args []string, terminal bool) error {
	a.logDebug("Opening file in editor", logrus.Fields{
		"filePath": filePath,
		"editor":   editor,
		"args":     args,
	})

	cleanPath, err := a.validatePathForEditor(filePath)
	if err != nil {
		return err
	}
	editorPath, err := a.lookUpEditor(editor)
	if err != nil {
		return err
	}

	if terminal {
		editorPath, args = wrapTerminalEditor(editorPath, appendPath(args, cleanPath))
	} else {
		args = appendPath(args, cleanPath)
	}

	if err := runCommand(editorPath, args); err != nil {
		a.logError("Failed to open file in editor", err, logrus.Fields{
			"editor": editor,
			"args":   args,
		})
		return fmt.Errorf("failed to open file in %s: %w", editor, err)
	}

	a.logDebug("Successfully opened file in editor", logrus.Fields{
		"editor":   editor,
		"filePath": filePath,
	})
	return nil
}

// OpenInDefaultEditor opens a file in the system's default editor via `open`.
func (a *App) OpenInDefaultEditor(filePath string) error {
	a.logDebug("Opening file in default editor", logrus.Fields{
		"filePath": filePath,
	})

	cleanPath, err := a.validatePathForEditor(filePath)
	if err != nil {
		return err
	}

	if err := runCommand("open", []string{cleanPath}); err != nil {
		a.logError("Failed to open file in default editor", err, logrus.Fields{
			"filePath": cleanPath,
		})
		return fmt.Errorf("failed to open file in default editor: %w", err)
	}

	a.logDebug("Successfully opened file in default editor", logrus.Fields{
		"filePath": filePath,
	})
	return nil
}
