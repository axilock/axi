package main

import (
	"fmt"

	"github.com/axilock/axi/internal/config"
	"github.com/axilock/axi/internal/fetcher"
	"google.golang.org/grpc"
)

type UpdateCheckCmd struct{}

func (c *UpdateCheckCmd) Run(conn *grpc.ClientConn, cfg *config.Config) error {
	response, err := fetcher.UpdateRequest(conn, cfg.Version, string(cfg.Environment))
	if err != nil {
		return err
	}

	if response.ToUpdate {
		fmt.Printf("Update available. %s => %s\n", cfg.Version, response.LatestClientver)
		fmt.Printf("Tip: you can enable autoupdate by setting ``autoupdate: true`` in ~/.axi/config.yaml\n")
	} else {
		fmt.Printf("Already on latest version %s\n", cfg.Version)
	}
	return nil
}
