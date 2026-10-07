package terminal

func (t *Terminal) processDcs(r rune) {
	t.state = stateGROUND
}
