package pty

import (
	"io"
	"os"
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

func (p *Pty) RunCmd(_ string) error {
	ticker := time.NewTicker(time.Millisecond * 100)
	for range ticker.C {
		if p.col > 0 && p.row > 0 {
			break
		}
	}

	ticker.Stop()

	cpty, err := conpty.New(80, 25)
	if err != nil {
		return err
	}

	dir, _ := os.Getwd()
	pid, _, err := cpty.Spawn(
		"C:\\WINDOWS\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
		[]string{},
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
