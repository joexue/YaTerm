// Package markdown shows Markdown text on a white page, in the light theme
// whatever the app's theme is, e.g. text selected in a terminal, and edits
// it in a note.
package markdown

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"yaterm/note"
	yatheme "yaterm/theme"
)

// New makes the page of the Markdown text in win, its dialogs show there. It
// shows the text rendered (the preview) and, by Edit in its menu, the
// Markdown itself to edit, in a note, Preview shows it again.
func New(win fyne.Window, text string) *Page {
	p := &Page{content: container.NewStack()}

	p.editor = note.New(win)
	p.editor.SetText(text)
	p.editor.DefaultName = func(now time.Time) string {
		return now.Format("yaterm-markdown-2006-01-02-150405.md")
	}
	p.editor.FolderPref = prefDir
	p.editor.MenuItems = func() []*fyne.MenuItem {
		edit := fyne.NewMenuItemWithIcon("Edit", yatheme.LightIcon(theme.DocumentCreateIcon()), nil)
		edit.Disabled = true
		return []*fyne.MenuItem{
			edit,
			fyne.NewMenuItemWithIcon("Preview", yatheme.LightIcon(theme.VisibilityIcon()), p.preview),
		}
	}
	p.editor.OnTitle = func(title string) {
		if p.OnTitle != nil {
			p.OnTitle(title)
		}
	}
	p.editor.OnShortcut = func(sc fyne.Shortcut) bool {
		return p.OnShortcut != nil && p.OnShortcut(sc)
	}
	p.editor.OnClose = func() {
		if p.OnClose != nil {
			p.OnClose()
		}
	}

	p.ExtendBaseWidget(p)
	p.preview()
	return p
}

// prefDir is the preference of the folder a Markdown file was last opened
// from or saved to
const prefDir = "markdownDir"

// Page is a Markdown page, its text rendered or edited, with a menu of its
// own on a right click
type Page struct {
	widget.BaseWidget

	// OnClose closes the page, its tab, for Close in the menu, which is left
	// out when it is nil
	OnClose func()

	// OnTitle gets the name of the file the Markdown was opened from or
	// saved to, for its tab
	OnTitle func(title string)

	// OnShortcut gets the shortcuts typed in the editor first, see
	// note.Note.OnShortcut
	OnShortcut func(sc fyne.Shortcut) bool

	editor  *note.Note       // the Markdown, the page's text
	text    *widget.RichText // the rendered text, while previewed
	editing bool
	content *fyne.Container // the preview or the editor
}

// CreateRenderer is a private method to Fyne which links this widget to its
// renderer
func (p *Page) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.content)
}

// Editor is the text to focus while editing, nil while previewed
func (p *Page) Editor() fyne.Focusable {
	if !p.editing {
		return nil
	}
	return p.editor.Editor()
}

// preview shows the Markdown of the editor rendered, its lines wrapped at
// the width of the window. The rich text is light by a theme override, but
// Fyne gives two parts the app's colors: text nested in a list (its parent
// is not set, Fyne then looks at the app's theme) and code blocks (drawn with
// the app's theme). So lists and code blocks are made lines of the text.
func (p *Page) preview() {
	rt := widget.NewRichTextFromMarkdown(p.editor.Text())
	rt.Segments = flatten(rt.Segments)
	rt.Wrapping = fyne.TextWrapWord
	p.text = rt

	page := canvas.NewRectangle(yatheme.Light().Color(theme.ColorNameBackground, theme.VariantLight))
	// A scroll container, not rt.Scroll: Fyne drops the renderer of a widget
	// not drawn for a minute (e.g. in a tab in the background), and RichText
	// with its own scroll makes a new one that draws nothing, the tab would
	// be empty when shown again
	view := container.NewThemeOverride(container.NewStack(page, container.NewVScroll(rt)), yatheme.Light())

	p.show(view, false)
}

// edit shows the Markdown in the editor, focused
func (p *Page) edit() {
	p.show(p.editor, true)
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		c.Focus(p.editor.Editor())
	}
}

// show puts o on the page
func (p *Page) show(o fyne.CanvasObject, editing bool) {
	p.editing = editing
	p.content.Objects = []fyne.CanvasObject{o}
	p.content.Refresh()
}

