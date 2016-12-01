package kdt

import (
	"crypto/sha1"
	"encoding/json"
	"log"
	"os"

	"github.com/urfave/cli"
	kcp "github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

// Config for client
type Config struct {
	Key          string `json:"key"`
	Crypt        string `json:"crypt"`
	Mode         string `json:"mode"`
	MTU          int    `json:"mtu"`
	SndWnd       int    `json:"sndwnd"`
	RcvWnd       int    `json:"rcvwnd"`
	DataShard    int    `json:"datashard"`
	ParityShard  int    `json:"parityshard"`
	DSCP         int    `json:"dscp"`
	Comp         bool   `json:"comp"`
	AckNodelay   bool   `json:"acknodelay"`
	NoDelay      int    `json:"nodelay"`
	Interval     int    `json:"interval"`
	Resend       int    `json:"resend"`
	NoCongestion int    `json:"nc"`
	SockBuf      int    `json:"sockbuf"`
	KeepAlive    int    `json:"keepalive"`
	Log          string `json:"log"`
	BufferSize   int    `json:"bufsize"`
}

// ClientConfig for client
type ClientConfig struct {
	Config
	RemoteAddr string `json:"remoteaddr"`
	Conn       int    `json:"conn"`
	AutoExpire int    `json:"autoexpire"`
	Url        string `json:"url"`
	Output     string `json:"output"`
	Input      string `json:"input"`
	Overwrite  bool   `json:"Overwrite"`
}

// ServerConfig for server
type ServerConfig struct {
	Config
	Listen string `json:"listen"`
	Root   string `json:"root"`
}

func parseJSONConfig(config *Config, path string) error {
	file, err := os.Open(path) // For read access.
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config)
}

func initConfig(config *Config, c *cli.Context) error {
	config.Key = c.String("key")
	config.Crypt = c.String("crypt")
	config.Mode = c.String("mode")
	config.MTU = c.Int("mtu")
	config.SndWnd = c.Int("sndwnd")
	config.RcvWnd = c.Int("rcvwnd")
	config.DataShard = c.Int("datashard")
	config.ParityShard = c.Int("parityshard")
	config.DSCP = c.Int("dscp")
	config.Comp = c.Bool("comp")
	config.AckNodelay = c.Bool("acknodelay")
	config.NoDelay = c.Int("nodelay")
	config.Interval = c.Int("interval")
	config.Resend = c.Int("resend")
	config.NoCongestion = c.Int("nc")
	config.SockBuf = c.Int("sockbuf")
	config.KeepAlive = c.Int("keepalive")
	config.Log = c.String("log")
	config.BufferSize = c.Int("log")

	if c.String("c") != "" {
		err := parseJSONConfig(config, c.String("c"))
		if printError(err) != nil {
            return err
        }
	}

	// log redirect
	if config.Log != "" {
		f, err := os.OpenFile(config.Log, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if printError(err) != nil {
            return err
        }
		defer f.Close()
		log.SetOutput(f)
	}

	switch config.Mode {
	case "normal":
		config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 0, 30, 2, 1
	case "fast":
		config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 0, 20, 2, 1
	case "fast2":
		config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 1, 20, 2, 1
	case "fast3":
		config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 1, 10, 2, 1
	}

	log.Println("version:", VERSION)

	log.Println("encryption:", config.Crypt)
	log.Println("nodelay parameters:", config.NoDelay, config.Interval, config.Resend, config.NoCongestion)
	log.Println("sndwnd:", config.SndWnd, "rcvwnd:", config.RcvWnd)
	log.Println("compression:", config.Comp)
	log.Println("mtu:", config.MTU)
	log.Println("datashard:", config.DataShard, "parityshard:", config.ParityShard)
	log.Println("acknodelay:", config.AckNodelay)
	log.Println("dscp:", config.DSCP)
	log.Println("sockbuf:", config.SockBuf)
	log.Println("keepalive:", config.KeepAlive)
	log.Println("bufsize:", config.BufferSize)
    return nil
}

func (config *Config) CreateBlockCrypt() kcp.BlockCrypt {
	pass := pbkdf2.Key([]byte(config.Key), []byte(SALT), 4096, 32, sha1.New)
	var block kcp.BlockCrypt
    var err error
	switch config.Crypt {
	case "tea":
		block, err = kcp.NewTEABlockCrypt(pass[:16])
	case "xor":
		block, err = kcp.NewSimpleXORBlockCrypt(pass)
	case "none":
		block, err = kcp.NewNoneBlockCrypt(pass)
	case "aes-128":
		block, err = kcp.NewAESBlockCrypt(pass[:16])
	case "aes-192":
		block, err = kcp.NewAESBlockCrypt(pass[:24])
	case "blowfish":
		block, err = kcp.NewBlowfishBlockCrypt(pass)
	case "twofish":
		block, err = kcp.NewTwofishBlockCrypt(pass)
	case "cast5":
		block, err = kcp.NewCast5BlockCrypt(pass[:16])
	case "3des":
		block, err = kcp.NewTripleDESBlockCrypt(pass[:24])
	case "xtea":
		block, err = kcp.NewXTEABlockCrypt(pass[:16])
	case "salsa20":
		block, err = kcp.NewSalsa20BlockCrypt(pass)
	default:
		config.Crypt = "aes"
		block, err = kcp.NewAESBlockCrypt(pass)
	}
	if err != nil {
		checkError(err)
	}
    return block
}

func (config *ClientConfig) Init(c *cli.Context) error {
	config.RemoteAddr = c.String("remoteaddr")
	config.Conn = c.Int("conn")
	config.AutoExpire = c.Int("autoexpire")
	config.Url = c.String("url")
	config.Overwrite = c.Bool("overwrite")
	log.Println("remote address:", config.RemoteAddr)
	log.Println("conn:", config.Conn)
	log.Println("autoexpire:", config.AutoExpire)
	log.Println("url:", config.Url)
	log.Println("overwrite:", config.Overwrite)
	return initConfig(&config.Config, c)
}

func (config *ServerConfig) Init(c *cli.Context) error {
	config.Listen = c.String("listen")
	config.Root = c.String("root")
	log.Println("listening on:", config.Listen)
	log.Println("root directory:", config.Root)
	return initConfig(&config.Config, c)
}
