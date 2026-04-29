package main

import (
	"github.com/FireManXiong/xsession/x/x_log"
	"github.com/FireManXiong/xsession/x/x_rpc"
	"time"
)

func main() {
	go func() {
		x_rpc.NewServer(
			x_rpc.WithAddr(":8080"),
			x_rpc.WithSvr(&RpcSvr{}),
		).Start()
	}()
	
	time.Sleep(time.Second)

	go func() {
		a := 10
		b := 20
		err := x_rpc.NewClient(x_rpc.WithAddr(":8080")).Start().Call("RpcAdd", &a, &b)
		if err != nil {
			x_log.Infof("err:%v", err)
		}
		x_log.Infof("a:%v,b:%v", a, b)
	}()
	select {}
}

type RpcSvr struct {
}

func (slf *RpcSvr) RpcAdd(a *int, b *int) error {
	*b = *a + 30
	return nil
}
