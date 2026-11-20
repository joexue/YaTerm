//go:build !windows

package ui

import "fyne.io/fyne/v2"

// KeepSizeWhenMinimized is for Windows only, see minimize_windows.go
func KeepSizeWhenMinimized(fyne.Window) {}
