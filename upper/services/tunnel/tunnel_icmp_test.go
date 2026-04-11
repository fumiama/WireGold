//go:build linux

package tunnel

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"testing"
	"time"

	curve "github.com/fumiama/go-x25519"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/fumiama/WireGold/gold/link"
)

const (
	icmpNS1   = "wgtest_ns1"
	icmpNS2   = "wgtest_ns2"
	icmpIP1   = "10.0.0.1"
	icmpIP2   = "10.0.0.2"
	icmpVeth1 = "veth1"
	icmpVeth2 = "veth2"
)

// setupICMPNetns creates two network namespaces connected by a veth pair.
// It returns a cleanup function. Requires root.
func setupICMPNetns(t *testing.T) func() {
	t.Helper()

	cmds := [][]string{
		{"ip", "netns", "add", icmpNS1},
		{"ip", "netns", "add", icmpNS2},
		{"ip", "link", "add", icmpVeth1, "type", "veth", "peer", "name", icmpVeth2},
		{"ip", "link", "set", icmpVeth1, "netns", icmpNS1},
		{"ip", "link", "set", icmpVeth2, "netns", icmpNS2},
		{"ip", "netns", "exec", icmpNS1, "ifconfig", icmpVeth1, icmpIP1, "up"},
		{"ip", "netns", "exec", icmpNS2, "ifconfig", icmpVeth2, icmpIP2, "up"},
	}

	for _, args := range cmds {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			// best-effort cleanup
			exec.Command("ip", "netns", "del", icmpNS1).Run()
			exec.Command("ip", "netns", "del", icmpNS2).Run()
			t.Fatalf("setup netns: %v failed: %v\n%s", args, err, out)
		}
	}

	return func() {
		exec.Command("ip", "netns", "del", icmpNS1).Run()
		exec.Command("ip", "netns", "del", icmpNS2).Run()
	}
}

// enterNetns pins the current goroutine to its OS thread, switches into
// the named network namespace, and returns a function that restores the
// original namespace and unlocks the thread.
func enterNetns(nsName string) (func(), error) {
	runtime.LockOSThread()

	origFd, err := unix.Open("/proc/self/ns/net", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("open current netns: %w", err)
	}

	targetFd, err := unix.Open("/var/run/netns/"+nsName, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		unix.Close(origFd)
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("open target netns %s: %w", nsName, err)
	}

	if err := unix.Setns(targetFd, unix.CLONE_NEWNET); err != nil {
		unix.Close(targetFd)
		unix.Close(origFd)
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("setns to %s: %w", nsName, err)
	}
	unix.Close(targetFd)

	return func() {
		unix.Setns(origFd, unix.CLONE_NEWNET)
		unix.Close(origFd)
		runtime.UnlockOSThread()
	}, nil
}

// initMeInNetns initializes a link.Me at dst inside the given network namespace.
// The underlying socket fd remains bound to that namespace after return.
func initMeInNetns(t testing.TB, nsName string, cfg *link.MyConfig, dst *link.Me) {
	t.Helper()
	var merr any
	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				merr = r
			}
			close(done)
		}()
		restore, err := enterNetns(nsName)
		if err != nil {
			merr = err
			return
		}
		defer restore()
		*dst = link.NewMe(cfg)
	}()
	<-done
	if merr != nil {
		t.Fatalf("initMeInNetns(%s): %v", nsName, merr)
	}
}

func TestTunnelICMP(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping ICMP test: requires root")
	}
	for i := 1; i <= 4; i++ {
		sz := 1024 * i
		if !t.Run(strconv.Itoa(sz), func(t *testing.T) {
			testTunnelICMP(t, uint16(sz))
		}) {
			return
		}
	}
}

func testTunnelICMP(t *testing.T, mtu uint16) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logFormat{enableColor: false})

	cleanup := setupICMPNetns(t)
	defer cleanup()

	testICMPTunnel(t, true, false, nil, mtu)  // plain text
	testICMPTunnel(t, false, false, nil, mtu) // normal

	testICMPTunnel(t, true, true, nil, mtu)  // plain text + base14
	testICMPTunnel(t, false, true, nil, mtu) // normal + base14

	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		t.Fatal(err)
	}
	testICMPTunnel(t, false, false, &buf, mtu) // preshared
	testICMPTunnel(t, false, true, &buf, mtu)  // preshared + base14
}

