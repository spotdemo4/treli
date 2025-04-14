package util

import (
	"bufio"
	"context"
	"os"
	"os/exec"
)

type Stdout string
type Stderr string
type ExitCode int

func Run(ctx context.Context, cmd *exec.Cmd) (chan any, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	c := make(chan any, 10)
	quit := make(chan int)

	go func() {
		scan := bufio.NewScanner(stdout)
		for scan.Scan() {
			c <- Stdout(scan.Text())
		}
	}()
	go func() {
		scan := bufio.NewScanner(stderr)
		for scan.Scan() {
			c <- Stderr(scan.Text())
		}
	}()

	go func() {
		select {
		case <-quit:
		case <-ctx.Done():
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				cmd.Process.Kill() // If the process is not responding to the interrupt signal, kill it
			}
		}
	}()

	go func() {
		defer close(c)
		defer close(quit)

		if err := cmd.Wait(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				c <- ExitCode(exitError.ExitCode())
			} else {
				c <- ExitCode(1)
			}
		} else {
			c <- ExitCode(0)
		}

		quit <- 0
	}()

	return c, nil
}
