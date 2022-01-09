package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/spf13/pflag"
)

type Config struct {
	file     string
	interval time.Duration
}

func main() {
	var cfg Config
	pflag.StringVarP(&cfg.file, "file", "f", "targets.json", "File to write targets to")
	pflag.DurationVarP(&cfg.interval, "interval", "i", 30*time.Second, "Interval to refresh targets at")
	pflag.Parse()

	if err := discover(cfg); err != nil {
		log.Fatalln(err)
	}
}

func discover(cfg Config) error {
	log.Println("Connecting to Docker API")
	ctx := context.Background()
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	var listOpts = types.ContainerListOptions{All: true}

	log.Printf("Writing to '%s' every %s", cfg.file, cfg.interval)
	t := time.Tick(cfg.interval)
	for ; true; <-t {
		containers, err := c.ContainerList(ctx, listOpts)
		if err != nil {
			return err
		}

		ch := make(chan Target, len(containers))
		for _, ctr := range containers {
			go target(ctx, ch, c, ctr.ID)
		}

		targets := make([]Target, 0, len(containers))
		for range containers {
			t := <-ch
			if t == nil {
				continue
			}
			targets = append(targets, t)
		}
		close(ch)

		data, err := json.MarshalIndent(targets, "", "  ")
		if err != nil {
			return err
		}

		if err := os.WriteFile(cfg.file, data, 0644); err != nil {
			return err
		}
	}

	return nil
}

const Prefix = "__meta_docker_container_"

const (
	MetaID   = Prefix + "id"
	MetaName = Prefix + "name"

	MetaStatus = Prefix + "status"
	MetaLabel  = Prefix + "label_"

	LabelPath = "__path__"
)

func target(ctx context.Context, r chan<- Target, c *client.Client, containerID string) {
	ctr, err := c.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Printf("Failed to inspect '%s'. Results may be incomplete: %s", containerID, err)
		r <- nil
		return
	}

	t := Target{
		MetaID:     ctr.ID,
		MetaName:   strings.TrimPrefix(ctr.Name, "/"),
		MetaStatus: ctr.State.Status,

		LabelPath: ctr.LogPath,
	}

	for k, v := range ctr.Config.Labels {
		k := MetaLabel + k
		k = strings.ReplaceAll(k, ".", "_")
		k = strings.ReplaceAll(k, "-", "_")

		t[k] = v
	}

	if t[MetaStatus] != "running" {
		r <- nil
		return
	}

	r <- t
	return
}

type Target map[string]string

func (t Target) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"targets": []string{t[MetaID]},
		"labels":  map[string]string(t),
	}
	return json.Marshal(m)
}
