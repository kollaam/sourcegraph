package indexer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/inconshreveable/log15"
)

type Runner interface {
	Startup(ctx context.Context) error
	Teardown(ctx context.Context) error
	Invoke(ctx context.Context, image string, cs *CommandSpec) error
	MakeArgs(ctx context.Context, image string, cs *CommandSpec, mountPoint string) []string
}

type CommandSpec struct {
	command []string
	env     map[string]string
}

func FromArgs(args []string) *CommandSpec {
	cs := &CommandSpec{
		env: map[string]string{},
	}

	return cs.AddArgs(args...)
}

func (cs *CommandSpec) AddArgs(args ...string) *CommandSpec {
	cs.command = append(cs.command, args...)
	return cs
}

func (cs *CommandSpec) AddFlag(name, value string) *CommandSpec {
	cs.command = append(cs.command, name, value)
	return cs
}

// TODO - maybe not this format?
func (cs *CommandSpec) AddEnv(name, value string) *CommandSpec {
	cs.env[name] = value
	return cs
}

//
//

// Commander abstracts running processes on the host machine.
type Commander interface {
	// Run invokes the given command on the host machine.
	Run(ctx context.Context, args ...string) error
}

// CommanderFunc is a function version of the Commander interface.
type CommanderFunc func(ctx context.Context, args ...string) error

// Run invokes the given command on the host machine. See the Commander interface for additional details.
func (f CommanderFunc) Run(ctx context.Context, args ...string) error {
	return f(ctx, args...)
}

// DefaultCommander is a commander that uses exec.Cmd to invoke commands on the host machine.
var DefaultCommander Commander = CommanderFunc(runCommand)

// runCommand invokes the given command on the host machine.
func runCommand(ctx context.Context, args ...string) error {
	command, args := args[0], args[1:]
	switch command {
	case "git":
	case "docker":
	case "ignite":

	default:
		return fmt.Errorf("illegal command '%s'", command)
	}

	// TODO
	// errors.Wrap(err, fmt.Sprintf("failed `%s`", strings.Join(args, " ")))

	cmd, stdout, stderr, err := makeCommand(ctx, command, args...)
	if err != nil {
		return err
	}

	log15.Debug(fmt.Sprintf("Running command: %s %s", command, strings.Join(args, " ")))

	wg := parallel(
		func() { processStream("stdout", stdout) },
		func() { processStream("stderr", stderr) },
	)

	if err := cmd.Start(); err != nil {
		return err
	}

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

// makeCommand returns a new exec.Cmd and pipes to its stdout/stderr streams.
func makeCommand(ctx context.Context, command string, args ...string) (_ *exec.Cmd, stdout, stderr io.Reader, err error) {
	cmd := exec.CommandContext(ctx, command, args...)

	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stderr, err = cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	return cmd, stdout, stderr, nil
}

// parallel runs each function in its own goroutine and returns a wait group that
// blocks until all invocations have returned.
func parallel(funcs ...func()) *sync.WaitGroup {
	var wg sync.WaitGroup

	for _, f := range funcs {
		wg.Add(1)

		go func(f func()) {
			defer wg.Done()
			f()
		}(f)
	}

	return &wg
}

// processStream prefixes and logs each line of the given reader.
func processStream(prefix string, r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		log15.Info(fmt.Sprintf("%s: %s", prefix, scanner.Text()))
	}
}

//
//
//

func concatAll(values ...interface{}) []string {
	var union []string
	for _, v := range values {
		switch val := v.(type) {
		case []string:
			union = append(union, val...)
		case string:
			union = append(union, val)
		}
	}

	return union
}
