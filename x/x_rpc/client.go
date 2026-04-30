package x_rpc

import (
	"errors"
	"github.com/FireManXiong/xsession/x/x_conn"
	"github.com/FireManXiong/xsession/x/x_json"
	"github.com/FireManXiong/xsession/x/x_log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type callInfo struct {
	seqId uint64
	done  chan *callInfo
	req   any
	rsp   any
}
type Client struct {
	optioned
	conn    x_conn.Conn
	seqId   uint64
	seqMap  sync.Map
	tryConn func()
}

func NewClient(opts ...Option) *Client {
	s := &Client{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (slf *Client) Start(opts ...x_conn.Option) *Client {
	if slf.addr == "" {
		panic("x_rpc client addr is empty")
	}
	if slf.callTimeout <= 0 {
		slf.callTimeout = 15 * time.Second
	}
	if slf.retryDuration <= 0 {
		slf.retryDuration = 5 * time.Second
	}
	slf.tryConn = func() {
		for {
			err := slf.connect(opts...)
			if err != nil {
				time.Sleep(slf.retryDuration)
			} else {
				break
			}
		}
	}
	slf.tryConn()
	return slf
}

func (slf *Client) connect(opts ...x_conn.Option) error {
	conn, err := net.Dial("tcp", slf.addr)
	if err != nil {
		x_log.Errorf("x_rpc client connect error:%v", err)
		return err
	}
	var options []x_conn.Option
	handler := &x_conn.Handler{}
	handler.OnMessage = slf.onMessage
	handler.OnClose = slf.onClose
	options = append(options, x_conn.WithConn(conn))
	options = append(options, x_conn.WithHandler(handler))
	options = append(options, opts...)
	slf.conn = x_conn.NewConn(options...).Start()
	return err
}

func (slf *Client) onMessage(conn x_conn.Conn, id uint16, data []byte) {
	slf.rpcRsp(data)
}

func (slf *Client) onClose(conn x_conn.Conn) {
	slf.tryConn()
}
func (slf *Client) rpcRsp(data []byte) {
	rpcRsp := &callRsp{}
	if err := x_json.Unmarshal(data, rpcRsp); err != nil {
		x_log.Errorf("unmarshal x_rpc rsp error1: %s", err.Error())
		return
	}
	if info, ok := slf.seqMap.Load(rpcRsp.SeqId); ok {
		cInfo := info.(*callInfo)
		err := x_json.Unmarshal(rpcRsp.RspBytes, cInfo.rsp)
		if err != nil {
			x_log.Errorf("unmarshal x_rpc rsp error2: %s", err.Error())
			return
		}
		select {
		case cInfo.done <- cInfo:
		default:
		}
	} else {
		x_log.Errorf("x_rpc rsp not found: %d", rpcRsp.SeqId)
	}
}

func (slf *Client) Call(method string, req any, rsp any) error {
	seqId := atomic.AddUint64(&slf.seqId, 1)
	info := &callInfo{
		seqId: seqId,
		done:  make(chan *callInfo, 1),
		req:   req,
		rsp:   rsp,
	}
	reqBytes, err := x_json.Marshal(req)
	if err != nil {
		return err
	}
	rpcReq := &callReq{
		SeqId:    seqId,
		Method:   method,
		ReqBytes: reqBytes,
	}
	callBytes, err := x_json.Marshal(rpcReq)
	if err != nil {
		return err
	}
	slf.seqMap.Store(seqId, info)
	defer func() {
		slf.seqMap.Delete(seqId)
	}()
	err = slf.conn.Send(1, callBytes)
	if err != nil {
		return err
	}
	t := time.NewTimer(slf.callTimeout)
	defer t.Stop()
	select {
	case <-t.C:
		return errors.New("x_rpc timeout")
	case <-info.done:
	}
	return nil
}
