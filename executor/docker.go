package executor

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

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
	log.Println("meta:", string(job.Meta.Bytes))

	// TODO: Add docker registry auth
	t0 := time.Now()

	// TODO: 👇
	// reader, err := d.cli.ImagePull(context.Background(), "docker.io/library/alpine", types.ImagePullOptions{})
	// if err != nil {
	// 	log.Println(err)
	// 	job.Fail()
	// 	return
	// }
	log.Println("[PULLING]", time.Since(t0))

	// defer reader.Close()
	// io.Copy(os.Stdout, reader)

	t1 := time.Now()
	resp, err := d.cli.ContainerCreate(context.Background(), &container.Config{
		Image: "alpine",
		// Cmd:   []string{"echo", job.State, "&&", "sleep 1", "&&", "echo", "DONE 🎉"},
		Cmd: []string{"sh", "-c", fmt.Sprintf("echo '%s' && sleep 10 && echo 'DONE 🎉'", job.State)},
		Tty: false,
	}, nil, nil, nil, "")
	if err != nil {
		log.Println(err)
		job.Fail()
		return
	}
	log.Println("[CREATING]", time.Since(t1))

	t2 := time.Now()
	if err := d.cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Println(err)
		job.Fail()
		return
	}
	log.Println("[START]", time.Since(t2))

	out, err := d.cli.ContainerLogs(context.Background(), resp.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
	})
	if err != nil {
		log.Println("[ERROR]", err)
		job.Fail()
		return
	}

	go stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	t3 := time.Now()
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
		log.Println("[STATUS]", "changed to:", status.StatusCode)
	}
	log.Println("[WAIT]", time.Since(t3))

	t4 := time.Now()
	stopErr := d.cli.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if stopErr != nil {
		log.Println("[ERROR]", "failed to remove container", err)
	}
	log.Println("[CLEANUP]", time.Since(t4))

	job.Commit()
}
