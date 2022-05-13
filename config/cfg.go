package config

import (
	"bytes"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Config WireGold 配置文件
type Config struct {
	IP         string `yaml:"IP"`
	SubNet     string `yaml:"SubNet"`
	PrivateKey string `yaml:"PrivateKey"`
	EndPoint   string `yaml:"EndPoint"`
	MTU        int64  `yaml:"MTU"`
	Peers      []Peer `yaml:"Peers"`
}

// Peer 对端信息
type Peer struct {
	IP               string   `yaml:"IP"`
	SubNet           string   `yaml:"SubNet"`
	PublicKey        string   `yaml:"PublicKey"`
	EndPoint         string   `yaml:"EndPoint"`
	AllowedIPs       []string `yaml:"AllowedIPs"`
	KeepAliveSeconds int64    `yaml:"KeepAliveSeconds"`
	QueryList        []string `yaml:"QueryList"`
	QuerySeconds     int64    `yaml:"QuerySeconds"`
	AllowTrans       bool     `yaml:"AllowTrans"`
	MTU              int64    `yaml:"MTU"`
}

func Parse(path string) (c Config) {
	file, err := os.ReadFile(path)
	if err != nil {
		log.Fatal("open config file failed:", err)
	}
	err = yaml.NewDecoder(bytes.NewReader(file)).Decode(&c)
	if err != nil {
		log.Fatal("invalid config file:", err)
	}
	return
}
