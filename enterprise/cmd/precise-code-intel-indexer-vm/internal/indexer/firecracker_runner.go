package indexer

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
)

const firecrackerMountPoint = "/repo-dir"

type firecrackerRunner struct {
	repoDir            string
	firecrackerNumCPUs int
	firecrackerMemory  string
	commander          Commander
	firecrackerImage   string
	imageArchivePath   string
	name               string
	installImage       string
	indexer            string
	dockerRunner       Runner
}

var _ Runner = &firecrackerRunner{}

// TODO - ctor

const srcCliImage = "sourcegraph/src-cli:latest"

var firecrackerCommonFlags = []string{
	"--runtime", "docker",
	"--network-plugin", "docker-bridge",
}

func (r *firecrackerRunner) Startup(ctx context.Context) error {
	images := r.images()

	if err := r.ensureTars(ctx, images); err != nil {
		return err
	}

	runArgs := concatAll(
		"ignite", "run", "--name", r.name, "--ssh",
		firecrackerCommonFlags,
		r.resourceFlags(),
		r.copyFilesFlags(ctx, images),
		sanitizeImage(r.firecrackerImage),
	)
	if err := r.commander.Run(ctx, runArgs...); err != nil {
		return errors.Wrap(err, "failed to start firecracker vm")
	}

	for _, key := range sortKeys(images) {
		if err := r.commander.Run(ctx, r.prefix(concatAll("docker", "load", "-i", r.tarfileVirtualMachine(key)))...); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to load %s", images[key]))
		}
	}

	return r.dockerRunner.Startup(ctx)
}

func (r *firecrackerRunner) Teardown(ctx context.Context) error {
	if err := r.commander.Run(ctx, concatAll("ignite", "stop", firecrackerCommonFlags, r.name)...); err != nil {
		log15.Warn("failed to stop firecracker vm", "name", r.name, "err", err)
	}

	if err := r.commander.Run(ctx, concatAll("ignite", "rm", "-f", firecrackerCommonFlags, r.name)...); err != nil {
		log15.Warn("failed to remove firecracker vm", "name", r.name, "err", err)
	}

	return r.dockerRunner.Teardown(ctx)
}

func (r *firecrackerRunner) Invoke(ctx context.Context, image string, cs *CommandSpec) error {
	return r.commander.Run(ctx, r.MakeArgs(ctx, image, cs, firecrackerMountPoint)...)
}

func (r *firecrackerRunner) MakeArgs(ctx context.Context, image string, cs *CommandSpec, mountPoint string) []string {
	return r.prefix(r.dockerRunner.MakeArgs(ctx, image, cs, mountPoint))
}

func (r *firecrackerRunner) images() map[string]string {
	images := map[string]string{
		"src-cli": srcCliImage,
		"indexer": r.indexer,
	}
	if r.installImage != "" {
		images["install"] = r.installImage
	}

	return images
}

// TODO - rename
func (r *firecrackerRunner) ensureTars(ctx context.Context, images map[string]string) error {
	for _, key := range sortKeys(images) {
		if ok, err := fileExists(r.tarfileHost(key)); err != nil {
			return err
		} else if ok {
			continue
		}

		if err := r.commander.Run(ctx, "docker", "pull", images[key]); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to pull %s", images[key]))
		}
		if err := r.commander.Run(ctx, "docker", "save", "-o", r.tarfileHost(key), images[key]); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to save %s", images[key]))
		}
	}

	return nil
}

func (r *firecrackerRunner) copyFilesFlags(ctx context.Context, images map[string]string) []string {
	var copyfiles []string
	for _, key := range sortKeys(images) {
		copyfiles = append(copyfiles, fmt.Sprintf("%s:%s", r.tarfileHost(key), r.tarfileVirtualMachine(key)))
	}

	return prefixKeys("--copy-files", append(
		[]string{fmt.Sprintf("%s:%s", r.repoDir, firecrackerMountPoint)},
		copyfiles...,
	))
}

func (r *firecrackerRunner) resourceFlags() []string {
	return []string{"--cpus", strconv.Itoa(r.firecrackerNumCPUs), "--memory", r.firecrackerMemory}
}

func (r *firecrackerRunner) prefix(args []string) []string {
	return concatAll("ignite", "exec", r.name, "--", args)
}

func (r *firecrackerRunner) tarfileHost(key string) string {
	return filepath.Join(r.imageArchivePath, fmt.Sprintf("%s.tar", key))
}

func (r *firecrackerRunner) tarfileVirtualMachine(key string) string {
	return fmt.Sprintf("/%s.tar", key)
}
