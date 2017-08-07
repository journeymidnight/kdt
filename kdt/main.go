package main

import (
	"errors"
    "time"
    "log"
    "os"
	"math/rand"
    "github.com/journeymidnight/kdt"
	"github.com/urfave/cli"
)

const (
	// VERSION is injected by buildflags
	VERSION = "0.1"
	// SALT is use for pbkdf2 key expansion
	SALT = "lectpkdt"
)

func onProgress(client *kdt.Client, starttime time.Time, offset int64, transferred int64, total int64, ptimes int) {
	if ptimes % 1000 == 1 {
		usedTime := 1000.0 * float64(time.Since(starttime)) / float64(time.Second)
		speed := float64(transferred) / usedTime
		percent := float64(offset + transferred) * 100.0 / float64(total)
		log.Printf("onProgress percent=%.02f%% totaltransfered=%d offset=%d transfered=%d total=%d usedtime=%.2fms speed=%.2fKB/s\n", percent, offset + transferred, offset, transferred, total, usedTime, speed)
	}
}

func runClient(c *cli.Context) error {
    log.Println("runClient", c.NArg(), c.Args())
	config := &kdt.ClientConfig{}
	err := config.Init(c)
    client := kdt.CreateClient(config)
	ptimes := 0
	client.Callback = func (starttime time.Time, offset, transferred, total int64) {
		onProgress(client, starttime, offset, transferred, total, ptimes)
		ptimes += 1
	}
	if c.NArg() != 1 {
		log.Println("runClient invalid argument", c.Args)
		return errors.New("runClient invalid argument")
	}
    input := c.Args()[0]
    err = client.SendFile(input)

    log.Println("runClient ok", client, err)
    return err
}

func runServer(c *cli.Context) error {
    log.Println("runServer")
	config := &kdt.ServerConfig{}
	err := config.Init(c)
    server, err := kdt.ReceiveFiles(config)
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
