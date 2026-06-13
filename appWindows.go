//go:build windows

// Package main implements the backend functionality for the code search application.
// It provides functions for searching through code files, validating directories,
// and interacting with the system's file manager.
package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/sirupsen/logrus"
)

// ShowInFolder opens the containing folder of the given file path in the system's file manager.
func (a *App) ShowInFolder(filePath string) error {
	a.logDebug("Opening file location in folder", logrus.Fields{
		"filePath": filePath,
	})

	absDir, err := a.validatePathForShowInFolder(filePath)
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "start", absDir)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
		err = cmd.Start()
	case "darwin":
		a.logError("macOS folder opening not implemented", nil, logrus.Fields{})
		return fmt.Errorf("macOS folder opening not implemented")
	default:
		a.logError("Unsupported platform for ShowInFolder", nil, logrus.Fields{
			"platform": runtime.GOOS,
		})
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

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
func (a *App) openInEditor(filePath string, editor string, args []string) error {
	a.logDebug("Opening file in editor", logrus.Fields{
		"filePath": filePath,
		"editor":   editor,
		"args":     args,
	})

	cleanPath, err := a.validatePathForEditor(filePath)
	if err != nil {
		return err
	}
	if err := a.lookUpEditor(editor); err != nil {
		return err
	}

	cmd := exec.Command(editor, append(args, cleanPath)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	if err := cmd.Start(); err != nil {
		a.logError("Failed to open file in editor", err, logrus.Fields{
			"editor": editor,
			"args":   args,
		})
		return fmt.Errorf("failed to open file in %s: %v", editor, err)
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

	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "start", "", filePath)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
		if err := cmd.Start(); err != nil {
			a.logError("Failed to open file in default editor", err, logrus.Fields{
				"filePath": filePath,
			})
			return fmt.Errorf("failed to open file in default editor: %v", err)
		}
	default:
		a.logError("Unsupported platform for OpenInDefaultEditor", nil, logrus.Fields{
			"platform": runtime.GOOS,
		})
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	a.logDebug("Successfully opened file in default editor", logrus.Fields{
		"filePath": filePath,
	})
	return nil
}
