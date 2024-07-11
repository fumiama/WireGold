package main

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	base14 "github.com/fumiama/go-base16384"
	curve "github.com/fumiama/go-x25519"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/helper"
	"github.com/fumiama/WireGold/upper"
	"github.com/fumiama/WireGold/upper/services/wg"
)

func main() {
	help := flag.Bool("h", false, "display this help")
	gen := flag.Bool("g", false, "generate key pair")
	pshgen := flag.Bool("pg", false, "generate preshared key")
	showp := flag.Bool("p", false, "show my publickey")
	file := flag.String("c", "config.yaml", "specify conf file")
	debug := flag.Bool("d", false, "print debug logs")
	warn := flag.Bool("w", false, "only show logs above warn level")
	logfile := flag.String("l", "-", "write log to file")
	flag.Parse()
	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else if *warn {
		logrus.SetLevel(logrus.WarnLevel)
	}
	if *help {
		displayHelp("")
	}
	if *gen {
		k, err := curve.New(nil)
		if err != nil {
			panic(err)
		}
		pubk, err := base14.UTF16BE2UTF8(base14.Encode((*k.Public())[:]))
		if err != nil {
			panic(err)
		}
		prvk, err := base14.UTF16BE2UTF8(base14.Encode((*k.Private())[:]))
		if err != nil {
			panic(err)
		}
		fmt.Println("PublicKey:", helper.BytesToString(pubk[:57]))
		fmt.Println("PrivateKey:", helper.BytesToString(prvk[:57]))
		os.Exit(0)
	}
	if *pshgen {
		var buf [32]byte
		_, err := rand.Read(buf[:])
		if err != nil {
			panic(err)
		}
		pshk, err := base14.UTF16BE2UTF8(base14.Encode(buf[:]))
		if err != nil {
			panic(err)
		}
		fmt.Println("PresharedKey:", helper.BytesToString(pshk[:57]))
		os.Exit(0)
	}
	if *logfile != "-" {
		f, err := os.Create(*logfile)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		logrus.SetOutput(f)
	}
	if helper.IsNotExist(*file) {
		f := new(bytes.Buffer)
		var r string
		fmt.Print("IP: ")
		fmt.Scanln(&r)
		if r == "" {
			fmt.Println("nil ip")
			return
		}
		f.WriteString("IP: " + strings.TrimSpace(r) + "\n")
		r = ""

		fmt.Print("SubNet: ")
		fmt.Scanln(&r)
		if r == "" {
			fmt.Println("nil subnet")
			return
		}
		f.WriteString("SubNet: " + strings.TrimSpace(r) + "\n")
		r = ""

		fmt.Print("PrivateKey: ")
		fmt.Scanln(&r)
		if r == "" {
			fmt.Println("nil private key")
			return
		}
		f.WriteString("PrivateKey: " + strings.TrimSpace(r) + "\n")
		r = ""

		fmt.Print("EndPoint: ")
		fmt.Scanln(&r)
		if r == "" {
			fmt.Println("nil endpoint")
			return
		}
		f.WriteString("EndPoint: " + strings.TrimSpace(r) + "\n")
		r = ""

		fmt.Print("MTU: ")
		fmt.Scanln(&r)
		if r == "" {
			fmt.Println("nil endpoint")
			return
		}
		f.WriteString("MTU: " + strings.TrimSpace(r) + "\n")
		r = ""

		cfgf, err := os.Create(*file)
		if err != nil {
			panic(err)
		}
		cfgf.Write(f.Bytes())
		cfgf.Close()
	}
	c := config.Parse(*file)
	if c.IP == "" {
		displayHelp("nil ip")
	}
	if c.SubNet == "" {
		displayHelp("nil subnet")
	}
	if c.PrivateKey == "" {
		displayHelp("nil private key")
	}
	if c.EndPoint == "" {
		displayHelp("nil endpoint")
	}
	if c.MTU == 0 {
		displayHelp("nil mtu")
	}
	w, err := wg.NewWireGold(&c)
	if err != nil {
		panic(err)
	}

	if *showp {
		fmt.Println("PublicKey:", w.PublicKey)
		os.Exit(0)
	}

	mc := make(chan os.Signal, 1)
	signal.Notify(mc, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		<-mc
		w.Stop()
	}()
	w.Run(upper.ServiceWireGold, upper.ServiceWireGold)
}

func displayHelp(hint string) {
	fmt.Println(hint)
	flag.Usage()
	os.Exit(0)
}
