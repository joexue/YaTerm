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

// defaultProgram is the program when none is set
const defaultProgram = "bash"

// RunCmd starts the program of the command line cmd, "" for the default
// program. A program that can not be started falls back to the default, so a
// wrong setting does not close every new tab.
func (p *Pty) RunCmd(cmd string) error {
	ticker := time.NewTicker(time.Millisecond * 100)
	for range ticker.C {
		if p.col > 0 && p.row > 0 {
			break
		}
	}

	ticker.Stop()

	env := os.Environ()
	env = append(env, "TERM=xterm-256color")
	env = append(env, "COLORTERM=truecolor")

	start := func(cmd string) (*os.File, error) {
		path, args, ok := FindProgram(cmd)
		if !ok {
			return nil, &exec.Error{Name: cmd, Err: exec.ErrNotFound}
		}
		p.cmd = exec.Command(path, args...)
		p.cmd.Env = env
		return pty.Start(p.cmd)
	}

	f, err := start(cmd)
	if err != nil && cmd != defaultProgram {
		f, err = start(defaultProgram)
	}

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
	// Typing or pasting into a new pane before its shell has started
	if p.in == nil {
		return 0, errNotStarted
	}
	return p.in.Write(bytes)
}

func (p *Pty) Close() {
	// Closed before the shell started, e.g. right after a split
	if p.cmd == nil || p.cmd.Process == nil {
		return
	}

	syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
	_ = p.cmd.Wait()
	p.pty.Close()
}
