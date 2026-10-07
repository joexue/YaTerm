package terminal

func (t *Terminal) processTest(r rune) {
	switch r {
	case '8':
		t.testAlign()
	}

	t.state = stateGROUND
}

func (t *Terminal) testAlign() {
	for i := 0; i < t.row; i++ {
		for j := 0; j < t.col; j++ {
			t.screen.Project('E', i, j, t.fg, t.bg, t.style)
		}
	}

	t.cursorX = 0
	t.cursorY = 0
}
