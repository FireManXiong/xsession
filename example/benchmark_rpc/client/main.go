package main

import (
	"github.com/FireManXiong/xsession/x/x_rpc"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ClientCount = 1    // 并发客户端
	CallCount   = 7000 // 每个客户端发 6000 条
)

func main() {
	println("开始真实网络压测...")

	// 启动客户端
	var success atomic.Uint64
	var wg sync.WaitGroup
	wg.Add(ClientCount)

	start := time.Now()

	for i := 0; i < ClientCount; i++ {
		go func() {
			defer wg.Done()

			// 每个客户端一个连接
			cli := x_rpc.NewClient(x_rpc.WithAddr(":8080"))
			cli.Start()

			// 并发发送
			var reqWg sync.WaitGroup
			reqWg.Add(CallCount)

			for j := 0; j < CallCount; j++ {
				go func() {
					defer reqWg.Done()

					a := 10
					b := 0
					err := cli.Call("RpcAdd", &a, &b)
					if err == nil && b == 11 {
						success.Add(1)
					}
				}()
			}

			reqWg.Wait()
		}()
	}

	wg.Wait()

	cost := time.Since(start)
	total := success.Load()
	qps := float64(total) / cost.Seconds()

	println("\n========== 真实网络压测结果 ==========")
	println("总请求数:", total)
	println("耗时:", cost.String())
	println("真实 QPS:", int(qps))
}
