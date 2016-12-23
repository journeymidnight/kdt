package kdt

import (
	"crypto/sha1"
	"encoding/json"
	"log"
	"os"
	"flag"
	"strconv"

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
	TransferID   string
	ConfigFile   string
}

// ClientConfig for client
type ClientConfig struct {
	Config
	RemoteAddr 	string
	Conn       	int    `json:"conn"`
	AutoExpire  int    `json:"autoexpire"`
	Directory	string
	Destination	string
	StartPort   int
}

// ServerConfig for server
type ServerConfig struct {
	Config
	Listen 		string
	Root   		string
	StartPort   int
}

func createConfig() *Config {
	return &Config{
		Key: "it's a lecloudkcp secrect",
		Crypt: "none",
		Mode: "fast",
		MTU: 1350,
		DataShard: 10,
		ParityShard: 3,
		SndWnd: 1024,
		RcvWnd: 1024,
		DSCP: 0,
		Comp: false,
		AckNodelay: false,
		NoDelay: 0,
		Interval: 40,
		Resend: 0,
		NoCongestion: 0,
		SockBuf: 4194304,
		KeepAlive: 10,
		Log: "",
		BufferSize: 4096,
		ConfigFile: "",
	}
}

func CreateClientConfig() *ClientConfig {
	config := createConfig()
	return &ClientConfig{
		Config: *config,
		RemoteAddr: "",
		Conn: 1,
		AutoExpire: 0,
		Directory: "",
		StartPort: 8223,
	}
}

func CreateServerConfig() *ServerConfig {
	config := createConfig()
	return &ServerConfig{
		Config: *config,
		Listen: "",
		StartPort: 8223,
		Root: ".",
	}
}

func parseJSONConfig(config *Config, path string) error {
	file, err := os.Open(path) // For read access.
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config)
}

func (config *Config) ReviseConfig() {
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

}

func (config *ServerConfig) Revise() {
	config.Config.ReviseConfig()
}

func (config *ClientConfig) Revise() {
	config.Config.ReviseConfig()
	if config.RemoteAddr == "" && config.Destination != "" {
		config.RemoteAddr = config.Destination + ":" + strconv.Itoa(config.StartPort)
	}
}

func initConfig(config *Config, c *cli.Context) error {
	config.Key = c.String("key")
	config.Crypt = c.String("crypt")
	config.Mode = c.String("mode")
	config.MTU = c.Int("mtu")
	config.DataShard = c.Int("datashard")
	config.ParityShard = c.Int("parityshard")
	config.SndWnd = c.Int("sndwnd")
	config.RcvWnd = c.Int("rcvwnd")
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
	config.BufferSize = c.Int("bufsize")
	config.TransferID = c.String("transfer_id")

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
	//log.Println("version:", VERSION)
    return nil
}

func (config *Config) logConfig() {
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
	err := initConfig(&config.Config, c)
	config.Revise()
	config.Log()
	return err
}

func (config *ServerConfig) Init(c *cli.Context) error {
	config.Listen = c.String("listen")
	config.Root = c.String("root")
	err := initConfig(&config.Config, c)
	config.Revise()
	config.Log()
	return err
}

func (config *ClientConfig) Log() {
	config.Config.logConfig()
	log.Println("remote address:", config.RemoteAddr)
	log.Println("conn:", config.Conn)
	log.Println("autoexpire:", config.AutoExpire)
}

func (config *ServerConfig) Log() {
	config.Config.logConfig()
	log.Println("listening on:", config.Listen)
	log.Println("root directory:", config.Root)
}

