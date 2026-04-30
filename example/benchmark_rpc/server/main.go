package main

import (
	"github.com/FireManXiong/xsession/x/x_log"
	"github.com/FireManXiong/xsession/x/x_rpc"
)

type TestService struct{}

func (t *TestService) RpcAdd(a *int, b *int) error {
	*b = *a + 1
	return nil
}

func main() {
	server := x_rpc.NewServer(
		x_rpc.WithAddr(":8080"),
		x_rpc.WithSvr(&TestService{}),
	)
	x_log.Infof("server start :8080")
	server.Start()
}
