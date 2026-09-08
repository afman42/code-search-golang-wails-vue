//go:build windows

// Package main implements the backend functionality for the code search application.
// It provides functions for searching through code files, validating directories,
// and interacting with the system's file manager.
package main

import (
	"fmt"
	"os/exec"
	"syscall"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

// terminalEmulators lists terminal emulators to try on Windows, in order of
// preference. The first one found in PATH is used to wrap terminal editors.
var terminalEmulators = []struct {
	name string
	args []string
}{
	{"wt", []string{"-d", ".", "cmd", "/c"}},          // Windows Terminal
	{"alacritty", []string{"-e", "cmd", "/c"}},        // Alacritty (needs shell)
	{"wezterm", []string{"start", "--", "cmd", "/c"}}, // WezTerm
	{"cmd", []string{"/c", "start", `""`}},            // cmd fallback (empty title)
}

// wrapTerminalEditor wraps a terminal editor command in a terminal emulator.
// It returns the emulator command and args needed to run the editor inside it.
func wrapTerminalEditor(editor string, args []string) (string, []string) {
	emu, _ := detectTerminalEmulator()
	if emu.name == "" {
		// Fallback: run the editor directly (may not work without TTY)
		return editor, args
	}

	var cmdArgs []string
	switch emu.name {
	case "wt":
		// Windows Terminal: wt -d . cmd /c editor args...
		cmdArgs = append(emu.args, editor)
		cmdArgs = append(cmdArgs, args...)
	case "alacritty":
		// Alacritty: alacritty -e cmd /c editor args...
		cmdArgs = append(emu.args, editor)
		cmdArgs = append(cmdArgs, args...)
	case "wezterm":
		// WezTerm: wezterm start -- cmd /c editor args...
		cmdArgs = append(emu.args, editor)
		cmdArgs = append(cmdArgs, args...)
	case "cmd":
		// cmd /c start "" editor args...
		cmdArgs = []string{`/c`, `start`, `""`}
		cmdArgs = append(cmdArgs, editor)
		cmdArgs = append(cmdArgs, args...)
	default:
		wrappedArgs := append(emu.args, editor)
		wrappedArgs = append(wrappedArgs, args...)
		cmdArgs = wrappedArgs
	}
	return emu.name, cmdArgs
}

// detectTerminalEmulator finds the first available terminal emulator from
// the fallback list. Returns the emulator entry and true if found.
func detectTerminalEmulator() (struct {
	name string
	args []string
}, bool) {
	for _, emu := range terminalEmulators {
		if emu.name == "cmd" {
			// cmd is always available on Windows
			return emu, true
		}
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

	// This file is //go:build windows, so the platform switch was dead code.
	// Use explorer to reveal the folder. Passing the directory as its own
	// argument is space-safe (unlike `cmd /c start <dir>`, where a path with
	// spaces can be misread as the window-title argument). exec.Command
	// quotes each arg, so no manual escaping is needed.
	cmd := exec.Command("explorer", absDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	err = startAndReap(cmd)

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

	cmd := exec.Command(editorPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	if err := startAndReap(cmd); err != nil {
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

	cleanPath, err := a.validatePathForEditor(filePath)
	if err != nil {
		return err
	}

	pathPtr := windows.StringToUTF16Ptr(cleanPath)

	if err := windows.ShellExecute(0, windows.StringToUTF16Ptr("open"), pathPtr, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		a.logError("Failed to open file in default editor", err, logrus.Fields{
			"filePath": cleanPath,
		})
		return fmt.Errorf("failed to open file in default editor: %w", err)
	}

	a.logDebug("Successfully opened file in default editor", logrus.Fields{
		"filePath": cleanPath,
	})
	return nil
}
