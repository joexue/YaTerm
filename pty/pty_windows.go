package pty

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

/*
 * ================================
 *  ConPTY. Windows has one built in (kernel32.dll), it drops escape sequences
 *  it does not know, e.g. the DCS that starts tmux control mode (tmux -CC).
 *  The newer one of the Windows Terminal project (conpty.dll with
 *  OpenConsole.exe, NuGet Microsoft.Windows.Console.ConPTY) passes them on.
 *  It is used when both files are next to yaterm.exe.
 * ================================
 */

// conptyAPI are the ConPTY functions of the DLL in use
type conptyAPI struct {
	create, resize, close *windows.Proc
	dll                   string
}

// loadConpty loads conpty.dll from the directory of yaterm.exe when it is
// there with OpenConsole.exe, which it starts, or else kernel32.dll
var loadConpty = sync.OnceValues(func() (*conptyAPI, error) {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, "OpenConsole.exe")); err == nil {
			if api, err := conptyFrom(filepath.Join(dir, "conpty.dll")); err == nil {
				return api, nil
			}
		}
	}
	return conptyFrom("kernel32.dll")
})

func conptyFrom(name string) (*conptyAPI, error) {
	dll, err := windows.LoadDLL(name)
	if err != nil {
		return nil, err
	}
	api := &conptyAPI{dll: name}
	for _, f := range []struct {
		proc **windows.Proc
		name string
	}{
		{&api.create, "CreatePseudoConsole"},
		{&api.resize, "ResizePseudoConsole"},
		{&api.close, "ClosePseudoConsole"},
	} {
		if *f.proc, err = dll.FindProc(f.name); err != nil {
			_ = dll.Release()
			return nil, err
		}
	}
	return api, nil
}

// ConptyDLL is the DLL ConPTY is loaded from, the path of conpty.dll or
// "kernel32.dll", "" when none could be loaded
func ConptyDLL() string {
	api, err := loadConpty()
	if err != nil {
		return ""
	}
	return api.dll
}

// coord packs a size as the COORD the ConPTY functions take by value
func coord(col, row int) uintptr {
	return uintptr(uint16(col)) | uintptr(uint16(row))<<16
}

type Pty struct {
	sized sizeGate

	// mu guards the fields below: the UI thread resizes and writes, the
	// terminal's goroutine starts the program
	mu       sync.Mutex
	row, col int
	in       *os.File
	out      *os.File
	api      *conptyAPI
	hpc      windows.Handle
	process  *os.Process

	closeOnce sync.Once
}

func New() *Pty {
	return &Pty{sized: newSizeGate()}
}

func (p *Pty) Resize(row, col int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.row == row && p.col == col {
		return
	}

	p.row, p.col = row, col
	p.sized.set(row, col)
	if p.hpc == 0 {
		return
	}

	_, _, _ = p.api.resize.Call(uintptr(p.hpc), coord(col, row))
}

// defaultProgram is the program when none is set
const defaultProgram = `C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe`

