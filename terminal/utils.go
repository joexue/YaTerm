package terminal

func isC1ControlCharacter(r rune) bool {
	return r >= 0x80 && r <= 0x9F
}

func c1ToC0ControlCharacter(r rune) rune {
	return r - 0x40
}
