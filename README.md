<div align="center">
  <a href="https://crypko.ai/crypko/G39ZPfer7g6rz/">
    <img src=".github/Maria.png" width = "400" alt="WireGold-Maria">
  </a><br>
  <h1>WireGold</h1>
  Wire Golang Guard = WireGold<br><br>
</div>

## Usage
> If you are running in windows, remember to select the `wintun.dll` of your arch in `lower/wintun` and place it alongside the compiled exe

> It is highly recommanded to use [UDPspeeder](https://github.com/wangyu-/UDPspeeder) together if you are using a High-latency Lossy Link
```bash
wg [-c config.yaml] [-d|w] [-g] [-h] [-p] [-l log.txt]
```
#### Instructions
```bash
  -c string
        specify conf file (default "config.yaml")
  -d    print debug logs
  -g    generate key pair
  -h    display this help
  -l string
        write log to file (default "-")
  -p    show my publickey
  -pg
        generate preshared key
  -w    only show logs above warn level
```

## Config file example

- **macos mojave**: max mtu (under ipv4 endpoint) is `9159`
- **ipv6 endpoint**: the recommand mtu is `1280~1500` to prevent the big segments from being dropped

```yaml
IP: 192.168.233.1
SubNet: 192.168.233.0/24
PrivateKey: 暲菉斂狧污爉窫擸紈卆帞蔩慈睠庮扝憚瞼縀
EndPoint: 0.0.0.0:56789
MTU: 1504
SpeedLoop: 4096
Mask: 0x1234567890abcdef
Base14: true
Peers:
  -
    IP: "192.168.233.2"
    PublicKey: 徯萃嵾爻燸攗窍褃冔蒔犡緇袿屿組待族砇嘀
    PresharedKey: 瀸敀爅崾嘊嵜紼樴稍毯攣矐訷蟷扛嬋庩崛昀
    EndPoint: 1.2.3.4:56789
    AllowedIPs: ["192.168.233.2/32", "x192.168.233.3/32"] # accept packets from 192.168.233.3, but don not create route
    KeepAliveSeconds: 0
    QueryList: ["192.168.233.3"]
    MTU: 1400
    MTURandomRange: 128
    UseZstd: true
    QuerySeconds: 10
    AllowTrans: true
  -
    IP: "192.168.233.3"
    PublicKey: 牢喨粷詸衭譛浾蘹櫠砙杹蟫瑳叩刋橋経挵蘀
    PresharedKey: 竅琚喫従痸告烈兇厕趭萨假蔛瀇譄施烸蝫瘀
    EndPoint: ""
    AllowedIPs: ["192.168.233.3/32", "y192.168.66.1/32"] # add route to 192.168.66.1 into inner route table but do not add it to system one
    MTU: 752
    DoublePacket: true
    KeepAliveSeconds: 0
    AllowTrans: false
```

## Benckmark on localhost
> This benckmark is tested on Mac Book Air M1 within battery mode.

### UDP MTU 4096
```bash
goos: darwin
goarch: arm64
pkg: github.com/fumiama/WireGold/upper/services/tunnel
cpu: Apple M1
BenchmarkTunnelUDP/1024-plain-nob14-8               4938            228283 ns/op           4.49 MB/s     3642671 B/op        149 allocs/op
BenchmarkTunnelUDP/1024-normal-nob14-8              5100            234118 ns/op           4.37 MB/s     3642409 B/op        147 allocs/op
BenchmarkTunnelUDP/1024-plain-b14-8                 4528            249429 ns/op           4.11 MB/s     3825461 B/op        179 allocs/op
BenchmarkTunnelUDP/1024-normal-b14-8                4885            242048 ns/op           4.23 MB/s     3818262 B/op        175 allocs/op
BenchmarkTunnelUDP/1024-preshared-nob14-8           4833            242460 ns/op           4.22 MB/s     3632537 B/op        144 allocs/op
BenchmarkTunnelUDP/1024-preshared-b14-8             4348            239630 ns/op           4.27 MB/s     3820118 B/op        174 allocs/op
BenchmarkTunnelUDP/2048-plain-nob14-8               4766            280419 ns/op           7.30 MB/s     3656588 B/op        148 allocs/op
BenchmarkTunnelUDP/2048-normal-nob14-8              4353            250150 ns/op           8.19 MB/s     3639053 B/op        145 allocs/op
BenchmarkTunnelUDP/2048-plain-b14-8                 4136            278223 ns/op           7.36 MB/s     3848032 B/op        178 allocs/op
BenchmarkTunnelUDP/2048-normal-b14-8                4264            268694 ns/op           7.62 MB/s     3842609 B/op        176 allocs/op
BenchmarkTunnelUDP/2048-preshared-nob14-8           4154            262575 ns/op           7.80 MB/s     3640443 B/op        144 allocs/op
BenchmarkTunnelUDP/2048-preshared-b14-8             3932            287082 ns/op           7.13 MB/s     3846167 B/op        176 allocs/op
BenchmarkTunnelUDP/3072-plain-nob14-8               4006            267281 ns/op          11.49 MB/s     3690985 B/op        164 allocs/op
BenchmarkTunnelUDP/3072-normal-nob14-8              3942            271832 ns/op          11.30 MB/s     3670827 B/op        162 allocs/op
BenchmarkTunnelUDP/3072-plain-b14-8                 3529            291120 ns/op          10.55 MB/s     3993371 B/op        211 allocs/op
BenchmarkTunnelUDP/3072-normal-b14-8                3614            298778 ns/op          10.28 MB/s     3994267 B/op        211 allocs/op
BenchmarkTunnelUDP/3072-preshared-nob14-8           4036            297819 ns/op          10.31 MB/s     3674026 B/op        162 allocs/op
BenchmarkTunnelUDP/3072-preshared-b14-8             3705            300820 ns/op          10.21 MB/s     3989965 B/op        210 allocs/op
BenchmarkTunnelUDP/4096-plain-nob14-8               2604            398308 ns/op          10.28 MB/s     7389986 B/op        320 allocs/op
BenchmarkTunnelUDP/4096-normal-nob14-8              2744            399739 ns/op          10.25 MB/s     7348911 B/op        316 allocs/op
BenchmarkTunnelUDP/4096-plain-b14-8                 2788            430813 ns/op           9.51 MB/s     7965100 B/op        410 allocs/op
BenchmarkTunnelUDP/4096-normal-b14-8                2620            432984 ns/op           9.46 MB/s     7957374 B/op        407 allocs/op
BenchmarkTunnelUDP/4096-preshared-nob14-8           2750            395736 ns/op          10.35 MB/s     7348747 B/op        315 allocs/op
BenchmarkTunnelUDP/4096-preshared-b14-8             2628            431785 ns/op           9.49 MB/s     7961597 B/op        407 allocs/op
```
### UDP MTU 1024
```bash
goos: darwin
goarch: arm64
pkg: github.com/fumiama/WireGold/upper/services/tunnel
cpu: Apple M1
BenchmarkTunnelUDPSmallMTU/1024-plain-nob14-8       4770            256794 ns/op           3.99 MB/s     3715458 B/op        193 allocs/op
BenchmarkTunnelUDPSmallMTU/1024-normal-nob14-8      4945            242538 ns/op           4.22 MB/s     3681420 B/op        188 allocs/op
BenchmarkTunnelUDPSmallMTU/1024-plain-b14-8         4137            269202 ns/op           3.80 MB/s     4101089 B/op        254 allocs/op
BenchmarkTunnelUDPSmallMTU/1024-normal-b14-8        4592            253461 ns/op           4.04 MB/s     4109262 B/op        253 allocs/op
BenchmarkTunnelUDPSmallMTU/1024-preshared-nob14-8   4764            243752 ns/op           4.20 MB/s     3675691 B/op        186 allocs/op
BenchmarkTunnelUDPSmallMTU/1024-preshared-b14-8     4086            282682 ns/op           3.62 MB/s     4107240 B/op        253 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-plain-nob14-8       4728            252759 ns/op           8.10 MB/s     3762231 B/op        234 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-normal-nob14-8      4245            257036 ns/op           7.97 MB/s     3729842 B/op        232 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-plain-b14-8         3615            308642 ns/op           6.64 MB/s     4469625 B/op        342 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-normal-b14-8        3624            311780 ns/op           6.57 MB/s     4487346 B/op        345 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-preshared-nob14-8   3999            260043 ns/op           7.88 MB/s     3723444 B/op        231 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-preshared-b14-8     3558            315744 ns/op           6.49 MB/s     4476565 B/op        343 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-plain-nob14-8       3814            265654 ns/op          11.56 MB/s     3802900 B/op        280 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-normal-nob14-8      4380            291992 ns/op          10.52 MB/s     3760254 B/op        276 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-plain-b14-8         3340            338760 ns/op           9.07 MB/s     4849826 B/op        434 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-normal-b14-8        3302            345620 ns/op           8.89 MB/s     4852322 B/op        434 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-preshared-nob14-8   4424            265290 ns/op          11.58 MB/s     3761816 B/op        277 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-preshared-b14-8     3148            344490 ns/op           8.92 MB/s     4849613 B/op        434 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-plain-nob14-8       2586            399489 ns/op          10.25 MB/s     7570823 B/op        467 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-normal-nob14-8      2576            402297 ns/op          10.18 MB/s     7504731 B/op        464 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-plain-b14-8         2240            484812 ns/op           8.45 MB/s     9081331 B/op        696 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-normal-b14-8        2240            504749 ns/op           8.11 MB/s     9069168 B/op        693 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-preshared-nob14-8   2594            392716 ns/op          10.43 MB/s     7480678 B/op        460 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-preshared-b14-8     2234            506134 ns/op           8.09 MB/s     9066223 B/op        691 allocs/op
```
### TCP MTU 4096
```bash
goos: darwin
goarch: arm64
pkg: github.com/fumiama/WireGold/upper/services/tunnel
cpu: Apple M1
BenchmarkTunnelTCP/1024-plain-nob14-8               4627            246837 ns/op           4.15 MB/s     3684040 B/op        201 allocs/op
BenchmarkTunnelTCP/1024-normal-nob14-8              4833            257150 ns/op           3.98 MB/s     3682260 B/op        199 allocs/op
BenchmarkTunnelTCP/1024-plain-b14-8                 4396            272838 ns/op           3.75 MB/s     3850134 B/op        231 allocs/op
BenchmarkTunnelTCP/1024-normal-b14-8                4104            252293 ns/op           4.06 MB/s     3844674 B/op        226 allocs/op
BenchmarkTunnelTCP/1024-preshared-nob14-8           4530            264767 ns/op           3.87 MB/s     3680243 B/op        197 allocs/op
BenchmarkTunnelTCP/1024-preshared-b14-8             4231            287111 ns/op           3.57 MB/s     3847164 B/op        227 allocs/op
BenchmarkTunnelTCP/2048-plain-nob14-8               4275            276425 ns/op           7.41 MB/s     3698728 B/op        200 allocs/op
BenchmarkTunnelTCP/2048-normal-nob14-8              4033            261234 ns/op           7.84 MB/s     3701433 B/op        200 allocs/op
BenchmarkTunnelTCP/2048-plain-b14-8                 3680            303246 ns/op           6.75 MB/s     3875541 B/op        231 allocs/op
BenchmarkTunnelTCP/2048-normal-b14-8                3626            288219 ns/op           7.11 MB/s     3878505 B/op        230 allocs/op
BenchmarkTunnelTCP/2048-preshared-nob14-8           3868            287679 ns/op           7.12 MB/s     3696931 B/op        200 allocs/op
BenchmarkTunnelTCP/2048-preshared-b14-8             3586            305008 ns/op           6.71 MB/s     3878416 B/op        230 allocs/op
BenchmarkTunnelTCP/3072-plain-nob14-8               3666            298452 ns/op          10.29 MB/s     3767509 B/op        246 allocs/op
BenchmarkTunnelTCP/3072-normal-nob14-8              3450            304848 ns/op          10.08 MB/s     3761811 B/op        246 allocs/op
BenchmarkTunnelTCP/3072-plain-b14-8                 3549            315641 ns/op           9.73 MB/s     4032830 B/op        291 allocs/op
BenchmarkTunnelTCP/3072-normal-b14-8                3440            327234 ns/op           9.39 MB/s     4038470 B/op        292 allocs/op
BenchmarkTunnelTCP/3072-preshared-nob14-8           3522            302663 ns/op          10.15 MB/s     3760304 B/op        245 allocs/op
BenchmarkTunnelTCP/3072-preshared-b14-8             3390            326384 ns/op           9.41 MB/s     4040489 B/op        293 allocs/op
BenchmarkTunnelTCP/4096-plain-nob14-8               2431            435457 ns/op           9.41 MB/s     7515476 B/op        480 allocs/op
BenchmarkTunnelTCP/4096-normal-nob14-8              2500            433178 ns/op           9.46 MB/s     7511114 B/op        478 allocs/op
BenchmarkTunnelTCP/4096-plain-b14-8                 2337            457177 ns/op           8.96 MB/s     8033760 B/op        568 allocs/op
BenchmarkTunnelTCP/4096-normal-b14-8                2374            465704 ns/op           8.80 MB/s     8040812 B/op        567 allocs/op
BenchmarkTunnelTCP/4096-preshared-nob14-8           2532            436310 ns/op           9.39 MB/s     7510565 B/op        477 allocs/op
BenchmarkTunnelTCP/4096-preshared-b14-8             2360            459261 ns/op           8.92 MB/s     8037878 B/op        566 allocs/op
```
### TCP MTU 1024
```bash
goos: darwin
goarch: arm64
pkg: github.com/fumiama/WireGold/upper/services/tunnel
cpu: Apple M1
BenchmarkTunnelTCPSmallMTU/1024-plain-nob14-8       3318            312084 ns/op           3.28 MB/s     3797015 B/op        307 allocs/op
BenchmarkTunnelTCPSmallMTU/1024-normal-nob14-8      4102            303641 ns/op           3.37 MB/s     3795618 B/op        308 allocs/op
BenchmarkTunnelTCPSmallMTU/1024-plain-b14-8         3746            314102 ns/op           3.26 MB/s     4147318 B/op        368 allocs/op
BenchmarkTunnelTCPSmallMTU/1024-normal-b14-8        3609            315252 ns/op           3.25 MB/s     4152014 B/op        368 allocs/op
BenchmarkTunnelTCPSmallMTU/1024-preshared-nob14-8   3826            300693 ns/op           3.41 MB/s     3793725 B/op        304 allocs/op
BenchmarkTunnelTCPSmallMTU/1024-preshared-b14-8     3628            327852 ns/op           3.12 MB/s     4150869 B/op        367 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-plain-nob14-8       3553            315709 ns/op           6.49 MB/s     3945193 B/op        426 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-normal-nob14-8      3254            329794 ns/op           6.21 MB/s     3933224 B/op        427 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-plain-b14-8         3222            357250 ns/op           5.73 MB/s     4538189 B/op        529 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-normal-b14-8        3080            359401 ns/op           5.70 MB/s     4555108 B/op        535 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-preshared-nob14-8   3463            320078 ns/op           6.40 MB/s     3936771 B/op        426 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-preshared-b14-8     2990            363645 ns/op           5.63 MB/s     4555897 B/op        535 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-plain-nob14-8       3228            336736 ns/op           9.12 MB/s     4090750 B/op        550 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-normal-nob14-8      3076            347067 ns/op           8.85 MB/s     4084480 B/op        554 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-plain-b14-8         2798            395353 ns/op           7.77 MB/s     4952186 B/op        700 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-normal-b14-8        2725            403959 ns/op           7.60 MB/s     4965324 B/op        705 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-preshared-nob14-8   3366            344086 ns/op           8.93 MB/s     4080821 B/op        549 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-preshared-b14-8     2797            403142 ns/op           7.62 MB/s     4962100 B/op        703 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-plain-nob14-8       2360            490867 ns/op           8.34 MB/s     7940290 B/op        871 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-normal-nob14-8      2223            486839 ns/op           8.41 MB/s     7927235 B/op        872 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-plain-b14-8         2002            557560 ns/op           7.35 MB/s     9201342 B/op       1087 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-normal-b14-8        1868            564007 ns/op           7.26 MB/s     9216972 B/op       1091 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-preshared-nob14-8   2263            491698 ns/op           8.33 MB/s     7925404 B/op        869 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-preshared-b14-8     2050            559663 ns/op           7.32 MB/s     9211292 B/op       1086 allocs/op
```
