// Package log logs the output of a terminal to a file: the user picks the
// file in a dialog, which starts in the folder of the last log. The writing
// is done by the terminal, see terminal.StartLog.
package log

// Spec:
// 1 when user click right click menu "Start Logging", give user a file dialog to let
//   user choose the place to write logfile
// 2 if user choose the directory, the directory should save into the config file,
//   then next time, when use click "Start Logging", this dir will be the default place
// 3 The default log file name is: yaterm-YYYY-MM-DD-HHMMSS.log, user can change it
//   to any.
// 4 Once the use confirm the file name and place, the yaterm starts to log, anything
//   output the terminal of pane where user take action will store to the log file to.
// 5 Once the log starts, the menu "Start Logging" change to "Stop Logging"
// 6 When user click "Stop Logging", sotp loging output to log file and close the file,
//   The menu change back to "Start Logging".

import (
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

// prefDir is the preference of the folder the last log went to (spec 2)
const prefDir = "logDir"

// DefaultName is the name a log started at now gets (spec 3)
func DefaultName(now time.Time) string {
	return now.Format("yaterm-2006-01-02-150405.log")
}

// Choose asks for the file to log to (spec 1-3): the dialog starts in the
// folder of the last log with the default name, which the user may change.
// start gets the file, created by the dialog, nothing happens when the user
// cancels. The folder is kept for the next time.
func Choose(win fyne.Window, start func(f fyne.URIWriteCloser)) {
	prefs := fyne.CurrentApp().Preferences()

	d := dialog.NewFileSave(func(f fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if f == nil {
			return // cancelled
		}
		prefs.SetString(prefDir, filepath.Dir(f.URI().Path()))
		start(f)
	}, win)

	d.SetFileName(DefaultName(time.Now()))
	if dir := prefs.String(prefDir); dir != "" {
		// A folder removed since is left out, the dialog starts in its own
		if l, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
			d.SetLocation(l)
		}
	}

	// The dialog is small by default, a file list needs room
	size := win.Canvas().Size()
	d.Resize(fyne.NewSize(min(800, size.Width*0.9), min(560, size.Height*0.9)))
	d.Show()
}
