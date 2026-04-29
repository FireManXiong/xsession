package x_net

import (
	"github.com/FireManXiong/xsession/x/x_conn"
	"net"
)

func Listen(addr string, opts ...x_conn.Option) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	for {
		c, err := lis.Accept()
		if err != nil {
			continue
		}
		var options []x_conn.Option
		options = append(options, x_conn.WithConn(c))
		options = append(options, opts...)
		x_conn.NewConn(options...).Start()
	}
}

func Dial(addr string, opts ...x_conn.Option) (x_conn.Conn, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	var options []x_conn.Option
	options = append(options, x_conn.WithConn(c))
	options = append(options, opts...)
	return x_conn.NewConn(options...).Start(), nil
}
