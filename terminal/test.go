package terminal

func (t *Terminal) ProcessTest(r rune) {
	switch r {
	case '8':
		t.testAlign()
	}

	t.state = stateGROUND
}

func (t *Terminal) testAlign() {
	if f := t.OnProject; f != nil {
		for i := 0; i < t.row; i++ {
			for j := 0; j < t.col; j++ {
				f('E', i, j, t.fg, t.bg)
			}
		}
	}

	t.cursorX = 0
	t.cursorY = 0
}
