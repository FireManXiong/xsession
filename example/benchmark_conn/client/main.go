package main

import (
	"github.com/FireManXiong/xsession/x/x_conn"
	"github.com/FireManXiong/xsession/x/x_log"
	"github.com/FireManXiong/xsession/x/x_net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	TestDuration = 10 * time.Second
	MsgID        = 1001
	ClientNum    = 200 // 1=单连接, 100=多连接
)

var (
	sendTotal atomic.Uint64
	recvTotal atomic.Uint64
)

func main() {
	x_log.Infof("========== 通用压测 ==========")
	x_log.Infof("连接数量: %d", ClientNum)
	x_log.Infof("压测时长: %v", TestDuration)

	var conns []x_conn.Conn
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(ClientNum)

	for i := 0; i < ClientNum; i++ {
		go func() {
			defer wg.Done()
			handler := &x_conn.Handler{
				OnMessage: func(c x_conn.Conn, id uint16, data []byte) {
					recvTotal.Add(1)
				},
			}
			conn, err := x_net.Dial("127.0.0.1:8080", x_conn.WithHandler(handler), x_conn.WithBatchCount(50))
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}()
	}
	wg.Wait()

	x_log.Infof("连接建立完成: %d 个", len(conns))
	x_log.Infof("开始压测...")

	start := time.Now()
	endTime := start.Add(TestDuration)

	var sendWg sync.WaitGroup
	for _, conn := range conns {
		sendWg.Add(1)
		go func(c x_conn.Conn) {
			defer sendWg.Done()
			msg := []byte("ping")

			// 🔥 关键修复：控制发送速度，避免缓冲区溢出
			for time.Now().Before(endTime) {
				err := c.Send(MsgID, msg)
				if err == nil {
					sendTotal.Add(1)
				}
				// 微小停顿，让收发平衡（非常关键！）
				time.Sleep(500 * time.Nanosecond)
			}
		}(conn)
	}
	sendWg.Wait()

	time.Sleep(300 * time.Millisecond)

	elapsed := time.Since(start)
	send := sendTotal.Load()
	recv := recvTotal.Load()
	qps := float64(recv) / elapsed.Seconds()

	x_log.Infof("========== 压测完成 ==========")
	x_log.Infof("总发送: %d", send)
	x_log.Infof("总接收: %d", recv)
	x_log.Infof("总耗时: %v", elapsed)
	x_log.Infof("✅ 真实稳定 QPS: %.2f", qps)

	select {}
}
