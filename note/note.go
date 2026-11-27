// Package note is the note widget: text the user writes or sends from a
// terminal, which can be opened from and saved to a file.
package note

// Spec:

// 1. It has right click popup menu includes items: Open, Save, Save As
// 2. If it is not opened file, then the default name is
//    yaterm-note-YYYY-MM-DD-HHMMSS.txt
// 3. "Save as" give user chance to rename the file and choose the directory to
//    store the file
// 4. Like the "Start Logging", we should store the user's choose, then next time,
//    this folder will be default place
// 5. User can edit the text in this widget

// How to call it:
// 1. Add a "New Note" into pane's righ click popup menu, when it is clicked,
//    add a new tab, name it "Note x", x is index increased each time.
// 2. When use open a file, then the "Note x" changed to the file's name. the
//    text content of not widget will come from the file opened.
// 3. When user slect data, user can right click the popup menu choose "Add to Note"
//    then choose the submenu from "Add to Note", submenu includes the items
//    openning note, such as "Note 1", "Note 2" ..., the slected text will append
//    to the selected note content.

import (
	"fmt"
	"image/color"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// prefDir is the preference of the folder a note was last opened from or
// saved to (spec 4)
const prefDir = "noteDir"

// Note is the text of a note, in the terminal's monospace font, not wrapped
// so text from a terminal keeps its columns. Its menu opens and saves it.
type Note struct {
	widget.BaseWidget
	text *noteText

	// OnTitle gets the name of the file the note was opened from or saved
	// to, for its tab
	OnTitle func(title string)

	// OnShortcut gets the shortcuts first, the window's (e.g. switch tab)
	// would not come while the note has the focus. True when it took one,
	// else the text gets it, e.g. Copy.
	OnShortcut func(sc fyne.Shortcut) bool

	// OnClose closes the note, its tab, for Close in the menu, which is left
	// out when it is nil
	OnClose func()

	win  fyne.Window
	file fyne.URI // nil until opened or saved
}

// New makes an empty note in win, its dialogs show there
func New(win fyne.Window) *Note {
	n := &Note{win: win}
	n.text = &noteText{note: n}
	n.text.MultiLine = true
	n.text.Wrapping = fyne.TextWrapOff
	n.text.TextStyle = fyne.TextStyle{Monospace: true}
	n.text.ExtendBaseWidget(n.text)
	n.ExtendBaseWidget(n)
	return n
}

// CreateRenderer is a private method to Fyne which links this widget to its
// renderer, see noteRenderer
func (n *Note) CreateRenderer() fyne.WidgetRenderer {
	r := &noteRenderer{
		note: n,
		// Made with the renderer: Fyne drops the theme of a widget with its
		// renderer, e.g. of a tab not shown for a while
		view:  container.NewThemeOverride(n.text, paper{}),
		cover: canvas.NewRectangle(color.Transparent),
	}
	r.Refresh()
	return r
}

// noteRenderer draws the text on the note's paper and without the border of
// a text input: the note fills its tab, a frame around it says nothing. The
// entry has no option for it, and the border's width is the cursor's too, so
// the border is covered by an outline in the color of the paper, where the
// text never goes.
type noteRenderer struct {
	note  *Note
	view  *container.ThemeOverride // the text in the colors of paper
	cover *canvas.Rectangle
}

// paper is the app's theme with the colors of a note: white with a hint of yellow
// and dark text on it, the light text of a dark theme would not be read there
type paper struct{}

var (
	paperYellow = color.NRGBA{R: 0xFF, G: 0xFD, B: 0xE7, A: 0xFF}
	paperInk    = color.NRGBA{R: 0x21, G: 0x21, B: 0x21, A: 0xFF}
	paperFaint  = color.NRGBA{R: 0x6D, G: 0x5E, B: 0x1A, A: 0xFF}
)

// Color is the paper's for the background and the text, else the app's
func (paper) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameInputBackground:
		return paperYellow
	case theme.ColorNameForeground:
		return paperInk
	case theme.ColorNamePlaceHolder:
		return paperFaint
	}
	return fyne.CurrentApp().Settings().Theme().Color(n, v)
}

// Font is the app's
func (paper) Font(s fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(s)
}

// Icon is the app's
func (paper) Icon(n fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(n)
}

