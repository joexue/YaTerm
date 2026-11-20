package ui

import (
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

/*
 * ================================
 *  A window minimized on a monitor with another scale than the primary one
 *  (e.g. a laptop at 200% and an external monitor at 100%) came back very
 *  small. Windows moves a minimized window to (-32000, -32000), Fyne takes
 *  that as a move to no monitor, falls back to the scale of the primary one
 *  and shrinks the window to its minimum size for the "new" scale, which the
 *  restore keeps. The moves of the minimized window are not passed to GLFW,
 *  so Fyne never sees them. The restore moves it back where it was.
 * ================================
 */

const wmMove = 0x0003

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	comctl32        = syscall.NewLazyDLL("comctl32.dll")
	procIsIconic    = user32.NewProc("IsIconic")
	procSetSubclass = comctl32.NewProc("SetWindowSubclass")
	procDefSubclass = comctl32.NewProc("DefSubclassProc")
)

// minimizeProc is the window procedure in front of GLFW's, see above
var minimizeProc = syscall.NewCallback(func(hwnd, msg, wParam, lParam, _, _ uintptr) uintptr {
	if msg == wmMove {
		if iconic, _, _ := procIsIconic.Call(hwnd); iconic != 0 {
			return 0
		}
	}
	r, _, _ := procDefSubclass.Call(hwnd, msg, wParam, lParam)
	return r
})

// KeepSizeWhenMinimized puts minimizeProc in front of the window's procedure.
// The native window is made once the app runs, so it waits for it; the
// subclass must be set on the thread of the window, the main one.
func KeepSizeWhenMinimized(w fyne.Window) {
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	go func() {
		for range 50 {
			done := false
			fyne.DoAndWait(func() {
				nw.RunNative(func(ctx any) {
					c, ok := ctx.(driver.WindowsWindowContext)
					if !ok || c.HWND == 0 {
						return
					}
					_, _, _ = procSetSubclass.Call(c.HWND, minimizeProc, 1, 0)
					done = true
				})
			})
			if done {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}