// TappedSecondary shows the menu of the preview, light like the page, see
// fyne.SecondaryTappable. The editor has its own, the note's.
func (p *Page) TappedSecondary(e *fyne.PointEvent) {
	c := fyne.CurrentApp().Driver().CanvasForObject(p)
	if c == nil || p.editing {
		return
	}
	copyIcon := yatheme.LightIcon(theme.ContentCopyIcon())
	clip := fyne.CurrentApp().Clipboard()
	items := []*fyne.MenuItem{
		fyne.NewMenuItemWithIcon("Edit", yatheme.LightIcon(theme.DocumentCreateIcon()), p.edit),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItemWithIcon("Copy as Markdown", copyIcon, func() { clip.SetContent(p.editor.Text()) }),
		fyne.NewMenuItemWithIcon("Copy as Text", copyIcon, func() { clip.SetContent(p.plainText()) }),
		fyne.NewMenuItemSeparator(),
	}
	items = append(items, p.editor.SaveMenuItems()...)
	if p.OnClose != nil {
		items = append(items,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItemWithIcon("Close", yatheme.LightIcon(theme.Icon(theme.IconNameWindowClose)), p.OnClose),
		)
	}
	pop := widget.NewPopUpMenu(fyne.NewMenu("", items...), c)
	container.NewThemeOverride(pop, yatheme.Light())
	pop.ShowAtPosition(e.AbsolutePosition)
}

// plainText is the text of the page without the Markdown: the text of its
// parts, a block (a heading, the end of a paragraph, a line of a list or of
// code) ends a line. RichText.String has no line ends.
func (p *Page) plainText() string {
	var b strings.Builder
	for _, s := range p.text.Segments {
		b.WriteString(s.Textual())
		if !s.Inline() {
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// flatten makes lists and code blocks lines of the text
func flatten(segs []widget.RichTextSegment) []widget.RichTextSegment {
	var out []widget.RichTextSegment
	for _, s := range segs {
		switch s := s.(type) {
		case *widget.ListSegment:
			out = append(out, listLines(s, 0)...)
		case *widget.CodeBlockSegment:
			out = append(out, codeLines(s.Text)...)
		default:
			out = append(out, s)
		}
	}
	return out
}

// listLines is a line for each item of l, its bullet or number indented by
// level. A nested list is an item of its own, after the one it belongs to,
// its lines go one level deeper.
func listLines(l *widget.ListSegment, level int) []widget.RichTextSegment {
	var out []widget.RichTextSegment
	indent := strings.Repeat("    ", level)
	n := 0
	for _, item := range l.Items {
		if sub, ok := item.(*widget.ListSegment); ok {
			out = append(out, listLines(sub, level+1)...)
			continue
		}
		n++
		bullet := "•  "
		if l.Ordered {
			bullet = fmt.Sprintf("%d. ", n)
		}
		line := []widget.RichTextSegment{&widget.TextSegment{Text: indent + bullet, Style: widget.RichTextStyleInline}}
		var below []widget.RichTextSegment

		parts := []widget.RichTextSegment{item}
		if p, ok := item.(*widget.ParagraphSegment); ok {
			parts = p.Texts
		}
		for _, part := range parts {
			switch p := part.(type) {
			case *widget.ListSegment:
				below = append(below, listLines(p, level+1)...)
			case *widget.CodeBlockSegment:
				below = append(below, codeLines(p.Text)...)
			case *widget.TextSegment:
				p.Style.Inline = true
				line = append(line, p)
			default:
				line = append(line, p)
			}
		}
		// A block ends the line
		line = append(line, &widget.TextSegment{Style: widget.RichTextStyleParagraph})

		out = append(out, line...)
		out = append(out, below...)
	}
	return out
}

// codeLines is a line of the text for each line of a code block, in the style
// of inline code (monospace on the background of code), so it wraps as the
// text does. Tabs are four spaces, an empty line one space to be a line.
func codeLines(code string) []widget.RichTextSegment {
	var out []widget.RichTextSegment
	for _, l := range strings.Split(strings.TrimRight(code, "\n"), "\n") {
		l = strings.ReplaceAll(l, "\t", "    ")
		if l == "" {
			l = " "
		}
		style := widget.RichTextStyleCodeInline
		style.Inline = false
		out = append(out, &widget.TextSegment{Text: l, Style: style})
	}
	return out
}