// Size is the app's
func (paper) Size(n fyne.ThemeSizeName) float32 {
	return fyne.CurrentApp().Settings().Theme().Size(n)
}

// Layout sizes the text to the note and the cover over its border, placed
// as the entry places the border
func (r *noteRenderer) Layout(size fyne.Size) {
	r.view.Resize(size)
	b := r.note.Theme().Size(theme.SizeNameInputBorder)
	r.cover.Resize(fyne.NewSize(size.Width-b-.5, size.Height-b-.5))
	r.cover.Move(fyne.NewSquareOffsetPos(b / 2))
	r.cover.StrokeWidth = b + 1 // over the antialiasing of the border too
}

// MinSize is the text's
func (r *noteRenderer) MinSize() fyne.Size {
	return r.view.MinSize()
}

// Refresh gives the cover the color of the paper
func (r *noteRenderer) Refresh() {
	th, v := r.note.Theme(), fyne.CurrentApp().Settings().ThemeVariant()
	r.cover.StrokeColor = paper{}.Color(theme.ColorNameInputBackground, v)
	r.cover.CornerRadius = th.Size(theme.SizeNameInputRadius)
	r.Layout(r.note.Size())
	r.note.text.Refresh()
	r.cover.Refresh()
}

// Objects are the text and the cover over its border
func (r *noteRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.view, r.cover}
}

// Destroy does nothing, the renderer has nothing to free
func (r *noteRenderer) Destroy() {}

// Editor is the text, to focus
func (n *Note) Editor() fyne.Focusable {
	return n.text
}

// noteText is the text of a note: its shortcuts and menu are the note's
type noteText struct {
	widget.Entry
	note *Note
}

// TypedShortcut gives the shortcut to the note, see fyne.Shortcutable
func (t *noteText) TypedShortcut(sc fyne.Shortcut) {
	t.note.typedShortcut(sc)
}

// TappedSecondary shows the note's menu, see fyne.SecondaryTappable
func (t *noteText) TappedSecondary(e *fyne.PointEvent) {
	t.note.showMenu(e)
}

// DefaultName is the name a note saved at now gets (spec 2)
func DefaultName(now time.Time) string {
	return now.Format("yaterm-note-2006-01-02-150405.txt")
}

// Append adds text at the end of the note, on a line of its own
func (n *Note) Append(text string) {
	s := n.text.Text
	if s != "" && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	n.text.SetText(s + text)
}

// typedShortcut gives the shortcut to OnShortcut first, else to the text
func (n *Note) typedShortcut(sc fyne.Shortcut) {
	if n.OnShortcut != nil && n.OnShortcut(sc) {
		return
	}
	n.text.Entry.TypedShortcut(sc)
}

// showMenu shows the menu: the editing of the text, then Open, Save and Save
// As (spec 1), Save As once the note has a file, and Close
func (n *Note) showMenu(e *fyne.PointEvent) {
	c := fyne.CurrentApp().Driver().CanvasForObject(n)
	if c == nil {
		return
	}
	clip := fyne.CurrentApp().Clipboard()
	short := func(sc fyne.Shortcut) func() {
		return func() { n.text.Entry.TypedShortcut(sc) }
	}

	cut := fyne.NewMenuItemWithIcon("Cut", lightIcon(theme.ContentCutIcon()), short(&fyne.ShortcutCut{Clipboard: clip}))
	cp := fyne.NewMenuItemWithIcon("Copy", lightIcon(theme.ContentCopyIcon()), short(&fyne.ShortcutCopy{Clipboard: clip}))
	cut.Disabled = n.text.SelectedText() == ""
	cp.Disabled = cut.Disabled

	items := []*fyne.MenuItem{
		cut,
		cp,
		fyne.NewMenuItemWithIcon("Paste", lightIcon(theme.ContentPasteIcon()), short(&fyne.ShortcutPaste{Clipboard: clip})),
		fyne.NewMenuItem("Select All", short(&fyne.ShortcutSelectAll{})),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItemWithIcon("Open...", lightIcon(theme.FolderOpenIcon()), n.open),
	}
	// A note with no file is saved as a new one, Save asks for the name then
	// and Save As would do the same
	if n.file == nil {
		items = append(items, fyne.NewMenuItemWithIcon("Save...", lightIcon(theme.DocumentSaveIcon()), n.save))
	} else {
		items = append(items,
			fyne.NewMenuItemWithIcon("Save", lightIcon(theme.DocumentSaveIcon()), n.save),
			fyne.NewMenuItemWithIcon("Save As...", lightIcon(theme.DocumentSaveIcon()), n.saveAs),
		)
	}
	if n.OnClose != nil {
		items = append(items,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItemWithIcon("Close", lightIcon(theme.Icon(theme.IconNameWindowClose)), n.OnClose),
		)
	}
	menu := fyne.NewMenu("", items...)

	// The menu is light like the paper, in any theme of the app. Its icons
	// are made light by lightIcon, the theme does not reach them.
	pop := widget.NewPopUpMenu(menu, c)
	container.NewThemeOverride(pop, lightTheme{theme.DefaultTheme()})
	pop.ShowAtPosition(e.AbsolutePosition)
}

