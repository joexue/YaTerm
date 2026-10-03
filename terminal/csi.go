package terminal

func (t *Terminal) ProcessCsi(r rune) {
	//t.stateCode += string(r)
	if (r < '0' || r > '9') && r != ';' && r != '=' && r != '?' && r != '>' {
		/*
			code := t.state.code
			fyne.Do(func() {
				t.handleEscape(code)
			})
			t.state.code = ""
		*/
		t.state = GROUND
	}
}
