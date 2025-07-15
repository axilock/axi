package main

import (
	"github.com/axilock/axi/internal/auth"
	"github.com/axilock/axi/internal/config"
	"github.com/axilock/axi/internal/filesio"
	"google.golang.org/grpc"
)

type AuthCmd struct{}

func (a *AuthCmd) Run(cfg *config.Config, grpcConn *grpc.ClientConn) error {
	key, err := auth.Login(grpcConn, cfg.BackendUrl)
	if err != nil {
		return err
	}

	afs := filesio.AxiFS{Home: cfg.Home()}
	if err := afs.WriteAPIKey(key); err != nil {
		return err
	}
	return nil
}