// svgFill is a fill color of an SVG, not "none"
var svgFill = regexp.MustCompile(`fill="#[0-9a-fA-F]{6}"`)

// lightIcon is an icon of the theme (one color, e.g. ContentPasteIcon) in the
// text color of the light theme, as it is, not themed again: the menu gives
// its icons the app's colors, the light theme set on it does not reach them
func lightIcon(res fyne.Resource) fyne.Resource {
	r, g, b, _ := lightTheme{theme.DefaultTheme()}.Color(theme.ColorNameForeground, theme.VariantLight).RGBA()
	fill := fmt.Sprintf(`fill="#%02x%02x%02x"`, r>>8, g>>8, b>>8)
	return fyne.NewStaticResource("light_"+res.Name(), svgFill.ReplaceAll(res.Content(), []byte(fill)))
}

// lightTheme is a theme always in its light variant, whatever the app's is
type lightTheme struct {
	fyne.Theme
}

// Color is the light variant's
func (l lightTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return l.Theme.Color(n, theme.VariantLight)
}

// open replaces the text with the one of a file the user picks, the note
// belongs to that file then (call 2)
func (n *Note) open() {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, n.win)
			return
		}
		if r == nil {
			return // cancelled
		}
		defer r.Close()

		b, err := io.ReadAll(r)
		if err != nil {
			dialog.ShowError(err, n.win)
			return
		}
		n.text.SetText(string(b))
		n.setFile(r.URI())
	}, n.win)
	n.startIn(d)
	d.Show()
}

// save writes the note to its file, a note with none is saved as a new one
func (n *Note) save() {
	if n.file == nil {
		n.saveAs()
		return
	}
	w, err := storage.Writer(n.file)
	if err != nil {
		dialog.ShowError(err, n.win)
		return
	}
	n.write(w)
}

// saveAs writes the note to a file the user picks, by default its own or one
// of the default name (spec 2, 3), the note belongs to it then
func (n *Note) saveAs() {
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, n.win)
			return
		}
		if w == nil {
			return // cancelled
		}
		if n.write(w) {
			n.setFile(w.URI())
		}
	}, n.win)

	name := DefaultName(time.Now())
	if n.file != nil {
		name = n.file.Name()
	}
	d.SetFileName(name)
	n.startIn(d)
	d.Show()
}

// write writes the text to w and closes it, false when it failed, the user
// is told
func (n *Note) write(w fyne.URIWriteCloser) bool {
	_, err := io.WriteString(w, n.text.Text)
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		dialog.ShowError(err, n.win)
		return false
	}
	return true
}

// setFile makes the note belong to the file u: its folder is the one to start
// in the next time (spec 4), its name the title of the note
func (n *Note) setFile(u fyne.URI) {
	n.file = u
	fyne.CurrentApp().Preferences().SetString(prefDir, filepath.Dir(u.Path()))
	if n.OnTitle != nil {
		n.OnTitle(u.Name())
	}
}

// startIn opens a dialog in the folder of the note's file, else in the last
// one used (spec 4), sized for a file list
func (n *Note) startIn(d *dialog.FileDialog) {
	dir := fyne.CurrentApp().Preferences().String(prefDir)
	if n.file != nil {
		dir = filepath.Dir(n.file.Path())
	}
	if dir != "" {
		// A folder removed since is left out, the dialog starts in its own
		if l, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
			d.SetLocation(l)
		}
	}
	size := n.win.Canvas().Size()
	d.Resize(fyne.NewSize(min(800, size.Width*0.9), min(560, size.Height*0.9)))
}
