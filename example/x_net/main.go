package main

import (
	"fmt"
	"github.com/FireManXiong/xsession/x/x_conn"
	"github.com/FireManXiong/xsession/x/x_log"
	"github.com/FireManXiong/xsession/x/x_net"
	"time"
)

func main() {
	go func() {
		handler := &x_conn.Handler{}
		handler.OnOpen = func(conn x_conn.Conn) {
			x_log.Infof("server conn open")
		}
		handler.OnMessage = func(conn x_conn.Conn, id uint16, data []byte) {
			x_log.Infof("server conn msg %d,%v", id, data)
		}
		handler.OnClose = func(conn x_conn.Conn) {
			x_log.Infof("server conn close")
		}
		x_net.Listen(":9090", x_conn.WithHandler(handler))
	}()
	time.Sleep(time.Second * 2)

	go func() {
		handler := &x_conn.Handler{}
		handler.OnOpen = func(conn x_conn.Conn) {
			x_log.Infof("client a conn open")
			for i := 0; i < 10; i++ {
				time.Sleep(time.Second)
				err := conn.Send(1, []byte("hello"))
				if err != nil {
					x_log.Errorf("send err %v", err)
				}
			}
		}
		handler.OnMessage = func(conn x_conn.Conn, id uint16, data []byte) {
			fmt.Println("client c message", id, data)
		}
		handler.OnClose = func(conn x_conn.Conn) {
			x_log.Infof("client conn close")
		}
		_, err := x_net.Dial(":9090", x_conn.WithHandler(handler))
		if err != nil {
			x_log.Errorf("connect err %v", err)
		}

	}()

	select {}

}
