package tunnel

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"

	curve "github.com/fumiama/go-x25519"
	"github.com/fumiama/orbyte"
	"github.com/sirupsen/logrus"

	"github.com/fumiama/WireGold/gold/link"
	"github.com/fumiama/WireGold/helper"
)

func TestTunnelUDP(t *testing.T) {
	testTunnelNetwork(t, "udp", 4096)
}

func TestTunnelUDPSmallMTU(t *testing.T) {
	testTunnelNetwork(t, "udp", 1024)
}

func TestTunnelUDPLite(t *testing.T) {
	if runtime.GOOS == "darwin" {
		return
	}
	testTunnelNetwork(t, "udplite", 4096)
}

func TestTunnelUDPLiteSmallMTU(t *testing.T) {
	if runtime.GOOS == "darwin" {
		return
	}
	testTunnelNetwork(t, "udplite", 1024)
}

func TestTunnelTCP(t *testing.T) {
	testTunnelNetwork(t, "tcp", 4096)
}

func TestTunnelTCPSmallMTU(t *testing.T) {
	testTunnelNetwork(t, "tcp", 1024)
}

func TestTunnelIP(t *testing.T) {
	testTunnelNetwork(t, "ip", 4096)
}

func TestTunnelIPSmallMTU(t *testing.T) {
	testTunnelNetwork(t, "ip", 1024)
}

func BenchmarkTunnelUDP(b *testing.B) {
	benchmarkTunnelNetwork(b, "udp", 4096)
}

func BenchmarkTunnelUDPSmallMTU(b *testing.B) {
	benchmarkTunnelNetwork(b, "udp", 1024)
}

func BenchmarkTunnelUDPLite(b *testing.B) {
	if runtime.GOOS == "darwin" {
		return
	}
	benchmarkTunnelNetwork(b, "udplite", 4096)
}

func BenchmarkTunnelUDPLiteSmallMTU(b *testing.B) {
	if runtime.GOOS == "darwin" {
		return
	}
	benchmarkTunnelNetwork(b, "udplite", 1024)
}

func BenchmarkTunnelTCP(b *testing.B) {
	benchmarkTunnelNetwork(b, "tcp", 4096)
}

func BenchmarkTunnelTCPSmallMTU(b *testing.B) {
	benchmarkTunnelNetwork(b, "tcp", 1024)
}

func BenchmarkTunnelIP(b *testing.B) {
	benchmarkTunnelNetwork(b, "ip", 4096)
}

func BenchmarkTunnelIPSmallMTU(b *testing.B) {
	benchmarkTunnelNetwork(b, "ip", 1024)
}

