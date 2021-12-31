package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	base14 "github.com/fumiama/go-base16384"
	curve "github.com/fumiama/go-x25519"

	"github.com/fumiama/WireGold/config"
	"github.com/fumiama/WireGold/helper"
	"github.com/fumiama/WireGold/upper"
	"github.com/fumiama/WireGold/upper/services/wg"
)

func main() {
	help := flag.Bool("h", false, "display this help")
	gen := flag.Bool("g", false, "generate key pair")
	showp := flag.Bool("p", false, "show my publickey")
	file := flag.String("c", "config.yaml", "specify conf file")
	flag.Parse()
	if *help {
		displayHelp("")
	}
	if *gen {
		k, err := curve.New(nil)
		if err != nil {
			panic(err)
		}
		pubk, err := base14.UTF16be2utf8(base14.Encode((*k.Public())[:]))
		if err != nil {
			panic(err)
		}
		prvk, err := base14.UTF16be2utf8(base14.Encode((*k.Private())[:]))
		if err != nil {
			panic(err)
		}
		fmt.Println("PublicKey:", helper.BytesToString(pubk[:57]))
		fmt.Println("PrivateKey:", helper.BytesToString(prvk[:57]))
		os.Exit(0)
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
		f.WriteString("IP: " + r + "\n")
		r = ""

		fmt.Print("SubNet: ")
		fmt.Scanln(&r)
		if r == "" {
			fmt.Println("nil subnet")
			return
		}
		f.WriteString("SubNet: " + r + "\n")
		r = ""

		fmt.Print("PrivateKey: ")
		fmt.Scanln(&r)
		if r == "" {
			fmt.Println("nil private key")
			return
		}
		f.WriteString("PrivateKey: " + r + "\n")
		r = ""

		fmt.Print("EndPoint: ")
		fmt.Scanln(&r)
		if r == "" {
			fmt.Println("nil endpoint")
			return
		}
		f.WriteString("EndPoint: " + r + "\n")
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
	w, err := wg.NewWireGold(&c)
	if err != nil {
		panic(err)
	}

	if *showp {
		fmt.Println("PublicKey:", w.PublicKey)
		os.Exit(0)
	}

	defer w.Stop()
	w.Run(upper.ServiceWireGold, upper.ServiceWireGold, 32768-64)
}

func displayHelp(hint string) {
	fmt.Println(hint)
	flag.Usage()
	os.Exit(0)
}
