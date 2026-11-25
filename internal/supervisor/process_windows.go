//go:build windows

package supervisor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe" //nolint:depguard // Win32 job-object and thread-entry APIs take raw struct pointers

	"golang.org/x/sys/windows"
)

var (
	errResumeBeforeAssignment = errors.New("zcode: resume Windows backend before job assignment")
	errNoResumableThread      = errors.New("zcode: suspended backend has no resumable thread")
)

// windowsProcessTree owns a Job Object containing the backend and every
// descendant it creates. KILL_ON_JOB_CLOSE is the final orphan-process
// backstop; TerminateJobObject provides the explicit staged kill.
type windowsProcessTree struct {
	mu       sync.Mutex
	job      windows.Handle
	process  *os.Process
	assigned bool
	resumed  bool
	closed   bool
}

//nolint:ireturn // the platform constructor returns the shared processTree interface
func newProcessTree() (processTree, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("zcode: create Windows job object: %w", err)
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE

	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job)

		return nil, fmt.Errorf("zcode: configure Windows job object: %w", err)
	}

	return &windowsProcessTree{job: job}, nil
}

// processID converts an os.Process PID to the DWORD the Windows APIs take.
func processID(proc *os.Process) uint32 {
	return uint32(proc.Pid) //nolint:gosec // Windows process IDs are DWORDs; os.Process.Pid always fits
}

func configureCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_SUSPENDED,
	}
}

func (t *windowsProcessTree) assign(proc *os.Process) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return os.ErrProcessDone
	}
	if t.assigned {
		return nil
	}
	if err := assignProcessToJob(t.job, proc); err != nil {
		return err
	}

	t.process = proc
	t.assigned = true

	return nil
}

func (t *windowsProcessTree) resume() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return os.ErrProcessDone
	}
	if t.resumed {
		return nil
	}
	if t.process == nil || !t.assigned {
		return errResumeBeforeAssignment
	}
	if err := resumeProcessThread(t.process); err != nil {
		return err
	}

	t.resumed = true

	return nil
}

func (t *windowsProcessTree) terminate() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	if t.process != nil && !t.assigned {
		if err := t.assignLocked(t.process); err != nil {
			return err
		}
	}
	if !t.assigned {
		return nil
	}
	if err := windows.TerminateJobObject(t.job, 1); err != nil {
		return fmt.Errorf("zcode: terminate Windows job object: %w", err)
	}

	return nil
}

func (t *windowsProcessTree) assignLocked(proc *os.Process) error {
	if err := assignProcessToJob(t.job, proc); err != nil {
		return err
	}

	t.process = proc
	t.assigned = true

	return nil
}

func resumeProcessThread(proc *os.Process) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, processID(proc))
	if err != nil {
		return fmt.Errorf("zcode: snapshot suspended backend pid %d: %w", proc.Pid, err)
	}
	defer func() { _ = windows.CloseHandle(snapshot) }()

	entry := windows.ThreadEntry32{}
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Thread32First(snapshot, &entry); err != nil {
		return fmt.Errorf("zcode: enumerate suspended backend pid %d threads: %w", proc.Pid, err)
	}

	for {
		if entry.OwnerProcessID == processID(proc) {
			thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
			if err != nil {
				return fmt.Errorf("zcode: open suspended backend pid %d thread %d: %w", proc.Pid, entry.ThreadID, err)
			}

			_, resumeErr := windows.ResumeThread(thread)
			_ = windows.CloseHandle(thread)
			if resumeErr != nil {
				return fmt.Errorf("zcode: resume backend pid %d thread %d: %w", proc.Pid, entry.ThreadID, resumeErr)
			}

			return nil
		}

		if err := windows.Thread32Next(snapshot, &entry); err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				break
			}

			return fmt.Errorf("zcode: enumerate suspended backend pid %d threads: %w", proc.Pid, err)
		}
	}

	return fmt.Errorf("%w: pid %d", errNoResumableThread, proc.Pid)
}

func assignProcessToJob(job windows.Handle, proc *os.Process) error {
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, processID(proc))
	if err != nil {
		return fmt.Errorf("zcode: open backend pid %d for Windows job assignment: %w", proc.Pid, err)
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	if err := windows.AssignProcessToJobObject(job, handle); err != nil {
		return fmt.Errorf("zcode: assign backend pid %d to Windows job object: %w", proc.Pid, err)
	}

	return nil
}

func (t *windowsProcessTree) close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return
	}

	t.closed = true
	_ = windows.CloseHandle(t.job)
}
