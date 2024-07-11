package tunnel

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"strings"
	"testing"
	"time"

	curve "github.com/fumiama/go-x25519"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/link"
	"github.com/fumiama/WireGold/helper"
)

func TestTunnel(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logFormat{enableColor: false})

	selfpk, err := curve.New(nil)
	if err != nil {
		panic(err)
	}
	peerpk, err := curve.New(nil)
	if err != nil {
		panic(err)
	}
	t.Log("my priv key:", hex.EncodeToString(selfpk.Private()[:]))
	t.Log("my publ key:", hex.EncodeToString(selfpk.Public()[:]))
	t.Log("peer priv key:", hex.EncodeToString(peerpk.Private()[:]))
	t.Log("peer publ key:", hex.EncodeToString(peerpk.Public()[:]))

	m := link.NewMe(&link.MyConfig{
		MyIPwithMask: "192.168.1.2/32",
		MyEndpoint:   "127.0.0.1:21246",
		PrivateKey:   selfpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          4096,
	})
	m.AddPeer(&link.PeerConfig{
		PeerIP:         "192.168.1.3",
		EndPoint:       "127.0.0.1:21247",
		AllowedIPs:     []string{"192.168.1.3/32"},
		PubicKey:       peerpk.Public(),
		MTU:            4096,
		MTURandomRange: 1024,
		UseZstd:        true,
	})
	p := link.NewMe(&link.MyConfig{
		MyIPwithMask: "192.168.1.3/32",
		MyEndpoint:   "127.0.0.1:21247",
		PrivateKey:   peerpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          4096,
	})
	p.AddPeer(&link.PeerConfig{
		PeerIP:         "192.168.1.2",
		EndPoint:       "127.0.0.1:21246",
		AllowedIPs:     []string{"192.168.1.2/32"},
		PubicKey:       selfpk.Public(),
		MTU:            4096,
		MTURandomRange: 1024,
		UseZstd:        true,
	})
	tunnme, err := Create(&m, "192.168.1.3")
	if err != nil {
		t.Fatal(err)
	}
	tunnme.Start(1, 1, 4096)
	tunnpeer, err := Create(&p, "192.168.1.2")
	if err != nil {
		t.Fatal(err)
	}
	tunnpeer.Start(1, 1, 4096)

	time.Sleep(time.Second) // wait link up

	sendb := ([]byte)("1234")
	tunnme.Write(sendb)
	buf := make([]byte, 4)
	tunnpeer.Read(buf)
	if string(sendb) != string(buf) {
		t.Log("error: recv", buf)
		t.Fail()
	}

	sendb = make([]byte, 4096)
	rand.Read(sendb)
	tunnme.Write(sendb)
	buf = make([]byte, 4096)
	tunnpeer.Read(buf)
	if string(sendb) != string(buf) {
		t.Fatal("error: recv 4096 bytes data")
	}

	sendb = make([]byte, 65535)
	rand.Read(sendb)
	n, _ := tunnme.Write(sendb)
	t.Log("write", n, "bytes")
	buf = make([]byte, 65535)
	n, _ = io.ReadFull(&tunnpeer, buf)
	t.Log("read", n, "bytes")
	if string(sendb) != string(buf) {
		t.Fatal("error: recv 65535 bytes data")
		t.Log("expect", hex.EncodeToString(sendb))
		t.Log("got", hex.EncodeToString(buf))
	}

	tunnme.Stop()
	tunnpeer.Stop()
}

// logFormat specialize for go-cqhttp
type logFormat struct {
	enableColor bool
}

// Format implements logrus.Formatter
func (f logFormat) Format(entry *logrus.Entry) ([]byte, error) {
	buf := helper.SelectWriter()
	defer helper.PutWriter(buf)

	buf.WriteByte('[')
	if f.enableColor {
		buf.WriteString(getLogLevelColorCode(entry.Level))
	}
	buf.WriteString(strings.ToUpper(entry.Level.String()))
	if f.enableColor {
		buf.WriteString(colorReset)
	}
	buf.WriteString("] ")
	buf.WriteString(entry.Message)
	buf.WriteString("\n")

	ret := make([]byte, len(buf.Bytes()))
	copy(ret, buf.Bytes()) // copy buffer
	return ret, nil
}

const (
	colorCodePanic = "\x1b[1;31m" // color.Style{color.Bold, color.Red}.String()
	colorCodeFatal = "\x1b[1;31m" // color.Style{color.Bold, color.Red}.String()
	colorCodeError = "\x1b[31m"   // color.Style{color.Red}.String()
	colorCodeWarn  = "\x1b[33m"   // color.Style{color.Yellow}.String()
	colorCodeInfo  = "\x1b[37m"   // color.Style{color.White}.String()
	colorCodeDebug = "\x1b[32m"   // color.Style{color.Green}.String()
	colorCodeTrace = "\x1b[36m"   // color.Style{color.Cyan}.String()
	colorReset     = "\x1b[0m"
)

// getLogLevelColorCode 获取日志等级对应色彩code
func getLogLevelColorCode(level logrus.Level) string {
	switch level {
	case logrus.PanicLevel:
		return colorCodePanic
	case logrus.FatalLevel:
		return colorCodeFatal
	case logrus.ErrorLevel:
		return colorCodeError
	case logrus.WarnLevel:
		return colorCodeWarn
	case logrus.InfoLevel:
		return colorCodeInfo
	case logrus.DebugLevel:
		return colorCodeDebug
	case logrus.TraceLevel:
		return colorCodeTrace

	default:
		return colorCodeInfo
	}
}
