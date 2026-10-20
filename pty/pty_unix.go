//go:build !windows
// +build !windows

package pty

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type Pty struct {
	sized sizeGate

	// mu guards the fields below: the UI thread resizes and writes, the
	// terminal's goroutine starts the program
	mu       sync.Mutex
	row, col int
	file     *os.File // the pty, nil until the program started
	cmd      *exec.Cmd

	exited    chan struct{} // closed when the program ended
	closeOnce sync.Once
}

func New() *Pty {
	return &Pty{sized: newSizeGate(), exited: make(chan struct{})}
}

func (p *Pty) Resize(row, col int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.row == row && p.col == col {
		return
	}

	p.row, p.col = row, col
	p.sized.set(row, col)
	if p.file != nil {
		setSize(p.file, row, col)
	}
}

func setSize(f *os.File, row, col int) {
	_ = pty.Setsize(f, &pty.Winsize{Rows: uint16(row), Cols: uint16(col)})
}

// defaultProgram is the program when none is set: the user's shell, which
// differs between systems (bash on most Linux, zsh on macOS, sh or ksh on the
// BSDs, which have no bash unless it is installed), else the POSIX shell
// every one of them has
var defaultProgram = loginShell()

func loginShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		if _, _, ok := FindProgram(shell); ok {
			return shell
		}
	}
	return "/bin/sh"
}

// RunCmd starts the program of the command line cmd, "" for the default
// program. A program that can not be started falls back to the default, so a
// wrong setting does not close every new tab.
func (p *Pty) RunCmd(cmd string) error {
	p.sized.wait()

	env := os.Environ()
	env = append(env, "TERM=xterm-256color")
	env = append(env, "COLORTERM=truecolor")

	start := func(cmd string) (*exec.Cmd, *os.File, error) {
		path, args, ok := FindProgram(cmd)
		if !ok {
			return nil, nil, &exec.Error{Name: cmd, Err: exec.ErrNotFound}
		}
		c := exec.Command(path, args...)
		c.Env = env
		// On macOS the default shell starts as a login shell, as Terminal.app
		// and iTerm2 do, so it reads e.g. ~/.zprofile where Homebrew sets its
		// PATH. A "-" before the name tells any shell it is a login shell.
		if runtime.GOOS == "darwin" && cmd == defaultProgram {
			c.Args[0] = "-" + filepath.Base(path)
		}
		f, err := pty.Start(c)
		return c, f, err
	}

	c, f, err := start(cmd)
	if err != nil && cmd != defaultProgram {
		c, f, err = start(defaultProgram)
	}
	if err != nil {
		return err
	}

	p.mu.Lock()
	p.cmd, p.file = c, f
	// The size may have changed while it started
	setSize(f, p.row, p.col)
	p.mu.Unlock()

	// Wait for the program right away, so one that ends is no zombie
	go func() {
		_ = c.Wait()
		close(p.exited)
	}()
	return nil
}

func (p *Pty) Read(buf []byte) (int, error) {
	p.mu.Lock()
	f := p.file
	p.mu.Unlock()
	if f == nil {
		return 0, errNotStarted
	}
	return f.Read(buf)
}

func (p *Pty) Write(bytes []byte) (int, error) {
	p.mu.Lock()
	f := p.file
	p.mu.Unlock()
	// Typing or pasting into a new pane before its shell has started
	if f == nil {
		return 0, errNotStarted
	}
	return f.Write(bytes)
}

// Close hangs up, as closing a terminal does: the program and the others of
// its session get SIGHUP, which lets shells save their history, and the pty
// is closed. One still running a moment later is killed. It is done once,
// both the end of the program and the user closing its pane call it.
func (p *Pty) Close() {
	p.mu.Lock()
	c, f := p.cmd, p.file
	p.mu.Unlock()
	// Closed before the shell started, e.g. right after a split
	if c == nil || c.Process == nil {
		return
	}

	p.closeOnce.Do(func() {
		pgid := -c.Process.Pid // the shell leads its own session (Setsid)
		select {
		case <-p.exited:
		default:
			_ = syscall.Kill(pgid, syscall.SIGHUP)
		}
		_ = f.Close()

		go func() {
			select {
			case <-p.exited:
			case <-time.After(3 * time.Second):
				_ = syscall.Kill(pgid, syscall.SIGKILL)
			}
		}()
	})
}

// ConptyDLL is only for Windows, elsewhere it is ""
func ConptyDLL() string {
	return ""
}
