package app

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spotdemo4/treli/internal/util"
)

type App struct {
	Name        string
	Color       string
	Dir         string
	Exts        []string
	State       State
	InvertCheck bool

	checkstr string
	buildstr string
	startstr string

	msgs        chan Msg
	wg          *sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	ratelimiter *util.RateLimiter
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
	msgs chan Msg,
	name string,
	color string,
	dir string,
	exts []string,
	invertCheck bool,

	check string,
	build string,
	start string,
) *App {
	app := App{
		Name:        name,
		Color:       color,
		Dir:         dir,
		Exts:        exts,
		State:       StateIdle,
		InvertCheck: invertCheck,

		checkstr: check,
		buildstr: build,
		startstr: start,

		msgs:        msgs,
		wg:          &sync.WaitGroup{},
		ctx:         ctx,
		cancel:      nil,
		ratelimiter: util.NewRateLimiter(time.Second * 1),
	}

	return &app
}

func (a *App) Run() {
	// Rate limit calls
	ok := a.ratelimiter.Check()
	if !ok {
		return
	}
	a.ratelimiter.Wait()

	// Stop previously running instances
	a.Stop()
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel

	_, err := a.Check(ctx)
	if a.InvertCheck && err == nil {
		return
	}
	if !a.InvertCheck && err != nil {
		return
	}

	err = ctx.Err()
	if err != nil {
		return
	}

	_, err = a.Build(ctx)
	if err != nil {
		return
	}

	err = ctx.Err()
	if err != nil {
		return
	}

	_, err = a.Start(ctx)
}

func (a *App) Check(ctx context.Context) (bool, error) {
	return a.cmd(ctx, a.checkstr)
}

func (a *App) Build(ctx context.Context) (bool, error) {
	return a.cmd(ctx, a.buildstr)
}

func (a *App) Start(ctx context.Context) (bool, error) {
	return a.cmd(ctx, a.startstr)
}

func (a *App) Wait() {
	a.wg.Wait()
}

func (a *App) Stop() {
	if a.cancel != nil {
		a.cancel()
		a.wg.Wait()
	}
}

func (a *App) cmd(ctx context.Context, command string) (bool, error) {
	if command == "" {
		return false, nil
	}

	a.wg.Add(1)
	defer a.wg.Done()

	a.State = StateLoading
	k := a.msg(command, &MsgOpts{
		State: StateLoading,
	})

	// Create exec.Cmd
	var cmd *exec.Cmd
	c := strings.Split(command, " ")
	if len(c) == 1 {
		cmd = exec.Command(c[0])
	} else if len(c) > 1 {
		cmd = exec.Command(c[0], c[1:]...)
	} else {
		a.State = StateError
		a.msg("invalid command", &MsgOpts{
			State: StateError,
			Key:   &k,
		})
		return true, errors.New("invalid command")
	}
	if a.Dir != "" {
		cmd.Dir = a.Dir
	}

	// Check
	out, err := util.Run(ctx, cmd)
	if err != nil {
		a.State = StateError
		a.msg(
			fmt.Sprintf("could not run `%s`: %s", command, err.Error()),
			&MsgOpts{
				State: StateError,
				Key:   &k,
			},
		)

		return true, err
	}

	// Watch for output
	for line := range out {
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

				return true, errors.New("exited with bad code")
			}

			a.State = StateSuccess
			a.msg(command, &MsgOpts{
				State: StateSuccess,
				Key:   &k,
			})
		}
	}

	return true, nil
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
