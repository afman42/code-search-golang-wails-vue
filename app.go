//go:build linux

// Package main implements the backend functionality for the code search application.
// It provides functions for searching through code files, validating directories,
// and interacting with the system's file manager.
package main

import (
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// terminalEmulators lists terminal emulators to try on Linux, in order of
// preference. The first one found in PATH is used to wrap terminal editors.
var terminalEmulators = []struct {
	name string
	args []string
}{
	{"x-terminal-emulator", []string{"-e"}},
	{"gnome-terminal", []string{"--"}},
	{"konsole", []string{"-e"}},
	{"xfce4-terminal", []string{"-e"}},
	{"alacritty", []string{"-e"}},
	{"kitty", []string{"--"}},
	{"wezterm", []string{"start", "--"}},
	{"terminator", []string{"-e"}},
	{"xterm", []string{"-e"}},
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

// ShowInFolder opens the containing folder of the given file path in the system's file manager.
func (a *App) ShowInFolder(filePath string) error {
	a.logDebug("Opening file location in folder", logrus.Fields{
		"filePath": filePath,
	})

	absDir, err := a.validatePathForShowInFolder(filePath)
	if err != nil {
		return err
	}

	// This file is //go:build linux, so the platform switch was dead code.
	err = runCommand("xdg-open", []string{absDir})
	if err != nil {
		a.logError("Failed to open folder", err, logrus.Fields{
			"directory": absDir,
		})
		return err
	}

	a.logDebug("Successfully opened folder", logrus.Fields{
		"directory": absDir,
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

	err = runCommand(editorPath, args)
	if err != nil {
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

// OpenInDefaultEditor opens a file in the system's default editor
func (a *App) OpenInDefaultEditor(filePath string) error {
	a.logDebug("Opening file in default editor", logrus.Fields{
		"filePath": filePath,
	})

	// Validate the path (traversal + existence) exactly like openInEditor —
	// previously this binding passed the raw frontend-supplied path straight
	// to xdg-open, making it the one unvalidated exec path.
	cleanPath, err := a.validatePathForEditor(filePath)
	if err != nil {
		return err
	}

	if err := runCommand("xdg-open", []string{cleanPath}); err != nil {
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
