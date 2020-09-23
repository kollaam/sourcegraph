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
	images             map[string]string
}

var _ Runner = &firecrackerRunner{}

func NewFirecrackerRunner(
	repoDir string,
	firecrackerNumCPUs int,
	firecrackerMemory string,
	commander Commander,
	firecrackerImage string,
	imageArchivePath string,
	name string,
	installImage string,
	indexer string,
	dockerRunner Runner,
) Runner {
	images := map[string]string{
		"src-cli": srcCliImage,
		"indexer": indexer,
	}
	if installImage != "" {
		images["install"] = installImage
	}

	return &firecrackerRunner{
		repoDir:            repoDir,
		firecrackerNumCPUs: firecrackerNumCPUs,
		firecrackerMemory:  firecrackerMemory,
		commander:          commander,
		firecrackerImage:   firecrackerImage,
		imageArchivePath:   imageArchivePath,
		name:               name,
		installImage:       installImage,
		indexer:            indexer,
		dockerRunner:       dockerRunner,
		images:             images,
	}
}

const srcCliImage = "sourcegraph/src-cli:latest"

var firecrackerCommonFlags = []string{
	"--runtime", "docker",
	"--network-plugin", "docker-bridge",
}

func (r *firecrackerRunner) Startup(ctx context.Context) error {
	if err := r.ensureTarfilesOnHost(ctx, r.images); err != nil {
		return err
	}

	runArgs := concatAll(
		"ignite", "run", "--name", r.name, "--ssh",
		firecrackerCommonFlags,
		r.resourceFlags(),
		r.copyFilesFlags(ctx),
		sanitizeImage(r.firecrackerImage),
	)
	if err := r.commander.Run(ctx, runArgs...); err != nil {
		return errors.Wrap(err, "failed to start firecracker vm")
	}

	for _, key := range sortKeys(r.images) {
		copyArgs := concatAll(
			"ignite", "exec", r.name, "--",
			"docker", "load",
			"-i", r.tarfilePathOnVirtualMachine(key),
		)

		if err := r.commander.Run(ctx, copyArgs...); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to load %s", r.images[key]))
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
	return concatAll("ignite", "exec", r.name, "--", r.dockerRunner.MakeArgs(ctx, image, cs, mountPoint))
}

func (r *firecrackerRunner) ensureTarfilesOnHost(ctx context.Context, images map[string]string) error {
	for _, key := range sortKeys(images) {
		if ok, err := fileExists(r.tarfilePathOnHost(key)); err != nil {
			return err
		} else if ok {
			continue
		}

		if err := r.commander.Run(ctx, "docker", "pull", images[key]); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to pull %s", images[key]))
		}
		if err := r.commander.Run(ctx, "docker", "save", "-o", r.tarfilePathOnHost(key), images[key]); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to save %s", images[key]))
		}
	}

	return nil
}

func (r *firecrackerRunner) resourceFlags() []string {
	return []string{"--cpus", strconv.Itoa(r.firecrackerNumCPUs), "--memory", r.firecrackerMemory}
}

func (r *firecrackerRunner) copyFilesFlags(ctx context.Context) []string {
	var copyfiles []string
	for _, key := range sortKeys(r.images) {
		copyfiles = append(copyfiles, fmt.Sprintf("%s:%s", r.tarfilePathOnHost(key), r.tarfilePathOnVirtualMachine(key)))
	}

	return prefixKeys("--copy-files", append(
		[]string{fmt.Sprintf("%s:%s", r.repoDir, firecrackerMountPoint)},
		copyfiles...,
	))
}

func (r *firecrackerRunner) tarfilePathOnHost(key string) string {
	return filepath.Join(r.imageArchivePath, fmt.Sprintf("%s.tar", key))
}

func (r *firecrackerRunner) tarfilePathOnVirtualMachine(key string) string {
	return fmt.Sprintf("/%s.tar", key)
}
