package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Characters that end a word for double click selection, '/', '.', '-' and
// '_' are not among them so a path or a file name is selected as one word.
const wordDelimiters = " \t\"'`()[]{}<>|;,"

// selPos is a cell in the whole text: line is the absolute line (history and
// screen together, see Screen.histTotal), col the column.
type selPos struct {
	line, col int
}

func (p selPos) before(o selPos) bool {
	return p.line < o.line || p.line == o.line && p.col < o.col
}

// selection runs from anchor (where the mouse went down) to head (where it is
// now), in either direction, both cells included.
type selection struct {
	active       bool
	anchor, head selPos
}

// bounds returns the selection with start before end
func (sel selection) bounds() (selPos, selPos) {
	if sel.head.before(sel.anchor) {
		return sel.head, sel.anchor
	}
	return sel.anchor, sel.head
}

// cellAt returns the cell under a position in the screen. Called with mu held.
func (s *Screen) cellAt(pos fyne.Position) selPos {
	row := max(0, min(int(pos.Y/s.cellSize.Height), s.row-1))
	col := max(0, min(int(pos.X/s.cellSize.Width), s.col-1))
	return selPos{line: s.histTotal - s.offset + row, col: col}
}

// lineAt returns the cells of an absolute line, false when it is not kept
// any more. Called with mu held.
func (s *Screen) lineAt(abs int) (line, bool) {
	base := s.histTotal - s.hist.len()
	switch {
	case abs < base:
		return line{}, false
	case abs < s.histTotal:
		return s.hist.at(abs - base), true
	case abs-s.histTotal < len(s.lines):
		return s.lines[abs-s.histTotal], true
	}
	return line{}, false
}

// SelectStart clears the selection and remembers where a drag starts
func (s *Screen) SelectStart(pos fyne.Position) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.cellAt(pos)
	s.setSelection(selection{anchor: p, head: p})
}

// SelectTo selects from the drag start to pos
func (s *Screen) SelectTo(pos fyne.Position) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sel := s.sel
	sel.head = s.cellAt(pos)
	sel.active = true
	s.setSelection(sel)
}

// SelectWord selects the word under pos, for a double click
func (s *Screen) SelectWord(pos fyne.Position) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.cellAt(pos)
	l, ok := s.lineAt(p.line)
	if !ok || p.col >= len(l.cells) || !isWordRune(l.cells[p.col].r) {
		s.setSelection(selection{})
		return
	}

	from, to := p.col, p.col
	for from > 0 && isWordRune(l.cells[from-1].r) {
		from--
	}
	for to < len(l.cells)-1 && isWordRune(l.cells[to+1].r) {
		to++
	}

	s.setSelection(selection{
		active: true,
		anchor: selPos{line: p.line, col: from},
		head:   selPos{line: p.line, col: to},
	})
}

// ClearSelection removes the selection
func (s *Screen) ClearSelection() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.setSelection(selection{})
}

// setSelection changes the selection and redraws. Called with mu held.
func (s *Screen) setSelection(sel selection) {
	if sel == s.sel {
		return
	}

	redraw := sel.active || s.sel.active
	s.sel = sel
	if redraw {
		// Rows without changes are cheap, their objects are not refreshed
		s.viewDirty = true
		s.render()
	}
}

// SelectedText returns the selected text. Lines that were wrapped by the
// terminal are joined, others end with a new line, trailing blanks of each
// line are dropped.
func (s *Screen) SelectedText() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.sel.active {
		return ""
	}

	start, end := s.sel.bounds()
	var b strings.Builder
	for abs := start.line; abs <= end.line; abs++ {
		l, ok := s.lineAt(abs)
		if !ok {
			continue
		}

		from, to := 0, len(l.cells)-1
		if abs == start.line {
			from = start.col
		}
		if abs == end.line {
			to = min(to, end.col)
		}

		var lb strings.Builder
		for c := from; c <= to; c++ {
			r := l.cells[c].r
			if r == 0 {
				r = ' '
			}
			lb.WriteRune(r)
		}

		text := lb.String()
		joined := l.wrapped && abs != end.line
		if !joined {
			text = strings.TrimRight(text, " ")
		}
		b.WriteString(text)
		if !joined && abs != end.line {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// highlight returns the cells of a row with the search matches and the
// selection painted in, or the cells as they are when the row has neither.
// Called with mu held.
func (s *Screen) highlight(l *screenLine, cells []cell, abs int) []cell {
	selected := false
	var start, end selPos
	if s.sel.active {
		start, end = s.sel.bounds()
		selected = abs >= start.line && abs <= end.line
	}
	if !selected && !s.hasMatches(abs) {
		return cells
	}

	// A copy as wide as the screen, the selection may go past the line end
	buf := l.selBuf[:0]
	buf = append(buf, cells[:min(len(cells), s.col)]...)
	for len(buf) < s.col {
		buf = append(buf, cell{})
	}
	l.selBuf = buf

	s.markMatches(buf, abs)
	if !selected {
		return buf
	}

	from, to := 0, s.col-1
	if abs == start.line {
		from = start.col
	}
	if abs == end.line {
		to = min(to, end.col)
	}

	// Opaque, a see-through color would show where background rectangles
	// overlap to hide the seams between them. Drawn over the matches.
	selColor := over(theme.Color(theme.ColorNameSelection), theme.Color(theme.ColorNameBackground))
	for c := from; c <= to; c++ {
		buf[c].bg = selColor
	}
	return buf
}
