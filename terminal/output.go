package terminal

import (
	"unicode/utf8"
)

func (t *Terminal) ProcessOutput(bytes []byte, num int) {
	var (
		size int
		r    rune
	)

	for i := 0; i < num; i = i + size {
		bytes = bytes[size:]
		r, size = utf8.DecodeRune(bytes)
		if size == 0 {
			break
		}

		if f := t.OnProject; f != nil {
			f(r, t.cursorY, t.cursorX, t.fg, t.bg)
			t.cursorX += size
		}
	}
}
