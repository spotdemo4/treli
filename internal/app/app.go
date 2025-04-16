package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spotdemo4/treli/internal/util"
)

type App struct {
	Name  string
	Color string
	Dir   string
	Exts  []string
	State State

	OnStart  string
	OnChange string

	shell     string
	pause     bool
	prevState State
	msgs      chan Msg
	wg        *sync.WaitGroup
	stop      *util.Broadcast
	mu        *sync.Mutex
}

type Msg struct {
	AppName  string
	AppColor string
	AppState State

	Text  string
	Time  time.Time
	Key   string
	State State
}

func New(
	ctx context.Context,
	shell string,
	msgs chan Msg,
	name string,
	color string,
	dir string,
	exts []string,

	OnStart string,
	OnChange string,
) (*App, error) {
	if OnStart != "" {
		sp := strings.Split(OnStart, " ")
		_, err := exec.LookPath(sp[0])
		if err != nil {
			return nil, err
		}
	}
	if OnChange != "" {
		sp := strings.Split(OnChange, " ")
		_, err := exec.LookPath(sp[0])
		if err != nil {
			return nil, err
		}
	}

	app := App{
		Name:  name,
		Color: color,
		Dir:   dir,
		Exts:  exts,
		State: StateIdle,

		OnStart:  OnStart,
		OnChange: OnChange,

		shell: shell,
		pause: false,
		msgs:  msgs,
		wg:    &sync.WaitGroup{},
		stop:  util.NewBroadcast(),
		mu:    &sync.Mutex{},
	}

	go func() {
		<-ctx.Done()
		app.Stop()
	}()

	return &app, nil
}

func (a *App) Stop() {
	a.stop.Notify()
	a.Wait()
}

func (a *App) Wait() {
	a.mu.Lock()
	a.wg.Wait()
	a.mu.Unlock()
}

func (a *App) Pause() {
	if a.pause {
		a.pause = false
		a.State = a.prevState
	} else {
		a.pause = true
		a.prevState = a.State
		a.State = StatePause
	}
}

func (a *App) Run(command string) error {
	if a.pause {
		return nil
	}
	if command == "" {
		return nil
	}

	a.mu.Lock()
	a.wg.Add(1)
	defer a.wg.Done()
	a.mu.Unlock()

	a.State = StateLoading
	k := a.msg(command, &MsgOpts{
		State: StateLoading,
	})

	// Create exec.Cmd
	cmd := exec.Command(a.shell, "-c", command)
	if a.Dir != "" {
		cmd.Dir = a.Dir
	}

	// Check
	out, err := util.Run(cmd)
	if err != nil {
		a.State = StateError
		a.msg(
			fmt.Sprintf("could not run `%s`: %s", command, err.Error()),
			&MsgOpts{
				State: StateError,
				Key:   &k,
			},
		)

		return err
	}

	// Watch for output
	for {
		select {
		case line := <-out:
			switch line := line.(type) {
			case util.Stdout:
				a.msg(string(line), nil)

			case util.Stderr:
				a.msg(string(line), &MsgOpts{
					State: StateError,
				})

			case util.ExitCode:
				if line != 0 {
					a.State = StateError
					a.msg(command, &MsgOpts{
						State: StateError,
						Key:   &k,
					})

					return errors.New("exited with bad code")
				}

				a.State = StateSuccess
				a.msg(command, &MsgOpts{
					State: StateSuccess,
					Key:   &k,
				})
				return nil
			}

		case <-a.stop.Wait():
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				cmd.Process.Kill()
			}

			a.State = StateError
			a.msg(fmt.Sprintf("killed %s", command), &MsgOpts{
				State: StateError,
				Key:   &k,
			})
			return errors.New("killed")
		}
	}
}

type MsgOpts struct {
	Key   *string
	State State
}

func (a *App) msg(t string, opts *MsgOpts) string {
	var key string
	var state State

	if opts != nil {
		if opts.Key != nil {
			key = *opts.Key
		}
		if opts.State != StateIdle {
			state = opts.State
		}
	}
	if key == "" {
		key = uuid.NewString()
	}

	m := Msg{
		AppName:  a.Name,
		AppColor: a.Color,
		AppState: a.State,

		Text:  t,
		Time:  time.Now(),
		Key:   key,
		State: state,
	}
	a.msgs <- m

	return key
}