// RunCmd starts the program of the command line cmd, "" for the default
// program. A program that is not found falls back to the default, so a wrong
// setting does not close every new tab.
func (p *Pty) RunCmd(cmd string) error {
	p.sized.wait()
	p.mu.Lock()
	row, col := p.row, p.col
	p.mu.Unlock()

	// CreateProcess does not look in PATH, so the program gets its full path
	path, args, ok := FindProgram(cmd)
	if !ok {
		path, args, ok = FindProgram(defaultProgram)
		if !ok {
			return &exec.Error{Name: cmd, Err: exec.ErrNotFound}
		}
	}

	api, err := loadConpty()
	if err != nil {
		return err
	}

	// The pseudo console reads the input from one pipe and writes the output
	// to the other, it keeps its own handles of their ends
	var inRead, inWrite, outRead, outWrite windows.Handle
	if err := windows.CreatePipe(&inRead, &inWrite, nil, 0); err != nil {
		return err
	}
	if err := windows.CreatePipe(&outRead, &outWrite, nil, 0); err != nil {
		windows.CloseHandle(inRead)
		windows.CloseHandle(inWrite)
		return err
	}
	in := os.NewFile(uintptr(inWrite), "conpty-in")
	out := os.NewFile(uintptr(outRead), "conpty-out")

	var hpc windows.Handle
	r, _, _ := api.create.Call(coord(col, row), uintptr(inRead), uintptr(outWrite), 0, uintptr(unsafe.Pointer(&hpc)))
	windows.CloseHandle(inRead)
	windows.CloseHandle(outWrite)
	if r != 0 {
		in.Close()
		out.Close()
		return fmt.Errorf("CreatePseudoConsole of %s: HRESULT 0x%08x", api.dll, uint32(r))
	}

	pid, err := spawn(hpc, path, args)
	if err != nil {
		_, _, _ = api.close.Call(uintptr(hpc))
		in.Close()
		out.Close()
		return err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		_, _, _ = api.close.Call(uintptr(hpc))
		in.Close()
		out.Close()
		return err
	}

	p.mu.Lock()
	p.process = process
	p.api = api
	p.in = in
	p.out = out
	p.hpc = hpc

	// The size may have changed while it started
	_, _, _ = api.resize.Call(uintptr(hpc), coord(p.col, p.row))
	p.mu.Unlock()

	// ConPTY keeps its output pipe open after the shell exits (e.g. "exit"),
	// so Read never gets EOF and the pane never closes. Wait for the shell
	// and close the pseudo console, that ends the pipe and Read returns.
	go func() {
		_, _ = process.Wait()
		p.closePty()
	}()

	return nil
}

// spawn starts the program attached to the pseudo console, returns its pid
func spawn(hpc windows.Handle, path string, args []string) (int, error) {
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return 0, err
	}
	defer attrs.Delete()
	// The value of this attribute is the handle itself, not a pointer to it
	value := *(*unsafe.Pointer)(unsafe.Pointer(&hpc))
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, value, unsafe.Sizeof(hpc)); err != nil {
		return 0, err
	}

	// No std handles, so the program does not get those of YaTerm (e.g. when
	// it was started from a console) instead of the pseudo console
	si := windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = windows.STARTF_USESTDHANDLES

	appName, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	cmdLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{path}, args...)))
	if err != nil {
		return 0, err
	}
	var dir *uint16
	if wd, err := os.Getwd(); err == nil {
		dir, _ = windows.UTF16PtrFromString(wd)
	}

	// A nil environment gives the program the one of YaTerm
	var pi windows.ProcessInformation
	err = windows.CreateProcess(appName, cmdLine, nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT, nil, dir, &si.StartupInfo, &pi)
	if err != nil {
		return 0, err
	}
	windows.CloseHandle(pi.Thread)
	windows.CloseHandle(pi.Process)
	return int(pi.ProcessId), nil
}

// closePty closes the pseudo console once, both the exit of the shell and
// Close can do it. The output pipe ends when the pseudo console is gone, Read
// then gets EOF and closes it.
func (p *Pty) closePty() {
	p.closeOnce.Do(func() {
		_, _, _ = p.api.close.Call(uintptr(p.hpc))
		_ = p.in.Close()
	})
}

func (p *Pty) Read(buf []byte) (int, error) {
	n, err := p.out.Read(buf)
	if err == io.EOF {
		_ = p.out.Close()
	}
	return n, err
}

func (p *Pty) Write(bytes []byte) (int, error) {
	p.mu.Lock()
	in := p.in
	p.mu.Unlock()
	// Typing or pasting into a new pane before its shell has started
	if in == nil {
		return 0, errNotStarted
	}
	return in.Write(bytes)
}

func (p *Pty) Close() {
	p.mu.Lock()
	process := p.process
	p.mu.Unlock()
	if process == nil {
		return
	}

	_ = process.Kill()
	// Older ConPTYs block in ClosePseudoConsole until the output is read,
	// which may wait for this (UI) thread
	go p.closePty()
}
