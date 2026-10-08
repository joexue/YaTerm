package ui

// HistoryLines is the maximum number of lines kept after they scroll off the screen
const HistoryLines = 2000

// history is a ring buffer of the lines scrolled off the top of the screen
type history struct {
	lines [][]cell
	head  int // index of the oldest line
	count int
}

func newHistory(max int) *history {
	return &history{lines: make([][]cell, max)}
}

// push adds a line as the newest. When full the oldest line is dropped and
// returned so its memory can be reused, otherwise it returns nil.
func (h *history) push(line []cell) []cell {
	max := len(h.lines)
	if max == 0 {
		return line
	}

	if h.count < max {
		h.lines[(h.head+h.count)%max] = line
		h.count++
		return nil
	}

	old := h.lines[h.head]
	h.lines[h.head] = line
	h.head = (h.head + 1) % max
	return old
}

// at returns the i-th line, 0 is the oldest
func (h *history) at(i int) []cell {
	return h.lines[(h.head+i)%len(h.lines)]
}

func (h *history) len() int {
	return h.count
}
