package main

import (
	"github.com/FireManXiong/xsession/x/x_conn"
	"github.com/FireManXiong/xsession/x/x_log"
	"github.com/FireManXiong/xsession/x/x_net"
)

func main() {
	// 简单echo服务：收到什么原样返回
	handler := &x_conn.Handler{
		OnMessage: func(conn x_conn.Conn, id uint16, data []byte) {
			_ = conn.Send(id, data)
		},
	}

	// 启动服务
	x_log.Infof("server start :8080")
	x_net.Listen(":8080", x_conn.WithHandler(handler))
}
