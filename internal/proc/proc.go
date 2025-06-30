package proc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Proc struct {
	Name        string
	Exts        []string
	AutoStart   bool
	AutoRestart bool

	command  string
	dir      string
	shell    string
	onchange chan int

	logs   []string
	state  State
	wg     *sync.WaitGroup
	mu     *sync.Mutex
	cancel *context.CancelFunc
}

func New(
	name string,
	exts []string,
	autoStart bool,
	autoRestart bool,
	command string,
	dir string,
	shell string,
	onchange chan int,
) (*Proc, error) {
	// Make sure command exists in path
	cmd := strings.Split(command, " ")
	_, err := exec.LookPath(cmd[0])
	if err != nil {
		return nil, err
	}

	app := Proc{
		Name:        name,
		Exts:        exts,
		AutoStart:   autoStart,
		AutoRestart: autoRestart,

		command:  command,
		dir:      dir,
		shell:    shell,
		onchange: onchange,

		logs:  []string{},
		state: StateIdle,
		wg:    &sync.WaitGroup{},
		mu:    &sync.Mutex{},
	}

	return &app, nil
}

// Stops the process and waits for process to stop
func (a *Proc) Stop() error {
	cancel := a.getCancel()
	if cancel == nil {
		return errors.New("process has not started")
	}

	(*cancel)()
	a.Wait()

	return nil
}

// Starts the process
func (a *Proc) Start(ctx context.Context) error {
	if a.getCancel() != nil {
		return errors.New("process has already started")
	}

	nctx, cancel := context.WithCancel(ctx)
	a.setCancel(&cancel)
	defer a.setCancel(nil)

	return a.run(nctx)
}

// Waits for the process to stop
func (a *Proc) Wait() {
	a.wg.Wait()
}

// Runs the process
func (a *Proc) run(ctx context.Context) error {
	a.wg.Add(1)
	defer a.wg.Done()

	// Create exec.Cmd
	cmd := exec.Command(a.shell, "-c", a.command)
	if a.dir != "" {
		cmd.Dir = a.dir
	}

	// Create output pipe
	rpipe, wpipe, err := os.Pipe()
	if err != nil {
		return err
	}
	cmd.Stdout = wpipe
	cmd.Stderr = wpipe

	// Read from pipe
	scanner := bufio.NewScanner(rpipe)
	go func() {
		for scanner.Scan() {
			a.log("%s", scanner.Text())
		}
	}()

	// Start cmd
	err = cmd.Start()
	if err != nil {
		return err
	}
	a.setState(StateRunning)

	// Watch for stop
	go func() {
		<-ctx.Done()
		err := cmd.Process.Signal(os.Interrupt)
		if err != nil {
			cmd.Process.Kill()
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()
	if err != nil {
		a.setState(StateError)

		if exitError, ok := err.(*exec.ExitError); ok {
			a.log("exited with code %d", exitError.ExitCode())
		}
	} else {
		a.setState(StateSuccess)
	}

	// Check if we should restart
	if a.AutoRestart && ctx.Err() == nil {
		a.run(ctx)
	}

	return nil
}

func (a *Proc) log(msg string, ext ...any) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logs = append(a.logs, fmt.Sprintf(msg, ext...))
	a.update()
}

func (a *Proc) Logs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.logs
}

func (a *Proc) setState(state State) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.state = state
	a.update()
}

func (a *Proc) State() State {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.state
}

func (a *Proc) getCancel() *context.CancelFunc {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.cancel
}

func (a *Proc) setCancel(c *context.CancelFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cancel = c
}

func (a *Proc) update() {
	select {
	case a.onchange <- 1:
	default:
	}
}
