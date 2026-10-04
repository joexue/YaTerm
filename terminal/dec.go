package terminal

const (
	decStateCONTROL = iota
	decStateCONTENT
)

func (t *Terminal) ProcessDec(r rune) {
	if t.decState == decStateCONTROL {
		if r == '0' {
			t.decState = decStateCONTENT
		} else if r == 'B' {
			t.state = stateGROUND
		}
	} else {
		t.handleDec(r)
	}
}

func (t *Terminal) handleDec(r rune) {
}
