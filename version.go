package main

import (
	"fmt"

	"github.com/axilock/axi/internal/config"
)

type VersionCmd struct{}

func (c *VersionCmd) Run(cfg *config.Config) error {
	fmt.Println(cfg.Version)
	return nil
}
