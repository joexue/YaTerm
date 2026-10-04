package terminal

const (
	decStateCONTROL0 = iota
	decStateCONTROL1 = iota
	decStateCONTENT
)

func (t *Terminal) ProcessDec(r rune) {
	if t.decState == decStateCONTROL0 {
		if r == '0' {
			t.decState = decStateCONTENT
		} else if r == 'A' {
			t.state = stateGROUND
		} else if r == 'B' {
			t.state = stateGROUND
		}
	} else if t.decState == decStateCONTROL1 {
		if r == '0' {
			t.decState = decStateCONTENT
		} else if r == 'A' {
			t.state = stateGROUND
		} else if r == 'B' {
			t.state = stateGROUND
		}
	}

	t.handleDec(r)
}

func (t *Terminal) handleDec(r rune) {
}