func (config *Config) createCommonFlagSet() *flag.FlagSet {
	set := flag.NewFlagSet("kdt", flag.ContinueOnError)
	set.StringVar(&config.Key, "key", config.Key, "pre-shared secret between client and server")
	set.StringVar(&config.Crypt, "crypt", config.Crypt, "aes, aes-128, aes-192, salsa20, blowfish, twofish, cast5, 3des, tea, xtea, xor, none")
	set.StringVar(&config.Mode, "mode", config.Mode, "profiles: fast3, fast2, fast, normal")
	set.IntVar(&config.MTU, "mtu", config.MTU, "set maximum transmission unit for UDP packets")
	set.IntVar(&config.DataShard, "datashard", config.DataShard, "set reed-solomon erasure coding - datashard")
	set.IntVar(&config.ParityShard, "parityshard", config.ParityShard, "set reed-solomon erasure coding - parityshard")
	set.IntVar(&config.SndWnd, "sndwnd", config.SndWnd, "set send window size(num of packets)")
	set.IntVar(&config.RcvWnd, "rcvwnd", config.RcvWnd, "set receive window size(num of packets)")
	set.IntVar(&config.DSCP, "dscp", config.DSCP, "set DSCP(6bit)")
	set.BoolVar(&config.Comp, "comp", config.Comp, "enable compression")
	set.BoolVar(&config.AckNodelay, "acknodelay", config.AckNodelay, "flush ack immediately when a packet is received")
	set.IntVar(&config.NoDelay, "nodelay", config.NoDelay, "nodelay")
	set.IntVar(&config.Interval, "interval", config.Interval, "interval")
	set.IntVar(&config.Resend, "resend", config.Resend, "resend")
	set.IntVar(&config.NoCongestion, "nc", config.NoCongestion, "nc")
	set.IntVar(&config.SockBuf, "sockbuf", config.SockBuf, "sockbuf")
	set.IntVar(&config.KeepAlive, "keepalive", config.KeepAlive, "keepalive")
	set.StringVar(&config.Log, "log", config.Log, "specify a log file to output, default goes to stderr")
	set.IntVar(&config.BufferSize, "bufsize", config.BufferSize, "bufsize")
	set.StringVar(&config.ConfigFile, "conf", config.ConfigFile, "config from json file, which will override the command from shell")
	set.StringVar(&config.TransferID, "transfer_id", config.TransferID, "transfer_id")
	return set
}

func (config *ClientConfig) CreateFlagSet() *flag.FlagSet {
	set := config.Config.createCommonFlagSet()
	set.StringVar(&config.RemoteAddr, "remote", config.RemoteAddr, "kcp server address")
	set.IntVar(&config.Conn, "conn", config.Conn, "set num of UDP connections to server")
	set.IntVar(&config.AutoExpire, "autoexpire", config.AutoExpire, "set auto expiration time(in seconds) for a single UDP connection, 0 to disable")
	set.StringVar(&config.Directory, "directory", config.Directory, "base directory for sending file")
	return set
}

func (config *ServerConfig) CreateFlagSet() *flag.FlagSet {
	set := config.Config.createCommonFlagSet()
	set.StringVar(&config.Listen, "listen", config.Listen, "kcp server listen address")
	set.StringVar(&config.Root, "root", config.Root, "root directory")
	return set
}

func (config *ClientConfig) CreateWdtFlagSet() *flag.FlagSet {
	set := config.Config.createCommonFlagSet()
	set.StringVar(&config.Destination, "destination", config.Destination, "destination receiver address")
	set.IntVar(&config.StartPort, "start_port", config.StartPort, "destination receiver port")
	set.IntVar(&config.Conn, "conn", config.Conn, "set num of UDP connections to server")
	set.IntVar(&config.AutoExpire, "autoexpire", config.AutoExpire, "set auto expiration time(in seconds) for a single UDP connection, 0 to disable")
	set.StringVar(&config.Directory, "directory", config.Directory, "base directory for sending file")
	return set
}

func (config *ServerConfig) CreateWdtFlagSet() *flag.FlagSet {
	set := config.Config.createCommonFlagSet()
	set.StringVar(&config.Listen, "listen", config.Listen, "kcp server listen address")
	set.StringVar(&config.Root, "directory", config.Root, "root directory")
	set.IntVar(&config.StartPort, "start_port", config.StartPort, "start port")
	set.Bool("run_as_daemon", true, "run as daemon")
	return set
}

func (config *Config) LoadFile() {
	if config.ConfigFile != "" {
		err := parseJSONConfig(config, config.ConfigFile)
		log.Println("failed to parse json config", config.ConfigFile, err)
	}
}
