// Package markdown shows Markdown text on a white page, in the light theme
// whatever the app's theme is, e.g. text selected in a terminal.
package markdown

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	yatheme "yaterm/theme"
)

// New makes the page showing text as Markdown, its lines wrapped at the width
// of the window. The rich text is light by a theme override, but Fyne gives
// two parts the app's colors: text nested in a list (its parent is not set,
// Fyne then looks at the app's theme) and code blocks (drawn with the app's
// theme). So lists and code blocks are made lines of the text itself.
func New(text string) *Page {
	rt := widget.NewRichTextFromMarkdown(text)
	rt.Segments = flatten(rt.Segments)
	rt.Wrapping = fyne.TextWrapWord

	page := canvas.NewRectangle(yatheme.Light().Color(theme.ColorNameBackground, theme.VariantLight))
	// A scroll container, not rt.Scroll: Fyne drops the renderer of a widget
	// not drawn for a minute (e.g. in a tab in the background), and RichText
	// with its own scroll makes a new one that draws nothing, the tab would
	// be empty when shown again
	p := &Page{
		source: text,
		text:   rt,
		view:   container.NewThemeOverride(container.NewStack(page, container.NewVScroll(rt)), yatheme.Light()),
	}
	p.ExtendBaseWidget(p)
	return p
}

// Page is a Markdown page, with a menu of its own on a right click
type Page struct {
	widget.BaseWidget

	// OnClose closes the page, its tab, for Close in the menu, which is left
	// out when it is nil
	OnClose func()

	source string            // the Markdown shown
	text   *widget.RichText  // the page's text
	view   fyne.CanvasObject // the text on its white page
}

// CreateRenderer is a private method to Fyne which links this widget to its
// renderer
func (p *Page) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.view)
}

// TappedSecondary shows the menu, light like the page, see
// fyne.SecondaryTappable
func (p *Page) TappedSecondary(e *fyne.PointEvent) {
	c := fyne.CurrentApp().Driver().CanvasForObject(p)
	if c == nil {
		return
	}
	copyIcon := yatheme.LightIcon(theme.ContentCopyIcon())
	clip := fyne.CurrentApp().Clipboard()
	items := []*fyne.MenuItem{
		fyne.NewMenuItemWithIcon("Copy as Markdown", copyIcon, func() { clip.SetContent(p.source) }),
		fyne.NewMenuItemWithIcon("Copy as Text", copyIcon, func() { clip.SetContent(p.plainText()) }),
	}
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