func testTunnel(t *testing.T, nw string, isplain, isbase14 bool, pshk *[32]byte, mtu uint16) {
	fmt.Println("start", nw, "testing")
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

	epm := "127.0.0.1"
	if nw != "ip" {
		epm += ":0"
	}
	// under macos you need to run
	//
	// sudo ifconfig lo0 alias 127.0.0.2
	epp := "127.0.0.2"
	if nw != "ip" {
		epp += ":0"
	}

	m := link.NewMe(&link.MyConfig{
		MyIPwithMask: "192.168.1.2/32",
		MyEndpoint:   epm,
		Network:      nw,
		PrivateKey:   selfpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          mtu,
		Base14:       isbase14,
	})
	defer m.Close()

	p := link.NewMe(&link.MyConfig{
		MyIPwithMask: "192.168.1.3/32",
		MyEndpoint:   epp,
		Network:      nw,
		PrivateKey:   peerpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          mtu,
		Base14:       isbase14,
	})
	defer p.Close()

	ppp := peerpk.Public()
	spp := selfpk.Public()
	if isplain {
		ppp = nil
		spp = nil
	}

	m.AddPeer(&link.PeerConfig{
		PeerIP:         "192.168.1.3",
		EndPoint:       p.EndPoint().String(),
		AllowedIPs:     []string{"192.168.1.3/32"},
		PubicKey:       ppp,
		PresharedKey:   pshk,
		MTU:            mtu,
		MTURandomRange: mtu / 2,
		UseZstd:        true,
		DoublePacket:   true,
	})
	p.AddPeer(&link.PeerConfig{
		PeerIP:         "192.168.1.2",
		EndPoint:       m.EndPoint().String(),
		AllowedIPs:     []string{"192.168.1.2/32"},
		PubicKey:       spp,
		PresharedKey:   pshk,
		MTU:            mtu,
		MTURandomRange: mtu / 2,
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
	go tunnme.Write(sendb)
	buf := make([]byte, 4)
	tunnpeer.Read(buf)
	if string(sendb) != string(buf) {
		logrus.Errorln("error: recv", buf, "expect", sendb)
		t.Fail()
	}

	sendb = make([]byte, 4096)
	for i := 0; i < 4096; i++ {
		sendb[i] = byte(i)
	}
	go tunnme.Write(sendb)
	buf = make([]byte, 4096)
	_, err = io.ReadFull(&tunnpeer, buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(sendb) != string(buf) {
		t.Fatal("error: recv 4096 bytes data")
	}

	sendbufs := make(chan []byte, 32)

	go func() {
		time.Sleep(time.Second)
		for i := 0; i < 32; i++ {
			sendb := make([]byte, 65535)
			for j := 0; j < 65535; j++ {
				sendb[j] = byte(i + j)
			}
			n, _ := tunnme.Write(sendb)
			sendbufs <- sendb
			logrus.Debugln("loop", i, "write", n, "bytes")
		}
		close(sendbufs)
	}()
	buf = make([]byte, 65535)
	i := 0
	for sendb := range sendbufs {
		n, err := io.ReadFull(&tunnpeer, buf)
		if err != nil {
			t.Fatal(err)
		}
		logrus.Debugln("loop", i, "read", n, "bytes")
		if string(sendb) != string(buf) {
			t.Fatal("loop", i, "error: recv 65535 bytes data")
		}
		i++
	}

	for i := 0; i < 4096; i++ {
		sendb[i] = ^byte(i)
	}
	tunnme.Write(sendb)
	rd := bytes.NewBuffer(nil)

	tm := time.AfterFunc(time.Second*2, func() {
		tunnme.Stop()
		tunnpeer.Stop()
	})
	defer tm.Stop()

	_, err = io.CopyBuffer(rd, &tunnpeer, make([]byte, 200))
	if err != nil {
		t.Fatal(err)
	}
	if string(sendb) != rd.String() {
		t.Fatal("error: recv fragmented 4096 bytes data")
	}
}

func benchmarkTunnel(b *testing.B, sz int, nw string, isplain, isbase14 bool, pshk *[32]byte, mtu uint16) {
	selfpk, err := curve.New(nil)
	if err != nil {
		panic(err)
	}
	peerpk, err := curve.New(nil)
	if err != nil {
		panic(err)
	}

	epm := "127.0.0.1"
	if nw != "ip" {
		epm += ":0"
	}
	// under macos you need to run
	//
	// sudo ifconfig lo0 alias 127.0.0.2
	epp := "127.0.0.2"
	if nw != "ip" {
		epp += ":0"
	}

	m := link.NewMe(&link.MyConfig{
		MyIPwithMask: "192.168.1.2/32",
		MyEndpoint:   epm,
		Network:      nw,
		PrivateKey:   selfpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          mtu,
		Base14:       isbase14,
	})
	defer m.Close()

	p := link.NewMe(&link.MyConfig{
		MyIPwithMask: "192.168.1.3/32",
		MyEndpoint:   epp,
		Network:      nw,
		PrivateKey:   peerpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          mtu,
		Base14:       isbase14,
	})
	defer p.Close()

	ppp := peerpk.Public()
	spp := selfpk.Public()
	if isplain {
		ppp = nil
		spp = nil
	}

	m.AddPeer(&link.PeerConfig{
		PeerIP:         "192.168.1.3",
		EndPoint:       p.EndPoint().String(),
		AllowedIPs:     []string{"192.168.1.3/32"},
		PubicKey:       ppp,
		PresharedKey:   pshk,
		MTU:            mtu,
		MTURandomRange: mtu / 2,
		UseZstd:        true,
		DoublePacket:   true,
	})
	p.AddPeer(&link.PeerConfig{
		PeerIP:         "192.168.1.2",
		EndPoint:       m.EndPoint().String(),
		AllowedIPs:     []string{"192.168.1.2/32"},
		PubicKey:       spp,
		PresharedKey:   pshk,
		MTU:            mtu,
		MTURandomRange: mtu / 2,
		UseZstd:        true,
	})
	tunnme, err := Create(&m, "192.168.1.3")
	if err != nil {
		b.Fatal(err)
	}
	tunnme.Start(1, 1, 4096)
	tunnpeer, err := Create(&p, "192.168.1.2")
	if err != nil {
		b.Fatal(err)
	}
	tunnpeer.Start(1, 1, 4096)

	time.Sleep(time.Second) // wait link up

	b.SetBytes(int64(sz))
	b.ResetTimer()
	sendb := make([]byte, sz)
	for i := 0; i < b.N; i++ {
		rand.Read(sendb)
		go tunnme.Write(sendb)
		buf := make([]byte, sz)
		_, err = io.ReadFull(&tunnpeer, buf)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	time.Sleep(time.Second) // wait packets all received

	tunnme.Stop()
	tunnpeer.Stop()
}

func testTunnelNetwork(t *testing.T, nw string, mtu uint16) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logFormat{enableColor: false})

	// test without base14
	testTunnel(t, nw, true, false, nil, mtu)  // test plain text
	testTunnel(t, nw, false, false, nil, mtu) // test normal

	// test with base14
	testTunnel(t, nw, true, true, nil, mtu)  // test plain text
	testTunnel(t, nw, false, true, nil, mtu) // test normal

	var buf [32]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		panic(err)
	}
	// test without base14
	testTunnel(t, nw, false, false, &buf, mtu) // test preshared

	// test with base14
	testTunnel(t, nw, false, true, &buf, mtu) // test preshared
}

func benchmarkTunnelNetwork(b *testing.B, nw string, mtu uint16) {
	logrus.SetLevel(logrus.ErrorLevel)
	logrus.SetFormatter(&logFormat{enableColor: false})

	for i := 1; i <= 4; i++ {
		sz := 1024 * i
		b.Run(fmt.Sprintf("%d-plain-nob14", sz), func(b *testing.B) {
			benchmarkTunnel(b, sz, nw, true, false, nil, mtu) // test plain text
		})
		b.Run(fmt.Sprintf("%d-normal-nob14", sz), func(b *testing.B) {
			benchmarkTunnel(b, sz, nw, false, false, nil, mtu) // test normal
		})
		b.Run(fmt.Sprintf("%d-plain-b14", sz), func(b *testing.B) {
			benchmarkTunnel(b, sz, nw, true, true, nil, mtu) // test plain text
		})
		b.Run(fmt.Sprintf("%d-normal-b14", sz), func(b *testing.B) {
			benchmarkTunnel(b, sz, nw, false, true, nil, mtu) // test normal
		})
		var buf [32]byte
		_, err := rand.Read(buf[:])
		if err != nil {
			panic(err)
		}
		b.Run(fmt.Sprintf("%d-preshared-nob14", sz), func(b *testing.B) {
			benchmarkTunnel(b, sz, nw, false, false, &buf, mtu) // test preshared
		})
		b.Run(fmt.Sprintf("%d-preshared-b14", sz), func(b *testing.B) {
			benchmarkTunnel(b, sz, nw, false, true, &buf, mtu) // test preshared
		})
	}
}

// logFormat specialize for go-cqhttp
type logFormat struct {
	enableColor bool
}

// Format implements logrus.Formatter
func (f logFormat) Format(entry *logrus.Entry) ([]byte, error) {
	// this writer will not be put back

	buf := (*orbyte.Item[bytes.Buffer])(helper.SelectWriter()).Trans().Pointer()

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

	return buf.Bytes(), nil
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
