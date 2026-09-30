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

// terminalEmulator describes one terminal emulator fallback: the command
// name to LookPath and the args that launch a command inside it.
type terminalEmulator struct {
	name string
	args []string
}

// terminalEmulators lists terminal emulators to try on Linux, in order of
// preference. The first one found in PATH is used to wrap terminal editors.
var terminalEmulators = []terminalEmulator{
	{name: "x-terminal-emulator", args: []string{"-e"}},
	{name: "gnome-terminal", args: []string{"--"}},
	{name: "konsole", args: []string{"-e"}},
	{name: "xfce4-terminal", args: []string{"-e"}},
	{name: "alacritty", args: []string{"-e"}},
	{name: "kitty", args: []string{"--"}},
	{name: "wezterm", args: []string{"start", "--"}},
	{name: "terminator", args: []string{"-e"}},
	{name: "xterm", args: []string{"-e"}},
}

// wrapTerminalEditor wraps a terminal editor command in a terminal emulator.
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
func detectTerminalEmulator() (terminalEmulator, bool) {
	for _, emu := range terminalEmulators {
		if _, err := exec.LookPath(emu.name); err == nil {
			return emu, true
		}
	}
	return terminalEmulator{}, false
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
		a.warnErr("Failed to open folder", err, logrus.Fields{"directory": absDir})
		return fmt.Errorf("failed to open folder %q: %w", absDir, err)
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

	args = appendPath(args, cleanPath)
	if terminal {
		editorPath, args = wrapTerminalEditor(editorPath, args)
	}

	err = runCommand(editorPath, args)
	if err != nil {
		a.warnErr("Failed to open file in editor", err, logrus.Fields{"editor": editor})
		return fmt.Errorf("failed to open file in %q: %w", editor, err)
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
		a.warnErr("Failed to open file in default editor", err, nil)
		return fmt.Errorf("failed to open file in default editor: %w", err)
	}

	a.logDebug("Successfully opened file in default editor", logrus.Fields{
		"filePath": filePath,
	})
	return nil
}