func testICMPTunnel(t *testing.T, isplain, isbase14 bool, pshk *[32]byte, mtu uint16) {
	nw := "icmp"
	fmt.Println("start", nw, "testing, mtu", mtu, "plain", isplain, "b14", isbase14, "pshk", pshk != nil)

	selfpk, err := curve.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	peerpk, err := curve.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("my priv key:", hex.EncodeToString(selfpk.Private()[:]))
	t.Log("my publ key:", hex.EncodeToString(selfpk.Public()[:]))
	t.Log("peer priv key:", hex.EncodeToString(peerpk.Private()[:]))
	t.Log("peer publ key:", hex.EncodeToString(peerpk.Public()[:]))

	var m link.Me
	initMeInNetns(t, icmpNS1, &link.MyConfig{
		MyIPwithMask: "192.168.1.2/32",
		MyEndpoint:   icmpIP1,
		Network:      nw,
		PrivateKey:   selfpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          mtu,
		Base14:       isbase14,
	}, &m)
	defer m.Close()

	var p link.Me
	initMeInNetns(t, icmpNS2, &link.MyConfig{
		MyIPwithMask: "192.168.1.3/32",
		MyEndpoint:   icmpIP2,
		Network:      nw,
		PrivateKey:   peerpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          mtu,
		Base14:       isbase14,
	}, &p)
	defer p.Close()

	ppp := peerpk.Public()
	spp := selfpk.Public()
	if isplain {
		ppp = nil
		spp = nil
	}

	m.AddPeer(&link.PeerConfig{
		PeerIP:         "192.168.1.3",
		EndPoint:       icmpIP2,
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
		EndPoint:       icmpIP1,
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

	sendb = make([]byte, mtu+4)
	for i := 0; i < len(sendb); i++ {
		sendb[i] = byte(i)
	}

	for i := 1; i < len(sendb); i++ {
		rand.Read(sendb[:i])
		go tunnme.Write(sendb[:i])
		rbuf := make([]byte, i)
		_, err = io.ReadFull(&tunnpeer, rbuf)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(sendb[:i], rbuf) {
			t.Fatal("error: recv", i, "bytes data")
		}
	}

	for i := 0; i < len(sendb); i++ {
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
		t.Fatal("error: recv fragmented data")
	}
}

func BenchmarkTunnelICMP(b *testing.B) {
	if os.Getuid() != 0 {
		b.Skip("skipping ICMP benchmark: requires root")
	}
	benchmarkTunnelNetworkICMP(b, 4096)
}

func BenchmarkTunnelICMPSmallMTU(b *testing.B) {
	if os.Getuid() != 0 {
		b.Skip("skipping ICMP benchmark: requires root")
	}
	benchmarkTunnelNetworkICMP(b, 1024)
}

func benchmarkTunnelNetworkICMP(b *testing.B, mtu uint16) {
	logrus.SetLevel(logrus.ErrorLevel)
	logrus.SetFormatter(&logFormat{enableColor: false})

	cleanup := setupICMPBenchNetns(b)
	defer cleanup()

	for i := 1; i <= 4; i++ {
		sz := 1024 * i
		b.Run(fmt.Sprintf("%d-plain-nob14", sz), func(b *testing.B) {
			benchmarkICMPTunnel(b, sz, true, false, nil, mtu)
		})
		b.Run(fmt.Sprintf("%d-normal-nob14", sz), func(b *testing.B) {
			benchmarkICMPTunnel(b, sz, false, false, nil, mtu)
		})
		b.Run(fmt.Sprintf("%d-plain-b14", sz), func(b *testing.B) {
			benchmarkICMPTunnel(b, sz, true, true, nil, mtu)
		})
		b.Run(fmt.Sprintf("%d-normal-b14", sz), func(b *testing.B) {
			benchmarkICMPTunnel(b, sz, false, true, nil, mtu)
		})
		var buf [32]byte
		if _, err := rand.Read(buf[:]); err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("%d-preshared-nob14", sz), func(b *testing.B) {
			benchmarkICMPTunnel(b, sz, false, false, &buf, mtu)
		})
		b.Run(fmt.Sprintf("%d-preshared-b14", sz), func(b *testing.B) {
			benchmarkICMPTunnel(b, sz, false, true, &buf, mtu)
		})
	}
}

func setupICMPBenchNetns(b *testing.B) func() {
	b.Helper()

	cmds := [][]string{
		{"ip", "netns", "add", icmpNS1},
		{"ip", "netns", "add", icmpNS2},
		{"ip", "link", "add", icmpVeth1, "type", "veth", "peer", "name", icmpVeth2},
		{"ip", "link", "set", icmpVeth1, "netns", icmpNS1},
		{"ip", "link", "set", icmpVeth2, "netns", icmpNS2},
		{"ip", "netns", "exec", icmpNS1, "ifconfig", icmpVeth1, icmpIP1, "up"},
		{"ip", "netns", "exec", icmpNS2, "ifconfig", icmpVeth2, icmpIP2, "up"},
	}

	for _, args := range cmds {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			exec.Command("ip", "netns", "del", icmpNS1).Run()
			exec.Command("ip", "netns", "del", icmpNS2).Run()
			b.Fatalf("setup netns: %v failed: %v\n%s", args, err, out)
		}
	}

	return func() {
		exec.Command("ip", "netns", "del", icmpNS1).Run()
		exec.Command("ip", "netns", "del", icmpNS2).Run()
	}
}

func benchmarkICMPTunnel(b *testing.B, sz int, isplain, isbase14 bool, pshk *[32]byte, mtu uint16) {
	nw := "icmp"

	selfpk, err := curve.New(nil)
	if err != nil {
		b.Fatal(err)
	}
	peerpk, err := curve.New(nil)
	if err != nil {
		b.Fatal(err)
	}

	var m link.Me
	initMeInNetns(b, icmpNS1, &link.MyConfig{
		MyIPwithMask: "192.168.1.2/32",
		MyEndpoint:   icmpIP1,
		Network:      nw,
		PrivateKey:   selfpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          mtu,
		Base14:       isbase14,
	}, &m)
	defer m.Close()

	var p link.Me
	initMeInNetns(b, icmpNS2, &link.MyConfig{
		MyIPwithMask: "192.168.1.3/32",
		MyEndpoint:   icmpIP2,
		Network:      nw,
		PrivateKey:   peerpk.Private(),
		SrcPort:      1,
		DstPort:      1,
		MTU:          mtu,
		Base14:       isbase14,
	}, &p)
	defer p.Close()

	ppp := peerpk.Public()
	spp := selfpk.Public()
	if isplain {
		ppp = nil
		spp = nil
	}

	m.AddPeer(&link.PeerConfig{
		PeerIP:         "192.168.1.3",
		EndPoint:       icmpIP2,
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
		EndPoint:       icmpIP1,
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
