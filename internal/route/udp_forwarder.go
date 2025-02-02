package route

import (
	"context"
	"fmt"
	"net"
	"sync"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/types"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	UDPForwarder struct {
		ctx       context.Context
		forwarder *net.UDPConn
		dstAddr   net.Addr
		connMap   F.Map[string, *UDPConn]
		mu        sync.Mutex
	}
	UDPConn struct {
		srcAddr *net.UDPAddr
		conn    net.Conn
		buf     *UDPBuf
	}
	UDPBuf struct {
		data, oob []byte
		n, oobn   int
	}
)

const udpConnBufferSize = 4096

func NewUDPForwarder(ctx context.Context, forwarder *net.UDPConn, dstAddr net.Addr) *UDPForwarder {
	return &UDPForwarder{
		ctx:       ctx,
		forwarder: forwarder,
		dstAddr:   dstAddr,
		connMap:   F.NewMapOf[string, *UDPConn](),
	}
}

func newUDPBuf() *UDPBuf {
	return &UDPBuf{
		data: make([]byte, udpConnBufferSize),
		oob:  make([]byte, udpConnBufferSize),
	}
}

func (conn *UDPConn) SrcAddrString() string {
	return conn.srcAddr.Network() + "://" + conn.srcAddr.String()
}

func (conn *UDPConn) DstAddrString() string {
	return conn.conn.RemoteAddr().Network() + "://" + conn.conn.RemoteAddr().String()
}

func (w *UDPForwarder) Addr() net.Addr {
	return w.forwarder.LocalAddr()
}

func (w *UDPForwarder) Accept() (types.StreamConn, error) {
	buf := newUDPBuf()
	addr, err := w.readFromListener(buf)
	if err != nil {
		return nil, err
	}
	return &UDPConn{
		srcAddr: addr,
		buf:     buf,
	}, nil
}

func (w *UDPForwarder) dialDst() (dstConn net.Conn, err error) {
	switch dstAddr := w.dstAddr.(type) {
	case *net.UDPAddr:
		var laddr *net.UDPAddr
		if dstAddr.IP.IsLoopback() {
			laddr, _ = net.ResolveUDPAddr(dstAddr.Network(), "127.0.0.1:")
		}
		dstConn, err = net.DialUDP(w.dstAddr.Network(), laddr, dstAddr)
	case *net.TCPAddr:
		dstConn, err = net.DialTCP(w.dstAddr.Network(), nil, dstAddr)
	default:
		err = fmt.Errorf("unsupported network %s", w.dstAddr.Network())
	}
	return
}

func (w *UDPForwarder) readFromListener(buf *UDPBuf) (srcAddr *net.UDPAddr, err error) {
	buf.n, buf.oobn, _, srcAddr, err = w.forwarder.ReadMsgUDP(buf.data, buf.oob)
	if err == nil {
		logger.Debug().Msgf("read from listener udp://%s success (n: %d, oobn: %d)", w.Addr().String(), buf.n, buf.oobn)
	}
	return
}

func (dst *UDPConn) read() (err error) {
	switch dstConn := dst.conn.(type) {
	case *net.UDPConn:
		dst.buf.n, dst.buf.oobn, _, _, err = dstConn.ReadMsgUDP(dst.buf.data, dst.buf.oob)
	default:
		dst.buf.n, err = dstConn.Read(dst.buf.data[:dst.buf.n])
		dst.buf.oobn = 0
	}
	if err == nil {
		logger.Debug().Msgf("read from dst %s success (n: %d, oobn: %d)", dst.DstAddrString(), dst.buf.n, dst.buf.oobn)
	}
	return
}

func (w *UDPForwarder) writeToSrc(srcAddr *net.UDPAddr, buf *UDPBuf) (err error) {
	buf.n, buf.oobn, err = w.forwarder.WriteMsgUDP(buf.data[:buf.n], buf.oob[:buf.oobn], srcAddr)
	if err == nil {
		logger.Debug().Msgf("write to src %s://%s success (n: %d, oobn: %d)", srcAddr.Network(), srcAddr.String(), buf.n, buf.oobn)
	}
	return
}

func (dst *UDPConn) write() (err error) {
	switch dstConn := dst.conn.(type) {
	case *net.UDPConn:
		dst.buf.n, dst.buf.oobn, err = dstConn.WriteMsgUDP(dst.buf.data[:dst.buf.n], dst.buf.oob[:dst.buf.oobn], nil)
		if err == nil {
			logger.Debug().Msgf("write to dst %s success (n: %d, oobn: %d)", dst.DstAddrString(), dst.buf.n, dst.buf.oobn)
		}
	default:
		_, err = dstConn.Write(dst.buf.data[:dst.buf.n])
		if err == nil {
			logger.Debug().Msgf("write to dst %s success (n: %d)", dst.DstAddrString(), dst.buf.n)
		}
	}

	return nil
}

func (w *UDPForwarder) Handle(streamConn types.StreamConn) error {
	conn, ok := streamConn.(*UDPConn)
	if !ok {
		panic("unexpected conn type")
	}
	key := conn.srcAddr.String()

	w.mu.Lock()
	dst, ok := w.connMap.Load(key)
	if !ok {
		var err error
		dst = conn
		dst.conn, err = w.dialDst()
		if err != nil {
			return err
		}
		if err := dst.write(); err != nil {
			dst.conn.Close()
			return err
		}
		w.connMap.Store(key, dst)
	} else {
		conn.conn = dst.conn
		if err := conn.write(); err != nil {
			w.connMap.Delete(key)
			dst.conn.Close()
			return err
		}
	}
	w.mu.Unlock()

	for {
		select {
		case <-w.ctx.Done():
			return nil
		default:
			if err := dst.read(); err != nil {
				w.connMap.Delete(key)
				dst.conn.Close()
				return err
			}

			if err := w.writeToSrc(dst.srcAddr, dst.buf); err != nil {
				return err
			}
		}
	}
}

func (w *UDPForwarder) Close() error {
	errs := E.NewBuilder("errors closing udp conn")
	w.mu.Lock()
	defer w.mu.Unlock()
	w.connMap.RangeAll(func(key string, conn *UDPConn) {
		errs.Add(conn.conn.Close())
	})
	w.connMap.Clear()
	errs.Add(w.forwarder.Close())
	return errs.Error()
}
