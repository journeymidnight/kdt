package kdt

import (
	"log"
	"net"
	"os"
	"time"
	"github.com/klauspost/compress/snappy"
	"github.com/pkg/errors"
	kcp "github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

var (
	// VERSION is injected by buildflags
	VERSION = "SELFBUILD"
	// SALT is use for pbkdf2 key expansion
	SALT = "kcp-go"
)

type compStream struct {
	conn net.Conn
	w    *snappy.Writer
	r    *snappy.Reader
}

func (c *compStream) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *compStream) Write(p []byte) (n int, err error) {
	n, err = c.w.Write(p)
	err = c.w.Flush()
	return n, err
}

func (c *compStream) Close() error {
	return c.conn.Close()
}

func (c *compStream) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *compStream) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *compStream) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *compStream) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *compStream) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func newCompStream(conn net.Conn) *compStream {
	c := new(compStream)
	c.conn = conn
	c.w = snappy.NewBufferedWriter(conn)
	c.r = snappy.NewReader(conn)
	return c
}

type KCPStreamListener struct {
	kcpListener *kcp.Listener
	config *ServerConfig
}

func (listener *KCPStreamListener) Accept() (net.Conn, error) {
	conn, err := listener.kcpListener.AcceptKCP()
	if err != nil {
		return nil, err
	}
	log.Println("KCPStreamListener.Accept remote address:", conn.RemoteAddr())

	config := listener.config
	conn.SetStreamMode(true)
	conn.SetNoDelay(config.NoDelay, config.Interval, config.Resend, config.NoCongestion)
	conn.SetMtu(config.MTU)
	conn.SetWindowSize(config.SndWnd, config.RcvWnd)
	conn.SetACKNoDelay(config.AckNodelay)
	//conn.SetKeepAlive(config.KeepAlive)

	smuxConfig := smux.DefaultConfig()
	smuxConfig.MaxReceiveBuffer = config.SockBuf
	mux, err := smux.Server(conn, smuxConfig)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	// defer mux.Close()
	// for {
		p1, err := mux.AcceptStream()
		if err != nil {
			log.Println(err)
			return nil, err
		}
		var remoteAddr, localAddr string
		if p1.RemoteAddr() != nil {
			remoteAddr = p1.RemoteAddr().String()
		} else {
			remoteAddr = "none"
		}
		if p1.LocalAddr() != nil {
			localAddr = p1.LocalAddr().String()
		} else {
			localAddr = "none"
		}
		log.Println("AcceptStream", remoteAddr, localAddr)
		return p1, nil
	// }
}

func (listener *KCPStreamListener) Close() error{
	log.Println("KCPStreamListener.Close")
	return listener.kcpListener.Close()
}

func (listener *KCPStreamListener) Addr() net.Addr {
	return listener.kcpListener.Addr()
}

func CreateKCPStreamListener(config *ServerConfig, block kcp.BlockCrypt) (*KCPStreamListener, error) {
	lis, err := kcp.ListenWithOptions(config.Listen, block, config.DataShard, config.ParityShard)
	checkError(err)
    if err != nil {
        return nil, err
    }

	if err := lis.SetDSCP(config.DSCP); err != nil {
		log.Println("SetDSCP:", err, config.DSCP)
	}
	if err := lis.SetReadBuffer(config.SockBuf); err != nil {
		log.Println("SetReadBuffer:", err)
	}
	if err := lis.SetWriteBuffer(config.SockBuf); err != nil {
		log.Println("SetWriteBuffer:", err)
	}
	slis := &KCPStreamListener{kcpListener : lis, config : config}
    return slis, nil
}

func checkError(err error) {
	if err != nil {
		log.Println("checkError", err)
		os.Exit(-1)
	}
}

func printError(err error) error {
	if err != nil {
		log.Println("checkError", err)
		return err
	}
    return nil
}

func dialKcp(config *ClientConfig, block kcp.BlockCrypt) (net.Conn, error) {
	log.Println("dialKcp", config.RemoteAddr)
	kcpconn, err := kcp.DialWithOptions(config.RemoteAddr, block, config.DataShard, config.ParityShard)
	if err != nil {
		return nil, errors.Wrap(err, "createConn()")
	}
	kcpconn.SetStreamMode(true)
	kcpconn.SetNoDelay(config.NoDelay, config.Interval, config.Resend, config.NoCongestion)
	kcpconn.SetWindowSize(config.SndWnd, config.RcvWnd)
	kcpconn.SetMtu(config.MTU)
	kcpconn.SetACKNoDelay(config.AckNodelay)
	//kcpconn.SetKeepAlive(config.KeepAlive)

	if err := kcpconn.SetDSCP(config.DSCP); err != nil {
		log.Println("SetDSCP:", err, config.DSCP)
	}
	if err := kcpconn.SetReadBuffer(config.SockBuf); err != nil {
		log.Println("SetReadBuffer:", err)
	}
	if err := kcpconn.SetWriteBuffer(config.SockBuf); err != nil {
		log.Println("SetWriteBuffer:", err)
	}
	// stream multiplex
	smuxConfig := smux.DefaultConfig()
	smuxConfig.MaxReceiveBuffer = config.SockBuf
	var session *smux.Session
	if config.Comp {
		session, err = smux.Client(newCompStream(kcpconn), smuxConfig)
	} else {
		session, err = smux.Client(kcpconn, smuxConfig)
	}
	if err != nil {
		return nil, errors.Wrap(err, "createConn()")
	}
	stream, err := session.OpenStream()
	log.Println("OpenStream", stream, err)
	return stream, nil
}
