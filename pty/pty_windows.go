package pty

import (
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/ActiveState/termtest/conpty"
)

type Pty struct {
	row, col int

	in  io.Writer
	out io.Reader
	pty *conpty.ConPty
	cmd *exec.Cmd

	process *os.Process
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
	ticker := time.NewTicker(time.Second)
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

	p.cmd = &exec.Cmd{}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	//_, err = process.Wait()

	p.process = process
	p.in = cpty.InPipe()
	p.out = cpty.OutPipe()
	p.pty = cpty

	p.pty.Resize(uint16(p.col), uint16(p.row))

	return err
}

func (p *Pty) Read(buf []byte) (int, error) {
	return p.out.Read(buf)
}

func (p *Pty) Write(bytes []byte) (int, error) {
	return p.in.Write(bytes)
}

func (p *Pty) Close() {
	_ = p.process.Kill()
	_ = p.cmd.Wait()
	p.pty.Close()
}
