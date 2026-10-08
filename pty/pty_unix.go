//go:build !windows
// +build !windows

package pty

import (
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type Pty struct {
	row, col int

	in  io.Writer
	out io.Reader
	pty io.Closer
	cmd *exec.Cmd
}

func New() *Pty {
	return &Pty{}
}

func (p *Pty) Resize(row, col int) {
	if p.row == row && p.col == col {
		return
	}

	p.row, p.col = row, col

	if p.pty == nil {
		return
	}
	_ = pty.Setsize(p.pty.(*os.File), &pty.Winsize{
		Rows: uint16(row),
		Cols: uint16(col),
		X:    uint16(0),
		Y:    uint16(0),
	})
}

func (p *Pty) RunCmd(_ string) error {
	ticker := time.NewTicker(time.Millisecond * 100)
	for range ticker.C {
		if p.col > 0 && p.row > 0 {
			break
		}
	}

	ticker.Stop()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}

	env := os.Environ()
	env = append(env, "TERM=xterm-256color")
	env = append(env, "COLORTERM=truecolor")
	p.cmd = exec.Command(shell)

	// Start the command with a pty.
	f, err := pty.Start(p.cmd)

	if err == nil {
		p.in = f
		p.out = f
		p.pty = f

		_ = pty.Setsize(p.pty.(*os.File), &pty.Winsize{
			Rows: uint16(p.row),
			Cols: uint16(p.col),
			X:    uint16(0),
			Y:    uint16(0),
		})
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
	syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
	_ = p.cmd.Wait()
	p.pty.Close()
}
