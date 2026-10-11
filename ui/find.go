package ui

import (
	"image/color"
	"regexp"
	"sort"
	"strings"

	"fyne.io/fyne/v2/theme"
	"yaterm/terminal"
)

// findMatch is one search hit from start to end, both cells included, in
// absolute lines like the selection, so it stays on its text while scrolling.
type findMatch struct {
	start, end selPos
}

// findState is the search of the search bar. stale means the content changed
// since the matches were found, they are found again on the next flush.
type findState struct {
	re      *regexp.Regexp
	matches []findMatch
	cur     int
	stale   bool
}

// Search highlights every match of re in the history and the screen, and
// makes the one nearest to the view in the search direction the current
// one. A nil re clears the search. Returns the number of matches.
func (s *Screen) Search(re *regexp.Regexp, up bool) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.find = findState{re: re}
	if re != nil {
		s.runSearch()
		s.find.cur = s.nearestMatch(up)
		s.revealMatch()
	}

	s.viewDirty = true
	s.render()
	s.updateCursor()
	return len(s.find.matches)
}

// FindNext makes the next match the current one, up goes to older text,
// it wraps around at the ends. Returns the current match, 1 based, and the
// number of matches.
func (s *Screen) FindNext(up bool) (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.find.stale {
		s.runSearch()
	}

	n := len(s.find.matches)
	if n == 0 {
		return 0, 0
	}

	if up {
		s.find.cur = (s.find.cur - 1 + n) % n
	} else {
		s.find.cur = (s.find.cur + 1) % n
	}
	s.revealMatch()

	s.viewDirty = true
	s.render()
	s.updateCursor()
	return s.find.cur + 1, n
}

// runSearch finds all matches again. A logical line, the lines joined by
// auto wrap, is searched as one string, so a match may go over a wrap.
// Called with mu held.
func (s *Screen) runSearch() {
	s.find.stale = false

	// Keep the current match where it was if it is still found
	var curStart selPos
	hadCur := s.find.cur < len(s.find.matches)
	if hadCur {
		curStart = s.find.matches[s.find.cur].start
	}

	s.find.matches = s.find.matches[:0]
	if s.find.re == nil {
		return
	}

	var (
		sb strings.Builder
		at []selPos // the cell of each byte of the string
	)

	base := s.histTotal - s.hist.len()
	end := s.histTotal + len(s.lines)
	for abs := base; abs < end; {
		sb.Reset()
		at = at[:0]

		// Join the lines of one logical line
		for {
			l, _ := s.lineAt(abs)
			for c, cl := range l.cells {
				r := cl.r
				if r == terminal.WideTail || r == terminal.WidePad {
					continue // not text of its own
				}
				if r == 0 {
					r = ' '
				}
				n, _ := sb.WriteRune(r)
				for range n {
					at = append(at, selPos{line: abs, col: c})
				}
			}
			abs++
			if !l.wrapped || abs >= end {
				break
			}
		}

		// The blank end is not text, so "$" matches after the last character
		text := strings.TrimRight(sb.String(), " ")
		for _, m := range s.find.re.FindAllStringIndex(text, -1) {
			if m[1] > m[0] {
				s.find.matches = append(s.find.matches, findMatch{start: at[m[0]], end: at[m[1]-1]})
			}
		}
	}

	s.find.cur = 0
	if hadCur {
		for i, m := range s.find.matches {
			if m.start == curStart {
				s.find.cur = i
				break
			}
		}
	}
}

// nearestMatch returns the match to start from: searching up, the last one
// that starts in or above the bottom of the view, searching down, the first
// one in or below its top. Called with mu held.
func (s *Screen) nearestMatch(up bool) int {
	n := len(s.find.matches)
	if n == 0 {
		return 0
	}

	top := s.histTotal - s.offset
	if up {
		bottom := top + s.row - 1
		i := sort.Search(n, func(i int) bool { return s.find.matches[i].start.line > bottom })
		return (i - 1 + n) % n
	}

	i := sort.Search(n, func(i int) bool { return s.find.matches[i].start.line >= top })
	return i % n
}

// revealMatch scrolls the view so the current match is in it, in the middle
// when it has to move. Called with mu held.
func (s *Screen) revealMatch() {
	if s.find.cur >= len(s.find.matches) {
		return
	}

	line := s.find.matches[s.find.cur].start.line
	top := s.histTotal - s.offset
	if line >= top && line < top+s.row {
		return
	}

	newTop := line - s.row/2
	s.offset = max(0, min(s.histTotal-newTop, s.hist.len()))
}

// markMatches paints the matches on line abs into buf, the current one
// stronger and with the text in the background color so it stays readable.
// Called with mu held.
func (s *Screen) markMatches(buf []cell, abs int) {
	ms := s.find.matches
	i := sort.Search(len(ms), func(i int) bool { return ms[i].end.line >= abs })

	var all, cur, curFg color.Color
	for ; i < len(ms) && ms[i].start.line <= abs; i++ {
		if all == nil {
			bg := theme.Color(theme.ColorNameBackground)
			hit := theme.Color(theme.ColorNameWarning)
			all = over(withAlpha(hit, 0x60), bg)
			cur = over(hit, bg)
			curFg = bg
		}

		from, to := 0, len(buf)-1
		if ms[i].start.line == abs {
			from = ms[i].start.col
		}
		if ms[i].end.line == abs {
			to = min(to, ms[i].end.col)
		}
		from, to = wholeWide(buf, from, to)

		for c := from; c <= to; c++ {
			if i == s.find.cur {
				buf[c].bg, buf[c].fg = cur, curFg
			} else {
				buf[c].bg = all
			}
		}
	}
}

// hasMatches tells if line abs has a match. Called with mu held.
func (s *Screen) hasMatches(abs int) bool {
	ms := s.find.matches
	i := sort.Search(len(ms), func(i int) bool { return ms[i].end.line >= abs })
	return i < len(ms) && ms[i].start.line <= abs
}

// withAlpha returns c with alpha a, as a non premultiplied color
func withAlpha(c color.Color, a uint8) color.Color {
	r, g, b, ca := c.RGBA()
	if ca == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: uint8(r * 0xffff / ca >> 8),
		G: uint8(g * 0xffff / ca >> 8),
		B: uint8(b * 0xffff / ca >> 8),
		A: a,
	}
}
