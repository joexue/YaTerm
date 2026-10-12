package pty

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/ActiveState/termtest/conpty"
)

type Pty struct {
	row, col int

	in  io.Writer
	out io.Reader
	pty *conpty.ConPty

	process   *os.Process
	closeOnce sync.Once
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

	p.pty.Resize(uint16(col), uint16(row))
}

// defaultProgram is the program when none is set
const defaultProgram = `C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe`

// RunCmd starts the program of the command line cmd, "" for the default
// program. A program that is not found falls back to the default, so a wrong
// setting does not close every new tab.
func (p *Pty) RunCmd(cmd string) error {
	ticker := time.NewTicker(time.Millisecond * 100)
	for range ticker.C {
		if p.col > 0 && p.row > 0 {
			break
		}
	}

	ticker.Stop()

	// CreateProcess does not look in PATH, so the program gets its full path
	path, args, ok := FindProgram(cmd)
	if !ok {
		path, args, ok = FindProgram(defaultProgram)
		if !ok {
			return &exec.Error{Name: cmd, Err: exec.ErrNotFound}
		}
	}

	cpty, err := conpty.New(80, 25)
	if err != nil {
		return err
	}

	dir, _ := os.Getwd()
	pid, _, err := cpty.Spawn(
		path,
		append([]string{path}, args...),
		&syscall.ProcAttr{
			Env: os.Environ(),
			Dir: dir,
		},
	)

	if err != nil {
		return err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	p.process = process
	p.in = cpty.InPipe()
	p.out = cpty.OutPipe()
	p.pty = cpty

	p.pty.Resize(uint16(p.col), uint16(p.row))

	// ConPTY keeps its output pipe open after the shell exits (e.g. "exit"),
	// so Read never gets EOF and the pane never closes. Wait for the shell
	// and close the pseudo console, that ends the pipe and Read returns.
	go func() {
		_, _ = process.Wait()
		p.closePty()
	}()

	return nil
}

// closePty closes the pseudo console once, both the exit of the shell and
// Close can do it
func (p *Pty) closePty() {
	p.closeOnce.Do(func() {
		_ = p.pty.Close()
	})
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
	if p.process == nil {
		return
	}

	_ = p.process.Kill()
	p.closePty()
}
