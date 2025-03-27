package containers

import (
	"context"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"math/rand"
	"os"
	"strings"
)

type ExposedPort string

const (
	Remote    ExposedPort = "remote"
	InProcess ExposedPort = "in-process"
	Launchpad ExposedPort = "launchpad"
)

const (
	flagdPortRemote    = "8013"
	flagdPortInProcess = "8015"
	flagdPortLaunchpad = "8016"
)

type flagdConfig struct {
	version string
}

type FlagdContainer struct {
	testcontainers.Container
	portRemote    int
	portInProcess int
	portLaunchpad int
}

func (fc *FlagdContainer) GetPort(port ExposedPort) int {
	switch port {
	case Remote:
		return fc.portRemote
	case InProcess:
		return fc.portInProcess
	case Launchpad:
		return fc.portLaunchpad
	}
	return fc.portRemote
}

func NewFlagd(ctx context.Context) (*FlagdContainer, error) {
	version, err := readTestbedVersion()

	if err != nil {
		return nil, err
	}

	c := &flagdConfig{
		version: fmt.Sprintf("v%v", version),
	}

	return setupContainer(ctx, c)
}

func setupContainer(ctx context.Context, cfg *flagdConfig) (*FlagdContainer, error) {
	registry := "ghcr.io/open-feature"
	imgName := "flagd-testbed"

	fullImgName := registry + "/" + imgName + ":" + cfg.version

	req := testcontainers.ContainerRequest{
		Image: fullImgName,
		Name:  fmt.Sprintf("%s-%d", imgName, rand.Int()),
		ExposedPorts: []string{
			flagdPortRemote + "/tcp",
			flagdPortInProcess + "/tcp",
			flagdPortLaunchpad + "/tcp",
		},
		WaitingFor: wait.ForExposedPort(),
		Privileged: false,
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            true,
	})

	if err != nil {
		return nil, err
	}

	mappedPortRemote, errRemote := c.MappedPort(ctx, flagdPortRemote)
	mappedPortInProcess, errInProcess := c.MappedPort(ctx, flagdPortInProcess)
	mappedPortLaunchpad, errLaunchpad := c.MappedPort(ctx, flagdPortLaunchpad)
	if errRemote != nil || errInProcess != nil || errLaunchpad != nil {
		return nil, err
	}

	return &FlagdContainer{
		Container:     c,
		portRemote:    mappedPortRemote.Int(),
		portInProcess: mappedPortInProcess.Int(),
		portLaunchpad: mappedPortLaunchpad.Int(),
	}, nil
}

func readTestbedVersion() (string, error) {
	wd, _ := os.Getwd()
	fileName := "../flagd-testbed/version.txt"

	content, err := os.ReadFile(fmt.Sprintf("%s/%s", wd, fileName))
	if err != nil {
		fmt.Printf("Failed to read file: %s", fileName)
		return "", err
	}

	return strings.TrimSuffix(string(content), "\n"), nil
}
