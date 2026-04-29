package x_rpc

import (
	"encoding/json"
	"github.com/FireManXiong/xsession/x/x_conn"
	"github.com/FireManXiong/xsession/x/x_log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Optioned interface {
	setAddr(addr string)
	setSvr(svr any)
	setPrefix(prefix string)
	setCallTimeout(duration time.Duration)
	setRetryDuration(duration time.Duration)
}

type optioned struct {
	addr          string
	svr           any
	prefix        string
	callTimeout   time.Duration
	retryDuration time.Duration
}

func (slf *optioned) setAddr(addr string) {
	slf.addr = addr
}

func (slf *optioned) setSvr(svr any) {
	slf.svr = svr
}

func (slf *optioned) setPrefix(prefix string) {
	slf.prefix = prefix
}

func (slf *optioned) setCallTimeout(duration time.Duration) {
	slf.callTimeout = duration
}

func (slf *optioned) setRetryDuration(duration time.Duration) {
	slf.retryDuration = duration
}

type Server struct {
	optioned
	methods sync.Map
}

func (slf *Server) Start(opts ...x_conn.Option) *Server {
	if slf.prefix == "" {
		slf.prefix = "Rpc"
	}
	if slf.addr == "" {
		panic("addr is empty")
	}
	if slf.svr == nil {
		panic("svr is nil")
	}
	slf.register()
	slf.listen(opts...)
	return slf
}

func (slf *Server) listen(opts ...x_conn.Option) {
	lis, err := net.Listen("tcp", slf.addr)
	if err != nil {
		panic(err)
	}
	handler := &x_conn.Handler{}
	handler.OnMessage = slf.onMessage
	for {
		conn, err := lis.Accept()
		if err != nil {
			x_log.Errorf("accept error:%v", err)
			continue
		}
		var options []x_conn.Option
		options = append(options, x_conn.WithConn(conn))
		options = append(options, x_conn.WithHandler(handler))
		options = append(options, opts...)
		x_conn.NewConn(options...).Start()
	}
}

type rpcMethod struct {
	method *reflect.Method
	parent *reflect.Value
}

type callReq struct {
	SeqId    uint64
	Method   string
	ReqBytes []byte
}

type callRsp struct {
	SeqId    uint64
	RspBytes []byte
	Err      string
}

func (slf *rpcMethod) beCall(reqBytes []byte) (rspBytes []byte, err error) {
	in := make([]reflect.Value, 3)
	in[0] = *slf.parent
	req := reflect.New(slf.method.Type.In(1).Elem()).Interface()
	if err = json.Unmarshal(reqBytes, req); err != nil {
		return nil, err
	}
	in[1] = reflect.ValueOf(req)
	rsp := reflect.New(slf.method.Type.In(2).Elem()).Interface()
	in[2] = reflect.ValueOf(rsp)
	out := slf.method.Func.Call(in)
	if !out[0].IsNil() {
		return nil, out[0].Interface().(error)
	}
	rspBytes, err = json.Marshal(rsp)
	return
}

func (slf *Server) register() {
	var a = slf.svr
	v := reflect.ValueOf(a)
	t := reflect.TypeOf(a)
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		if !strings.HasPrefix(method.Name, slf.prefix) {
			continue
		}
		if method.Type.NumIn() != 3 || method.Type.NumOut() != 1 {
			panic("x_rpc method must have 3 in param and 1 out param")
			return
		}
		if method.Type.In(1).Kind() != reflect.Ptr {
			panic("x_rpc method second param must be a pointer")
			return
		}
		if method.Type.In(2).Kind() != reflect.Ptr {
			panic("x_rpc method third param must be a pointer")
			return
		}
		if method.Type.NumOut() != 1 {
			panic("x_rpc method must have 1 out param")
			return
		}
		if !method.Type.Out(0).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			panic("x_rpc method out param must be error")
			return
		}
		slf.methods.Store(method.Name, &rpcMethod{
			method: &method,
			parent: &v,
		})
	}
}

func (slf *Server) onMessage(conn x_conn.Conn, id uint16, data []byte) {
	rpcReq := &callReq{}
	if err := json.Unmarshal(data, rpcReq); err != nil {
		x_log.Errorf("unmarshal x_rpc req error1: %s", err.Error())
		return
	}
	if info, ok := slf.methods.Load(rpcReq.Method); ok {
		method, ok := info.(*rpcMethod)
		if !ok {
			x_log.Errorf("x_rpc method convert error: %s", rpcReq.Method)
			return
		}
		rpcRsp := &callRsp{}
		rspBytes, err := method.beCall(rpcReq.ReqBytes)
		if err != nil {
			rpcRsp.Err = err.Error()
		}
		rpcRsp.SeqId = rpcReq.SeqId
		rpcRsp.RspBytes = rspBytes
		rspBytes, err = json.Marshal(rpcRsp)
		if err != nil {
			x_log.Errorf("marshal x_rpc rsp error: %s", err.Error())
			return
		}
		err = conn.Send(id, rspBytes)
		if err != nil {
			x_log.Errorf("send x_rpc rsp error: %s", err.Error())
		}
	} else {
		x_log.Errorf("x_rpc req not found: %s", rpcReq.Method)
	}
}

type Option func(s Optioned)

func NewServer(opts ...Option) *Server {
	s := &Server{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func WithAddr(addr string) Option {
	return func(s Optioned) {
		s.setAddr(addr)
	}
}

func WithSvr(svr any) Option {
	return func(s Optioned) {
		s.setSvr(svr)
	}
}

func WithPrefix(prefix string) Option {
	return func(s Optioned) {
		s.setPrefix(prefix)
	}
}

func WithCallTimeout(duration time.Duration) Option {
	return func(s Optioned) {
		s.setCallTimeout(duration)
	}
}
