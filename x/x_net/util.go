package x_net

import (
	"github.com/FireManXiong/xsession/x/x_conn"
	"net"
)

func Listen(addr string, opts ...x_conn.Option) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	for {
		c, err := lis.Accept()
		if err != nil {
			continue
		}
		options := x_conn.Options{}
		options.Append(x_conn.WithConn(c)).Append(opts...)
		x_conn.NewConn(options...).Start()
	}
}

func Dial(addr string, opts ...x_conn.Option) (x_conn.Conn, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	options := x_conn.Options{}
	options.Append(x_conn.WithConn(c)).Append(opts...)
	return x_conn.NewConn(options...).Start(), nil
}
