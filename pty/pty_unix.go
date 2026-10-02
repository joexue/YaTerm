//go:build !windows
// +build !windows

package pty

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type Pty struct {
	row, col int

	in  io.Writer
	out io.Reader
	pty io.Closer
}

func New() *Pty {
	return &Pty{}
}

func (p *Pty) Resize(row, col int, width, height float32) {
	if p.pty == nil {
		return
	}

	_ = pty.Setsize(p.pty.(*os.File), &pty.Winsize{
		Rows: uint16(row),
		Cols: uint16(col),
		X:    uint16(width),
		Y:    uint16(height),
	})
}

func (p *Pty) RunCmd(_ string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}

	env := os.Environ()
	env = append(env, "TERM=xterm-256color")
	env = append(env, "COLORTERM=truecolor")
	c := exec.Command(shell)

	// Start the command with a pty.
	f, err := pty.Start(c)

	if err == nil {
		p.in = f
		p.out = f
		p.pty = f
	}
	return err
}

func (p *Pty) Read(buf []byte) (int, error) {
	return p.out.Read(buf)
}

func (p *Pty) Write(bytes []byte) (int, error) {
	return p.in.Write(bytes)
}

func (p *Pty) Close() {
	p.pty.Close()
}
