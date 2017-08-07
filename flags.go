package kdt

import "github.com/urfave/cli"

func createCommonFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   "key",
			Value:  "it's a kcp secrect",
			Usage:  "pre-shared secret between client and server",
			EnvVar: "LEKCPTUN_KEY",
		},
		cli.StringFlag{
			Name:  "crypt",
			Value: "none",
			Usage: "aes, aes-128, aes-192, salsa20, blowfish, twofish, cast5, 3des, tea, xtea, xor, none",
		},
		cli.StringFlag{
			Name:  "mode",
			Value: "fast",
			Usage: "profiles: fast3, fast2, fast, normal",
		},
		cli.IntFlag{
			Name:  "mtu",
			Value: 1350,
			Usage: "set maximum transmission unit for UDP packets",
		},
		cli.IntFlag{
			Name:  "datashard",
			Value: 10,
			Usage: "set reed-solomon erasure coding - datashard",
		},
		cli.IntFlag{
			Name:  "parityshard",
			Value: 3,
			Usage: "set reed-solomon erasure coding - parityshard",
		},
		cli.IntFlag{
			Name:  "dscp",
			Value: 0,
			Usage: "set DSCP(6bit)",
		},
		cli.BoolFlag{
			Name:  "comp",
			Usage: "enable compression",
		},
		cli.BoolFlag{
			Name:   "acknodelay",
			Usage:  "flush ack immediately when a packet is received",
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "nodelay",
			Value:  0,
			Hidden: false,
		},
		cli.IntFlag{
			Name:   "interval",
			Value:  40,
			Hidden: false,
		},
		cli.IntFlag{
			Name:   "resend",
			Value:  0,
			Hidden: false,
		},
		cli.IntFlag{
			Name:   "nc",
			Value:  0,
			Hidden: false,
		},
		cli.IntFlag{
			Name:   "sockbuf",
			Value:  4194304, // socket buffer size in bytes
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "keepalive",
			Value:  10, // nat keepalive interval in seconds
			Hidden: true,
		},
		cli.StringFlag{
			Name:  "log",
			Value: "",
			Usage: "specify a log file to output, default goes to stderr",
		},
		cli.StringFlag{
			Name:  "c",
			Value: "", // when the value is not empty, the config path must exists
			Usage: "config from json file, which will override the command from shell",
		},
		cli.StringFlag{
			Name:  "transfer_id",
			Value: "",
			Usage: "transfer id",
		},
	}
}

func concatFlags(x []cli.Flag, y []cli.Flag) []cli.Flag {
	flags := make([]cli.Flag, len(x)+len(y))
	for i, f := range x {
		flags[i] = f
	}
	for i, f := range y {
		flags[i+len(x)] = f
	}
	return flags
}

func CreateClientFlags() []cli.Flag {
	return concatFlags(createCommonFlags(), []cli.Flag{
		cli.StringFlag{
			Name:  "remoteaddr, r",
			Value: "",
			Usage: "kcp server address",
		},
		cli.IntFlag{
			Name:  "conn",
			Value: 1,
			Usage: "set num of UDP connections to server",
		},
		cli.IntFlag{
			Name:  "autoexpire",
			Value: 0,
			Usage: "set auto expiration time(in seconds) for a single UDP connection, 0 to disable",
		},
		cli.IntFlag{
			Name:  "sndwnd",
			Value: 1024,
			Usage: "set send window size(num of packets)",
		},
		cli.IntFlag{
			Name:  "rcvwnd",
			Value: 1024,
			Usage: "set receive window size(num of packets)",
		},
		cli.IntFlag{
			Name:  "bandwidth",
			Value: 0,
			Usage: "set transfer bandwidth(num of Kbits)",
		},
	})
}

func CreateServerFlags() []cli.Flag {
	return concatFlags(createCommonFlags(), []cli.Flag{
		cli.StringFlag{
			Name:  "listen,l",
			Value: ":29900",
			Usage: "kcp server listen address",
		},
		cli.StringFlag{
			Name:  "root",
			Value: ".",
			Usage: "root directory",
		},
		cli.IntFlag{
			Name:  "sndwnd",
			Value: 1024,
			Usage: "set send window size(num of packets)",
		},
		cli.IntFlag{
			Name:  "rcvwnd",
			Value: 1024,
			Usage: "set receive window size(num of packets)",
		},
	})
}
