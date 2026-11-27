// Package markdown shows Markdown text on a white page, in the light theme
// whatever the app's theme is, e.g. text selected in a terminal.
package markdown

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	yatheme "yaterm/theme"
)

// New makes the page showing text as Markdown. The rich text is light by a
// theme override, but Fyne gives two parts the app's colors: text nested in
// a list (its parent is not set, Fyne then looks at the app's theme) and code
// blocks (drawn with the app's theme). Lists are made lines of their own and
// code blocks are drawn here, in the light colors.
func New(text string) fyne.CanvasObject {
	rt := widget.NewRichTextFromMarkdown(text)
	rt.Segments = flatten(rt.Segments)

	page := canvas.NewRectangle(yatheme.Light().Color(theme.ColorNameBackground, theme.VariantLight))
	// A scroll container, not rt.Scroll: Fyne drops the renderer of a widget
	// not drawn for a minute (e.g. in a tab in the background), and RichText
	// with its own scroll makes a new one that draws nothing, the tab would
	// be empty when shown again
	return container.NewThemeOverride(container.NewStack(page, container.NewScroll(rt)), yatheme.Light())
}

// flatten makes lists lines of the text and code blocks the ones of this page
func flatten(segs []widget.RichTextSegment) []widget.RichTextSegment {
	var out []widget.RichTextSegment
	for _, s := range segs {
		switch s := s.(type) {
		case *widget.ListSegment:
			out = append(out, listLines(s, 0)...)
		case *widget.CodeBlockSegment:
			out = append(out, &codeBlock{text: s.Text})
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
				below = append(below, &codeBlock{text: p.Text})
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

// codeBlock is a code block in the light colors: monospace lines on a light
// box, scrolled sideways when wider than the page. Its text is not selected.
type codeBlock struct {
	text string
}

// Inline is false, a code block is a block of its own
func (c *codeBlock) Inline() bool { return false }

// Textual is the code
func (c *codeBlock) Textual() string { return c.text }

// Visual is the box with the lines of the code
func (c *codeBlock) Visual() fyne.CanvasObject {
	th, v := yatheme.Light(), theme.VariantLight

	bg := canvas.NewRectangle(th.Color(theme.ColorNameInputBackground, v))
	bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	bg.StrokeWidth = 1
	bg.CornerRadius = th.Size(theme.SizeNameInputRadius)

	lines := container.New(layout.NewCustomPaddedVBoxLayout(0))
	for _, l := range strings.Split(strings.TrimRight(c.text, "\n"), "\n") {
		t := canvas.NewText(strings.ReplaceAll(l, "\t", "    "), th.Color(theme.ColorNameForeground, v))
		t.TextStyle = fyne.TextStyle{Monospace: true}
		t.TextSize = th.Size(theme.SizeNameText)
		lines.Add(t)
	}
	return container.NewStack(bg, container.NewPadded(container.NewHScroll(lines)))
}

// Update does nothing, the code does not change
func (c *codeBlock) Update(fyne.CanvasObject) {}

// Select does nothing, see codeBlock
func (c *codeBlock) Select(_, _ fyne.Position) {}

// SelectedText is empty, see codeBlock
func (c *codeBlock) SelectedText() string { return "" }

// Unselect does nothing, see codeBlock
func (c *codeBlock) Unselect() {}
