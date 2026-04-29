# xsession
轻量级高性能 TCP 长连接框架，内置 JSON-RPC，极简架构、无第三方依赖，
专为游戏服务、内网长连接、自定义 TCP 协议场景设计。

## 🔖 项目介绍
- 底层基于原生 TCP 封装，完美解决 **粘包/拆包**
- 三协程模型：读循环 / 批量写循环 / 串行业务循环
- 批量合并发包，大幅减少系统调用，降低 GC 压力
- 原子状态管理，连接安全关闭，杜绝协程、内存泄漏
- 反射实现轻量 JSON-RPC，低侵入、自动方法注册
- 客户端自动重连 + RPC 调用超时
- 所有消息回调串行执行，天然无锁、并发安全

## 📁 目录结构

    xsession/
    ├── x/
    │ ├── x_conn # TCP 底层连接核心
    │ ├── x_log # 日志工具
    │ ├── x_net # 网络字节序工具
    │ └── x_rpc # JSON-RPC 客户端 / 服务端
    ├── example/ # 完整使用示例
    └── go.mod

## 🧾 协议格式
采用 **大端序** 固定包头：[4Byte 整体长度] [2Byte MsgID] [Body 数据]


## 🚀 快速使用

```
go get github.com/FireManXiong/xsession 
```

### x_net 客户端 / 服务端

```go
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

```

### x_rpc 客户端 / 服务端
```go
package main

import (
	"github.com/FireManXiong/xsession/x/x_log"
	"github.com/FireManXiong/xsession/x_xrpc"
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
	client := x_rpc.NewClient(x_rpc.WithAddr(":8080")).Start()

	a := 10
	b := 20
	err := client.Call("RpcAdd", &a, &b)
	if err != nil {
		x_log.Infof("rpc err: %v", err)
	}
	x_log.Infof("a:%d, b:%d", a, b)

	select {}
}

type RpcSvr struct{}
func (slf *RpcSvr) RpcAdd(a *int, b *int) error {
	*b = *a + 30
	return nil
}

```

## ✅ RPC 方法规范
    方法默认前缀：Rpc
    入参：(结构体接收器, *请求体, *响应体)
    返回值：固定 error
    参数必须为指针，支持服务端修改后回传客户端
## ⚙️ 配置选项
    x_conn 连接配置
    WithWriteChSize 发送队列缓冲大小
    WithBitChSize 业务串行任务队列大小
    WithMaxRemainSize 最大拆包缓存上限
    WithBatchCount 批量发包合并数量
    WithBatchDuration 批量发包等待时长
## x_rpc 配置
    WithAddr 监听 / 连接地址
    WithSvr 绑定业务服务实例
    WithPrefix 自定义 RPC 方法前缀
    WithCallTimeout RPC 调用超时
    WithRetryDuration 客户端重连间隔
## 🛡 并发安全
    业务逻辑单协程串行执行，免锁开发
    连接状态使用 atomic 原子操作
    RPC 请求通过 sync.Map 安全管理
    断开 / 超时 / 异常全部资源回收，无泄漏
## 🎯 适用场景
    游戏长连接网关、逻辑服务
    内网轻量 RPC 服务
    物联网 TCP 长连接设备
    自定义私有协议长连接开发
## 📌 后续规划

