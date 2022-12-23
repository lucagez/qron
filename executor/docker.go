package executor

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/lucagez/tinyq"
)

type DockerExecutor struct {
	cli *client.Client
}

func NewDockerExecutor() DockerExecutor {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	return DockerExecutor{
		cli: cli,
	}
}

type DockerConfig struct {
	Image string `json:"image,omitempty"`
}

func (d *DockerExecutor) Run(job tinyq.Job) {
	// TODO: Add docker registry auth
	reader, err := d.cli.ImagePull(context.Background(), "docker.io/library/alpine", types.ImagePullOptions{})
	if err != nil {
		log.Println(err)
		job.Fail()
		return
	}

	defer reader.Close()
	io.Copy(os.Stdout, reader)

	resp, err := d.cli.ContainerCreate(context.Background(), &container.Config{
		Image: "alpine",
		Cmd:   []string{"echo", job.State},
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		log.Println(err)
		job.Fail()
		return
	}

	if err := d.cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Println(err)
		job.Fail()
		return
	}

	statusCh, errCh := d.cli.ContainerWait(context.Background(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Println("[ERROR]", err)
			job.Fail()

			stopErr := d.cli.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{
				Force:         true,
				RemoveVolumes: true,
				RemoveLinks:   true,
			})
			if stopErr != nil {
				log.Println("[ERROR]", "failed to stop container", err)
			}

			return
		}
	case status := <-statusCh:
		log.Println("[STATUS]", "changed to:", status)
	}

	out, err := d.cli.ContainerLogs(context.Background(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		log.Println("[ERROR]", err)
		job.Fail()
		return
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	stopErr := d.cli.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if stopErr != nil {
		log.Println("[ERROR]", "failed to remove container", err)
	}
}
