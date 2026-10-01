package terminal

type Terminal struct {
	width, height int
}

func New() *Terminal {
	t := &Terminal{}

	return t
}

func (t *Terminal) Resize(width, height int) {
	t.width = width
	t.height = height
}

func (t *Terminal) Size() (int, int) {
	return t.width, t.height
}
