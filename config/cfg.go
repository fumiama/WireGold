package config

import (
	"bytes"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Config WireGold 配置文件
type Config struct {
	IP         string `yaml:"IP"`
	SubNet     string `yaml:"SubNet"`
	PrivateKey string `yaml:"PrivateKey"`
	Network    string `yaml:"Network"` // Network udp, tcp or ws (WIP)
	EndPoint   string `yaml:"EndPoint"`
	MTU        int64  `yaml:"MTU"` // MTU of nic (will minus packet header len)
	SpeedLoop  uint16 `yaml:"SpeedLoop"`
	Mask       uint64 `yaml:"Mask"` // Mask 是异或报文所用掩码, 必须保证各端统一
	Peers      []Peer `yaml:"Peers"`
}

// Peer 对端信息
type Peer struct {
	IP               string   `yaml:"IP"`
	PublicKey        string   `yaml:"PublicKey"`
	PresharedKey     string   `yaml:"PresharedKey"`
	EndPoint         string   `yaml:"EndPoint"`
	AllowedIPs       []string `yaml:"AllowedIPs"`
	KeepAliveSeconds int64    `yaml:"KeepAliveSeconds"`
	QueryList        []string `yaml:"QueryList"`
	QuerySeconds     int64    `yaml:"QuerySeconds"`
	AllowTrans       bool     `yaml:"AllowTrans"`
	UseZstd          bool     `yaml:"UseZstd"`
	DoublePacket     bool     `yaml:"DoublePacket"`
	MTU              int64    `yaml:"MTU"` // MTU of PDU passed to p2p
	MTURandomRange   int64    `yaml:"MTURandomRange"`
}

// Parse 解析配置文件
func Parse(path string) (c Config) {
	file, err := os.ReadFile(path)
	if err != nil {
		logrus.Fatal("open config file failed:", err)
	}
	err = yaml.NewDecoder(bytes.NewReader(file)).Decode(&c)
	if err != nil {
		logrus.Fatal("invalid config file:", err)
	}
	return
}
