package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
)

type Runner interface {
	Startup(ctx context.Context) error
	Teardown(ctx context.Context) error
	Invoke(ctx context.Context, image string, commands, env []string) error
	MakeArgs(ctx context.Context, image string, commands, env []string) []string
}

//
//
//

type dockerRunner struct {
	repoDir            string
	firecrackerNumCPUs int
	firecrackerMemory  string
	commander          Commander
	root               string
}

var _ Runner = &dockerRunner{}

func (r *dockerRunner) Startup(ctx context.Context) error {
	return nil
}

func (r *dockerRunner) Teardown(ctx context.Context) error {
	return nil
}

func (r *dockerRunner) Invoke(ctx context.Context, image string, commands, env []string) error {
	args := r.MakeArgs(ctx, image, commands, env)
	if err := r.commander.Run(ctx, args[0], args[1:]...); err != nil {
		return errors.Wrap(err, "failed to SOMETHING")
	}

	return nil
}

func (r *dockerRunner) MakeArgs(ctx context.Context, image string, commands, env []string) []string {
	mountPoint := r.repoDir // TODO - might change if wrapped

	workingDirectory := "/data"
	if r.root != "" {
		workingDirectory = filepath.Join(workingDirectory, r.root)
	}

	args := []string{
		"docker", "run", "--rm",
		"--cpus", strconv.Itoa(r.firecrackerNumCPUs),
		"--memory", r.firecrackerMemory,
		"-v", fmt.Sprintf("%s:/data", mountPoint),
		"-w", workingDirectory,
	}
	args = append(args, image)
	for _, e := range env {
		args = append(args, "-e", e)
	}
	args = append(args, commands...)

	fmt.Printf("Args: %v\n", args)
	return args
}

//
//
//

type firecrackerRunner struct {
	repoDir            string
	firecrackerNumCPUs int
	firecrackerMemory  string
	commander          Commander
	firecrackerImage   string
	imageArchivePath   string
	name               string
	dockerRunner       *dockerRunner
}

var _ Runner = &firecrackerRunner{}

func (r *firecrackerRunner) Startup(ctx context.Context) error {
	mountPoint := "/repo-dir"

	images := map[string]string{
		// TODO
		// "install": index.InstallImage,
		// "indexer": index.Indexer,
		"src-cli": "sourcegraph/src-cli:latest",
	}

	copyfiles := []string{}
	for key, image := range images {
		tarfile := filepath.Join(r.imageArchivePath, fmt.Sprintf("%s.tar", key))
		copyfiles = append(copyfiles, "--copy-files", fmt.Sprintf("%s:%s", tarfile, fmt.Sprintf("/%s.tar", key)))

		if _, err := os.Stat(tarfile); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		if err := r.commander.Run(ctx, "docker", "pull", image); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to pull %s", image))
		}
		if err := r.commander.Run(ctx, "docker", "save", "-o", tarfile, image); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to save %s", image))
		}
	}

	args := []string{
		"ignite", "run",
		"--runtime", "docker",
		"--network-plugin", "docker-bridge",
		"--cpus", strconv.Itoa(r.firecrackerNumCPUs),
		"--memory", r.firecrackerMemory,
		"--copy-files", fmt.Sprintf("%s:%s", r.repoDir, mountPoint),
	}
	args = append(args, copyfiles...)
	args = append(
		args,
		"--ssh",
		"--name", r.name,
		sanitizeImage(r.firecrackerImage),
	)
	if err := r.commander.Run(ctx, args[0], args[1:]...); err != nil {
		return errors.Wrap(err, "failed to start firecracker vm")
	}

	for key, image := range images {
		if err := r.commander.Run(ctx, "ignite", "exec", r.name, "--", "docker", "load", "-i", fmt.Sprintf("/%s.tar", key)); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to load %s", image))
		}
	}

	return r.dockerRunner.Startup(ctx)
}

func (r *firecrackerRunner) Teardown(ctx context.Context) error {
	stopArgs := []string{
		"ignite", "stop",
		"--runtime", "docker",
		"--network-plugin", "docker-bridge",
		r.name,
	}
	if err := r.commander.Run(ctx, stopArgs[0], stopArgs[1:]...); err != nil {
		log15.Warn("failed to stop firecracker vm", "name", r.name, "err", err)
	}

	removeArgs := []string{
		"ignite", "rm", "-f",
		"--runtime", "docker",
		"--network-plugin", "docker-bridge",
		r.name,
	}
	if err := r.commander.Run(ctx, removeArgs[0], removeArgs[1:]...); err != nil {
		log15.Warn("failed to remove firecracker vm", "name", r.name, "err", err)
	}

	return r.dockerRunner.Teardown(ctx)
}

func (r *firecrackerRunner) Invoke(ctx context.Context, image string, commands, env []string) error {
	args := r.MakeArgs(ctx, image, commands, env)
	if err := r.commander.Run(ctx, args[0], args[1:]...); err != nil {
		return errors.Wrap(err, "failed to SOMETHING")
	}

	return nil
}

func (r *firecrackerRunner) MakeArgs(ctx context.Context, image string, commands, env []string) []string {
	return append([]string{"ignite", "exec", r.name, "--"}, r.dockerRunner.MakeArgs(ctx, image, commands, env)...)
}
