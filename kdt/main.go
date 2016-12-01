package main

import (
    "time"
    "log"
    "os"
	"math/rand"
    "git.letv.cn/ctp/kdt"
	"github.com/urfave/cli"
)

const (
	// VERSION is injected by buildflags
	VERSION = "0.1"
	// SALT is use for pbkdf2 key expansion
	SALT = "lectpkdt"
)

func onProgress(client *kdt.Client, total int64, ptimes int) {
	if ptimes % 1000 == 1 {
		usedTime := 1000.0 * float64(time.Since(client.StartTime)) / float64(time.Second)
		speed := float64(total) / usedTime
		log.Println("onProgress", total, usedTime, speed)
	}
}

func runClient(c *cli.Context) error {
    log.Println("runClient")
	config := &kdt.ClientConfig{}
	err := config.Init(c)
    client := kdt.CreateClient(config)
	ptimes := 0
	client.Callback = func (total int64) {
		onProgress(client, total, ptimes)
		ptimes += 1
	}
    input := c.String("input")
    err = client.SendFile(input)

    log.Println("runClient ok", client, err)
    return err
}

func runServer(c *cli.Context) error {
    log.Println("runServer")
	config := &kdt.ServerConfig{}
	err := config.Init(c)
    block := config.CreateBlockCrypt()
    server, err := kdt.ReceiveFiles(config, block)
    log.Println("runServer ok", server, err)
    return err
}

func main() {
	log.SetOutput(os.Stderr)
	// memprofile := "memprofile"
	// cpuprofile := "cpuprofile"
	// f, err := os.Create(cpuprofile)
	// if err != nil {
	// 	log.Fatal("could not create CPU profile: ", err)
	// }
	// if err := pprof.StartCPUProfile(f); err != nil {
	// 	log.Fatal("could not start CPU profile: ", err)
	// }
	// defer pprof.StopCPUProfile()
	rand.Seed(int64(time.Now().Nanosecond()))
	if VERSION == "SELFBUILD" {
		// add more log flags for debugging
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
	myApp := cli.NewApp()
	myApp.Name = "kdt"
	myApp.Usage = "kcptun-based data transferer"
	myApp.Version = VERSION
	myApp.Commands = []cli.Command{
		{
			Name:    "client",
			Aliases: []string{"c"},
			Usage:   "client",
			Action:  runClient,
			Flags:   kdt.CreateClientFlags(),
		},
		{
			Name:    "server",
			Aliases: []string{"s"},
			Usage:   "server",
			Action:  runServer,
			Flags:   kdt.CreateServerFlags(),
		},
	}
	myApp.Run(os.Args)
    // {
    //     f2, err := os.Create(memprofile)
    //     if err != nil {
    //         log.Fatal("could not create memory profile: ", err)
    //     }
    //     // runtime.GC() // get up-to-date statistics
    //     if err := pprof.WriteHeapProfile(f2); err != nil {
    //         log.Fatal("could not write memory profile: ", err)
    //     }
    //     f2.Close()
    // }
}
