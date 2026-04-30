package x_conn

import (
	"encoding/binary"
	"fmt"
	"github.com/FireManXiong/xsession/x/x_log"
	"net"
	"sync/atomic"
	"time"
)

type Conn interface {
	Send(id uint16, data []byte) error
	Close()
	Start() Conn
	Closed() bool
}

const (
	headerSize = 4
	idSize     = 2
)

type Handler struct {
	OnOpen    func(conn Conn)
	OnClose   func(conn Conn)
	OnMessage func(conn Conn, id uint16, data []byte)
}

type conn struct {
	conn          net.Conn
	handler       *Handler
	bitCh         chan func()
	writeCh       chan []byte
	closedFlag    uint32
	remain        []byte
	maxRemain     int
	batchCount    int
	batchDuration time.Duration
}

type Option func(c *conn)

type Options []Option

func (opts *Options) Append(opt ...Option) *Options {
	*opts = append(*opts, opt...)
	return opts
}

func NewConn(opts ...Option) Conn {
	c := &conn{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithConn(con net.Conn) Option {
	return func(c *conn) {
		c.conn = con
	}
}
func WithHandler(handler *Handler) Option {
	return func(c *conn) {
		c.handler = handler
	}
}

func WithWriteChSize(size int) Option {
	return func(c *conn) {
		c.writeCh = make(chan []byte, size)
	}
}

func WithBitChSize(size int) Option {
	return func(c *conn) {
		c.bitCh = make(chan func(), size)
	}
}

func WithMaxRemainSize(size int) Option {
	return func(c *conn) {
		c.maxRemain = size
	}
}

func WithBatchCount(count int) Option {
	return func(c *conn) {
		c.batchCount = count
	}
}

func WithBatchDuration(duration time.Duration) Option {
	return func(c *conn) {
		c.batchDuration = duration
	}
}

func (slf *conn) Start() Conn {
	if slf.conn == nil {
		return nil
	}
	if slf.bitCh == nil {
		slf.bitCh = make(chan func(), 1024)
	}
	if slf.writeCh == nil {
		slf.writeCh = make(chan []byte, 1024)
	}
	if slf.maxRemain <= 0 {
		slf.maxRemain = 10240
	}
	if slf.batchCount <= 0 {
		slf.batchCount = 20
	}
	if slf.batchDuration <= 0 {
		slf.batchDuration = 20 * time.Millisecond
	}
	go slf.readLoop()
	go slf.writeLoop()
	go slf.bitLoop()
	return slf
}

func (slf *conn) readLoop() {
	func() {
		slf.bitCh <- func() {
			if slf.handler.OnOpen != nil {
				slf.handler.OnOpen(slf)
			}
		}
	}()
	buf := make([]byte, 1024)
	for {
		n, err := slf.conn.Read(buf)
		if err != nil {
			x_log.Errorf("conn read err: %v", err)
			break
		}
		if len(slf.remain)+n > slf.maxRemain {
			x_log.Errorf("conn read err: remain too large")
			break
		}
		slf.remain = append(slf.remain, buf[:n]...)
		for {
			if len(slf.remain) < headerSize {
				break
			}
			bodyLen := int(binary.BigEndian.Uint32(slf.remain[:headerSize]))
			totalLen := headerSize + bodyLen
			if len(slf.remain) < totalLen {
				break
			}
			body := slf.remain[headerSize:totalLen]
			slf.remain = slf.remain[totalLen:]
			id := binary.BigEndian.Uint16(body[:idSize])
			data := body[idSize:]
			if slf.handler.OnMessage != nil {
				slf.bitCh <- func() {
					slf.handler.OnMessage(slf, id, data)
				}
			}
		}
	}
	func() {
		slf.bitCh <- func() {
			if slf.handler.OnClose != nil {
				slf.handler.OnClose(slf)
			}
			slf.Close()
		}
	}()
}

func (slf *conn) writeLoop() {
	var timer = time.NewTimer(slf.batchDuration)
	defer timer.Stop()
	var batch [][]byte
	var buf []byte
	flush := func() {
		if len(batch) == 0 {
			return
		}
		for _, data := range batch {
			buf = append(buf, data...)
		}
		_, _ = slf.conn.Write(buf)
		batch = batch[:0]
		buf = buf[:0]
	}
	for {
		select {
		case data, ok := <-slf.writeCh:
			if !ok {
				flush()
				return
			}
			batch = append(batch, data)
			if len(batch) >= slf.batchCount {
				flush()
			} else {
				timer.Reset(slf.batchDuration)
			}
		case <-timer.C:
			flush()
		}
	}
}

func (slf *conn) bitLoop() {
	for f := range slf.bitCh {
		f()
	}
}

func (slf *conn) Send(id uint16, data []byte) error {
	if slf.closed() {
		return fmt.Errorf("connection is closed")
	}
	bodySize := len(data) + idSize
	pack := make([]byte, headerSize+bodySize)
	binary.BigEndian.PutUint32(pack[:headerSize], uint32(bodySize))
	binary.BigEndian.PutUint16(pack[headerSize:idSize+headerSize], id)
	copy(pack[idSize+headerSize:], data)
	select {
	case slf.writeCh <- pack:
		return nil
	default:
		return fmt.Errorf("write channel is full")
	}
}

func (slf *conn) Close() {
	if atomic.LoadUint32(&slf.closedFlag) == 1 {
		return
	}
	atomic.StoreUint32(&slf.closedFlag, 1)
	if slf.conn == nil {
		return
	}
	_ = slf.conn.Close()
	close(slf.writeCh)
	close(slf.bitCh)
}

func (slf *conn) closed() bool {
	return atomic.LoadUint32(&slf.closedFlag) == 1
}
func (slf *conn) Closed() bool {
	return slf.closed()
}
