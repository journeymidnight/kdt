package kdt

import (
	"fmt"
	"log"
	"io"
	"math/rand"
	"net"
	"os"
	"time"
	"net/http"
	"net/url"
	"path"

	"github.com/klauspost/compress/snappy"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
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

func handleClient(p1, p2 io.ReadWriteCloser) {
	log.Println("stream opened")
	defer log.Println("stream closed")
	defer p1.Close()
	defer p2.Close()

	// start tunnel
	p1die := make(chan struct{})
	go func() { io.Copy(p1, p2); close(p1die) }()

	p2die := make(chan struct{})
	go func() { io.Copy(p2, p1); close(p2die) }()

	// wait for tunnel termination
	select {
	case <-p1die:
	case <-p2die:
	}
}
func Exists(name string) bool {
  fi, err := os.Stat(name)
  log.Println("Exists", name, fi, err)
  return err == nil
}
func saveToFile(fout io.Writer, stream io.Reader) (written int64, err error) {
	log.Println("saveToFile", fout, stream)
	written = 0
	const bufsize = 256*1024
	totalsize := 0
	startTime := time.Now()
	writeTimes := 0
	readTimes := 0
	logfile, err := os.Create("receiver.log")
	buf := make([]byte, bufsize)
	for {
		bufStartTime := time.Now()
		nr, er := stream.Read(buf)
		readTimes += 1
		totalsize += nr
		bufUsedTime := time.Since(bufStartTime)
		const writetofile = false
		logfile.WriteString(fmt.Sprintf("Receive: times=%v BufUsedTime=%v size=%v,%v err=%v\n", readTimes, bufUsedTime, nr, totalsize, err))
		if nr > 0 {
			if writetofile {
				nw, ew := fout.Write(buf[0:nr])
				if nw > 0 {
					written += int64(nw)
				}
				if ew != nil {
					log.Println("write error", nw, ew)
					er = ew
					break
				}
				if nr != nw {
					log.Println("write diff error", nr, nw, ew)
					er = io.ErrShortWrite
					break
				}
			} else {
				written += int64(nr)
			}
			writeTimes += 1
			if writeTimes % 10 == 1 {
				// time.Sleep(1*time.Millisecond)
			}
			if writeTimes % 1000 == 1 {
				usedTime := float64(time.Since(startTime).Nanoseconds()) / float64(time.Second)
				speed := float64(written) / usedTime
				logfile.Sync()
				log.Printf("saveToFile progress size=%d speed=%0.5f\n", written, speed / 1000.0)
			}
		}
		if er == io.EOF {
			log.Println("read eof error", nr, er)
			break
		}
		if er != nil {
			log.Println("read non-eof error", nr, er)
			err = er
			break
		}
	}
	log.Println("saveToFile", fout, stream, written, err)
	return written, err
}

func runClient(c *cli.Context) error {
	config := ClientConfig{}
	err := config.Init(c)
	block := config.CreateBlockCrypt()
	if false {
	resp, err := http.Get(config.Url)
	log.Println("http.Get", config.Url, resp, err)
	defer resp.Body.Close()
	log.Println("start save file")
	fout, err := os.Create("testout.dat")
	log.Println("open file", fout, err)
	io.Copy(fout, resp.Body)
	fout.Close()
	log.Println("save file ok", err)
	return nil
	}
	if config.Url == "" {
		log.Println("url is empty", config.Output)
		return errors.New("url is empty")
	}
	u, err := url.Parse(config.Url)
	if err != nil {
		log.Println("failed to parse url", config.Url)
		return err
	}
	config.RemoteAddr = u.Host
	if config.Output == "" {
		config.Output = path.Base(u.Path)
	}
	if config.Output == "" {
		log.Println("output file path is empty", config.Output)
		return errors.New("output file path is empty")
	}
	if Exists(config.Output) {
		log.Println("output file exists", config.Url, config.Output)
		return errors.New("output file exists")
	}
	log.Println("output file does not exist", config.Url, config.Output)
	fout, err := os.OpenFile(config.Output, os.O_WRONLY | os.O_CREATE, 0666)
	if err != nil {
		log.Println("failed to open file", err)
		return err
	}
	defer fout.Close()
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		// DialContext: (&net.Dialer{
		// 	Timeout:   30 * time.Second,
		// 	KeepAlive: 30 * time.Second,
		// }).DialContext,
		Dial: func (network, addr string) (net.Conn, error) {
			return dialKcp(&config, block)
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{Transport: transport}
	log.Println("client:", client, config.Url)
	startTime := time.Now()
	resp, err := client.Get(config.Url)
	log.Println("resp:", resp, err)
	defer resp.Body.Close()
	io.Copy(fout, resp.Body)
	usedTime := time.Since(startTime)
	log.Println("time used", usedTime)
	return nil
}

func handleClientStream(p1 io.ReadWriteCloser, config *ServerConfig) {
	log.Println("handleClientStream", p1, config)
	const readfromfile = false
	if readfromfile {
		fin, err := os.Open("test.dat")
		if err != nil {
			log.Println("failed to open test.dat", err)
			p1.Close()
			return
		}
		go func() {
			log.Println("start print data from client")
			io.Copy(os.Stdout, p1)
			log.Println("data from client is ok")
		}()
		go func() {
			log.Println("start send file data to client")
			io.Copy(p1, fin)
			log.Println("all file data is sent to client")
			fin.Close()
			log.Println("file is closed")
			p1.Close()
			log.Println("stream to client is closed")
		}()
	} else {
		go func() {
			var totalsize int64 = 0
			buf := make([]byte, 2*1024)
			for i := 0; i < len(buf); i += 1 {
				buf[i] = byte(rand.Int())
			}
			logfile, err := os.Create("sender.log")
			if err != nil {
				log.Println("failed to create sender.log", err)
			}
			startTime := time.Now()
			writeTimes := 0
			lastEndTime := time.Now()
			var maxBufUsedTime, maxInterval, maxLoopTime time.Duration = 0, 0, 0
			var minBufUsedTime, minInterval, minLoopTime time.Duration = time.Hour * 24, time.Hour * 24, time.Hour * 24
			for totalsize < 600*1024*1024 {
				loopStartTime := time.Now()
				interval := time.Since(lastEndTime)
				if minInterval > interval {
					minInterval = interval
				}
				if maxInterval < interval {
					maxInterval = interval
				}
				bufStartTime := time.Now()
				nw, err := p1.Write(buf)
				bufUsedTime := time.Since(bufStartTime)
				if minBufUsedTime > bufUsedTime {
					minBufUsedTime = bufUsedTime
				}
				if maxBufUsedTime < bufUsedTime {
					maxBufUsedTime = bufUsedTime
				}
				usedTime := time.Since(startTime)
				if nw < len(buf) || err != nil {
					log.Println("p1.Write error", len(buf), nw, err, totalsize, writeTimes)
				}
				totalsize += int64(len(buf))
				writeTimes += 1
				if writeTimes % 10 == 1 {
					time.Sleep(100*time.Microsecond)
				}
				if writeTimes % 300 == 1 || bufUsedTime >= 30*time.Millisecond {
					logfile.Sync()
					log.Println("write size", writeTimes, totalsize, bufUsedTime, minBufUsedTime, maxBufUsedTime, usedTime, interval, minInterval, maxInterval)
				}
				loopTime := time.Since(loopStartTime)
				if minLoopTime > loopTime {
					minLoopTime = loopTime
				}
				if maxLoopTime < loopTime {
					maxLoopTime = loopTime
				}
				logfile.WriteString(fmt.Sprintf(
					"WriteString: LoopTime=%v,%v,%v BufUsedTime=%v,%v,%v Interval=%v,%v,%v UsedTime=%v totalsize=%v,%v\n",
					loopTime, minLoopTime, maxLoopTime, 
					bufUsedTime, minBufUsedTime, maxBufUsedTime,
					interval, minInterval, maxInterval,
					usedTime, totalsize, writeTimes))
				lastEndTime = time.Now()
			}
			log.Println("data is all sent to client", totalsize)
			p1.Close()
		}()
	}
}

// handle multiplex-ed connection
func handleMux(conn io.ReadWriteCloser, config *ServerConfig) {
	// stream multiplex
	smuxConfig := smux.DefaultConfig()
	smuxConfig.MaxReceiveBuffer = config.SockBuf
	mux, err := smux.Server(conn, smuxConfig)
	if err != nil {
		log.Println(err)
		return
	}
	defer mux.Close()
	for {
		p1, err := mux.AcceptStream()
		if err != nil {
			log.Println(err)
			return
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
		handleClientStream(p1, config)
	}
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
	conn.SetKeepAlive(config.KeepAlive)

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
		// handleClientStream(p1, config)
		return p1, nil
	// }
}

func (listener *KCPStreamListener) Close() error{
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
		log.Println("SetDSCP:", err)
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

func runServer(c *cli.Context) error {
	config := ServerConfig{}
	err := config.Init(c)
	block := config.CreateBlockCrypt()
	log.Println("config", c, config, block)

	
	lis, err := kcp.ListenWithOptions(config.Listen, block, config.DataShard, config.ParityShard)
	checkError(err)

	if err := lis.SetDSCP(config.DSCP); err != nil {
		log.Println("SetDSCP:", err)
	}
	if err := lis.SetReadBuffer(config.SockBuf); err != nil {
		log.Println("SetReadBuffer:", err)
	}
	if err := lis.SetWriteBuffer(config.SockBuf); err != nil {
		log.Println("SetWriteBuffer:", err)
	}
	// lis, err := net.Listen("tcp", config.Listen)
	log.Println("listen tcp", lis, err)
	slis := &KCPStreamListener{kcpListener : lis, config : &config}

	mux := http.NewServeMux()
	// mux.Handle("/", http.FileServer(http.Dir("./")))
	mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir("./"))))
	return http.Serve(slis, mux)
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

func createConn(config *ClientConfig, block kcp.BlockCrypt) (net.Conn, error) {
	smuxConfig := smux.DefaultConfig()
	smuxConfig.MaxReceiveBuffer = config.SockBuf

	log.Println("createConn", config.RemoteAddr)
	kcpconn, err := kcp.DialWithOptions(config.RemoteAddr, block, config.DataShard, config.ParityShard)
	// kcpconn, err := kcp.Dial(config.RemoteAddr)
	if err != nil {
		return nil, errors.Wrap(err, "createConn()")
	}
	log.Println("createConn dial", kcpconn, config.RemoteAddr, err)
	kcpconn.SetStreamMode(true)
	kcpconn.SetNoDelay(config.NoDelay, config.Interval, config.Resend, config.NoCongestion)
	kcpconn.SetWindowSize(config.SndWnd, config.RcvWnd)
	kcpconn.SetMtu(config.MTU)
	kcpconn.SetACKNoDelay(config.AckNodelay)
	kcpconn.SetKeepAlive(config.KeepAlive)

	if err := kcpconn.SetDSCP(config.DSCP); err != nil {
		log.Println("SetDSCP:", err)
	}
	if err := kcpconn.SetReadBuffer(config.SockBuf); err != nil {
		log.Println("SetReadBuffer:", err)
	}
	if err := kcpconn.SetWriteBuffer(config.SockBuf); err != nil {
		log.Println("SetWriteBuffer:", err)
	}

	// stream multiplex
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
	kcpconn.SetKeepAlive(config.KeepAlive)

	if err := kcpconn.SetDSCP(config.DSCP); err != nil {
		log.Println("SetDSCP:", err)
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
