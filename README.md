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
BenchmarkTunnelUDP/1024-plain-nob14-8               4753            250629 ns/op           4.09 MB/s     3570503 B/op        117 allocs/op
BenchmarkTunnelUDP/1024-normal-nob14-8              4473            261136 ns/op           3.92 MB/s     3570565 B/op        118 allocs/op
BenchmarkTunnelUDP/1024-plain-b14-8                 4250            275495 ns/op           3.72 MB/s     3575369 B/op        121 allocs/op
BenchmarkTunnelUDP/1024-normal-b14-8                4204            278062 ns/op           3.68 MB/s     3575844 B/op        122 allocs/op
BenchmarkTunnelUDP/1024-preshared-nob14-8           4196            282707 ns/op           3.62 MB/s     3570427 B/op        118 allocs/op
BenchmarkTunnelUDP/1024-preshared-b14-8             4130            290609 ns/op           3.52 MB/s     3575814 B/op        121 allocs/op
BenchmarkTunnelUDP/2048-plain-nob14-8               4162            280754 ns/op           7.29 MB/s     3578825 B/op        117 allocs/op
BenchmarkTunnelUDP/2048-normal-nob14-8              4088            292733 ns/op           7.00 MB/s     3578858 B/op        118 allocs/op
BenchmarkTunnelUDP/2048-plain-b14-8                 4023            290264 ns/op           7.06 MB/s     3589002 B/op        121 allocs/op
BenchmarkTunnelUDP/2048-normal-b14-8                3904            298841 ns/op           6.85 MB/s     3589116 B/op        122 allocs/op
BenchmarkTunnelUDP/2048-preshared-nob14-8           3861            285661 ns/op           7.17 MB/s     3578690 B/op        117 allocs/op
BenchmarkTunnelUDP/2048-preshared-b14-8             4344            273054 ns/op           7.50 MB/s     3589101 B/op        122 allocs/op
BenchmarkTunnelUDP/3072-plain-nob14-8               4293            273633 ns/op          11.23 MB/s     3582928 B/op        121 allocs/op
BenchmarkTunnelUDP/3072-normal-nob14-8              4086            287291 ns/op          10.69 MB/s     3583011 B/op        122 allocs/op
BenchmarkTunnelUDP/3072-plain-b14-8                 4144            295141 ns/op          10.41 MB/s     3600854 B/op        128 allocs/op
BenchmarkTunnelUDP/3072-normal-b14-8                3759            310645 ns/op           9.89 MB/s     3601217 B/op        130 allocs/op
BenchmarkTunnelUDP/3072-preshared-nob14-8           3805            305162 ns/op          10.07 MB/s     3583013 B/op        122 allocs/op
BenchmarkTunnelUDP/3072-preshared-b14-8             3574            320329 ns/op           9.59 MB/s     3601409 B/op        130 allocs/op
BenchmarkTunnelUDP/4096-plain-nob14-8               2448            482246 ns/op           8.49 MB/s     7157679 B/op        241 allocs/op
BenchmarkTunnelUDP/4096-normal-nob14-8              2328            508454 ns/op           8.06 MB/s     7158430 B/op        244 allocs/op
BenchmarkTunnelUDP/4096-plain-b14-8                 2290            527956 ns/op           7.76 MB/s     7181296 B/op        254 allocs/op
BenchmarkTunnelUDP/4096-normal-b14-8                2276            528681 ns/op           7.75 MB/s     7181960 B/op        256 allocs/op
BenchmarkTunnelUDP/4096-preshared-nob14-8           2284            515491 ns/op           7.95 MB/s     7158606 B/op        243 allocs/op
BenchmarkTunnelUDP/4096-preshared-b14-8             2026            560506 ns/op           7.31 MB/s     7181980 B/op        256 allocs/op
```
### UDP MTU 1024
```bash
goos: darwin
goarch: arm64
pkg: github.com/fumiama/WireGold/upper/services/tunnel
cpu: Apple M1
BenchmarkTunnelUDPSmallMTU/1024-plain-nob14-8       3766            326631 ns/op           3.14 MB/s     3568299 B/op        126 allocs/op
BenchmarkTunnelUDPSmallMTU/1024-normal-nob14-8      3728            319552 ns/op           3.20 MB/s     3568454 B/op        128 allocs/op
BenchmarkTunnelUDPSmallMTU/1024-plain-b14-8         3625            323816 ns/op           3.16 MB/s     3575638 B/op        137 allocs/op
BenchmarkTunnelUDPSmallMTU/1024-normal-b14-8        3446            333510 ns/op           3.07 MB/s     3575925 B/op        138 allocs/op
BenchmarkTunnelUDPSmallMTU/1024-preshared-nob14-8   3537            335220 ns/op           3.05 MB/s     3568481 B/op        128 allocs/op
BenchmarkTunnelUDPSmallMTU/1024-preshared-b14-8     3486            337967 ns/op           3.03 MB/s     3575890 B/op        138 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-plain-nob14-8       3303            349278 ns/op           5.86 MB/s     3592804 B/op        140 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-normal-nob14-8      3331            376226 ns/op           5.44 MB/s     3593065 B/op        142 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-plain-b14-8         3116            421630 ns/op           4.86 MB/s     3605117 B/op        157 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-normal-b14-8        2634            381676 ns/op           5.37 MB/s     3606455 B/op        158 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-preshared-nob14-8   3138            391760 ns/op           5.23 MB/s     3591788 B/op        142 allocs/op
BenchmarkTunnelUDPSmallMTU/2048-preshared-b14-8     2959            391663 ns/op           5.23 MB/s     3605364 B/op        158 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-plain-nob14-8       3046            421705 ns/op           7.28 MB/s     3620443 B/op        156 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-normal-nob14-8      3001            413043 ns/op           7.44 MB/s     3631990 B/op        157 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-plain-b14-8         2503            406906 ns/op           7.55 MB/s     3640574 B/op        177 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-normal-b14-8        2776            416946 ns/op           7.37 MB/s     3643066 B/op        179 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-preshared-nob14-8   2947            422378 ns/op           7.27 MB/s     3626004 B/op        157 allocs/op
BenchmarkTunnelUDPSmallMTU/3072-preshared-b14-8     2547            459951 ns/op           6.68 MB/s     3648033 B/op        179 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-plain-nob14-8       1776            628904 ns/op           6.51 MB/s     7232490 B/op        285 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-normal-nob14-8      1782            643967 ns/op           6.36 MB/s     7238574 B/op        288 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-plain-b14-8         1549            674359 ns/op           6.07 MB/s     7262233 B/op        317 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-normal-b14-8        1826            690961 ns/op           5.93 MB/s     7260027 B/op        319 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-preshared-nob14-8   1868            649732 ns/op           6.30 MB/s     7242787 B/op        288 allocs/op
BenchmarkTunnelUDPSmallMTU/4096-preshared-b14-8     1654            682244 ns/op           6.00 MB/s     7255985 B/op        318 allocs/op
```
### TCP MTU 4096
```bash
goos: darwin
goarch: arm64
pkg: github.com/fumiama/WireGold/upper/services/tunnel
cpu: Apple M1
BenchmarkTunnelTCP/1024-plain-nob14-8               2323            459188 ns/op           2.23 MB/s     3576540 B/op        166 allocs/op
BenchmarkTunnelTCP/1024-normal-nob14-8              2472            438347 ns/op           2.34 MB/s     3576692 B/op        168 allocs/op
BenchmarkTunnelTCP/1024-plain-b14-8                 2728            418395 ns/op           2.45 MB/s     3583603 B/op        171 allocs/op
BenchmarkTunnelTCP/1024-normal-b14-8                2668            463060 ns/op           2.21 MB/s     3584519 B/op        172 allocs/op
BenchmarkTunnelTCP/1024-preshared-nob14-8           2660            454945 ns/op           2.25 MB/s     3576708 B/op        168 allocs/op
BenchmarkTunnelTCP/1024-preshared-b14-8             2690            437373 ns/op           2.34 MB/s     3584515 B/op        172 allocs/op
BenchmarkTunnelTCP/2048-plain-nob14-8               2580            455416 ns/op           4.50 MB/s     3590368 B/op        168 allocs/op
BenchmarkTunnelTCP/2048-normal-nob14-8              2294            458178 ns/op           4.47 MB/s     3590512 B/op        171 allocs/op
BenchmarkTunnelTCP/2048-plain-b14-8                 2414            462412 ns/op           4.43 MB/s     3605344 B/op        174 allocs/op
BenchmarkTunnelTCP/2048-normal-b14-8                2511            508527 ns/op           4.03 MB/s     3605658 B/op        177 allocs/op
BenchmarkTunnelTCP/2048-preshared-nob14-8           2433            482086 ns/op           4.25 MB/s     3590571 B/op        171 allocs/op
BenchmarkTunnelTCP/2048-preshared-b14-8             2361            494409 ns/op           4.14 MB/s     3605739 B/op        177 allocs/op
BenchmarkTunnelTCP/3072-plain-nob14-8               2487            498395 ns/op           6.16 MB/s     3600311 B/op        199 allocs/op
BenchmarkTunnelTCP/3072-normal-nob14-8              2170            542424 ns/op           5.66 MB/s     3600596 B/op        202 allocs/op
BenchmarkTunnelTCP/3072-plain-b14-8                 2259            524854 ns/op           5.85 MB/s     3621274 B/op        205 allocs/op
BenchmarkTunnelTCP/3072-normal-b14-8                2307            537656 ns/op           5.71 MB/s     3621514 B/op        209 allocs/op
BenchmarkTunnelTCP/3072-preshared-nob14-8           1855            545493 ns/op           5.63 MB/s     3600418 B/op        201 allocs/op
BenchmarkTunnelTCP/3072-preshared-b14-8             2198            535328 ns/op           5.74 MB/s     3621536 B/op        208 allocs/op
BenchmarkTunnelTCP/4096-plain-nob14-8               2043            587272 ns/op           6.97 MB/s     7181814 B/op        391 allocs/op
BenchmarkTunnelTCP/4096-normal-nob14-8              1832            609909 ns/op           6.72 MB/s     7182940 B/op        394 allocs/op
BenchmarkTunnelTCP/4096-plain-b14-8                 2044            572149 ns/op           7.16 MB/s     7209279 B/op        405 allocs/op
BenchmarkTunnelTCP/4096-normal-b14-8                2019            655180 ns/op           6.25 MB/s     7210261 B/op        409 allocs/op
BenchmarkTunnelTCP/4096-preshared-nob14-8           1652            636402 ns/op           6.44 MB/s     7182914 B/op        394 allocs/op
BenchmarkTunnelTCP/4096-preshared-b14-8             1885            624237 ns/op           6.56 MB/s     7210327 B/op        408 allocs/op
```
### TCP MTU 1024
```bash
goos: darwin
goarch: arm64
pkg: github.com/fumiama/WireGold/upper/services/tunnel
cpu: Apple M1
BenchmarkTunnelTCPSmallMTU/1024-plain-nob14-8       2061            582289 ns/op           1.76 MB/s     3577539 B/op        234 allocs/op
BenchmarkTunnelTCPSmallMTU/1024-normal-nob14-8      2172            561002 ns/op           1.83 MB/s     3577725 B/op        237 allocs/op
BenchmarkTunnelTCPSmallMTU/1024-plain-b14-8         2002            625224 ns/op           1.64 MB/s     3584694 B/op        244 allocs/op
BenchmarkTunnelTCPSmallMTU/1024-normal-b14-8        1957            590091 ns/op           1.74 MB/s     3585060 B/op        247 allocs/op
BenchmarkTunnelTCPSmallMTU/1024-preshared-nob14-8   2127            552614 ns/op           1.85 MB/s     3577669 B/op        236 allocs/op
BenchmarkTunnelTCPSmallMTU/1024-preshared-b14-8     2084            602128 ns/op           1.70 MB/s     3585057 B/op        247 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-plain-nob14-8       1899            595303 ns/op           3.44 MB/s     3596277 B/op        320 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-normal-nob14-8      1656            604450 ns/op           3.39 MB/s     3596115 B/op        323 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-plain-b14-8         1729            624733 ns/op           3.28 MB/s     3610414 B/op        339 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-normal-b14-8        1568            653317 ns/op           3.13 MB/s     3611234 B/op        342 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-preshared-nob14-8   1858            664597 ns/op           3.08 MB/s     3595764 B/op        322 allocs/op
BenchmarkTunnelTCPSmallMTU/2048-preshared-b14-8     1404            767077 ns/op           2.67 MB/s     3609789 B/op        339 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-plain-nob14-8       1761            846583 ns/op           3.63 MB/s     3614569 B/op        410 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-normal-nob14-8      1887            743407 ns/op           4.13 MB/s     3612869 B/op        411 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-plain-b14-8         1582            679431 ns/op           4.52 MB/s     3639650 B/op        435 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-normal-b14-8        1688            720574 ns/op           4.26 MB/s     3634744 B/op        435 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-preshared-nob14-8   1762            731901 ns/op           4.20 MB/s     3616570 B/op        414 allocs/op
BenchmarkTunnelTCPSmallMTU/3072-preshared-b14-8     1656            716281 ns/op           4.29 MB/s     3636078 B/op        434 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-plain-nob14-8       1482            847378 ns/op           4.83 MB/s     7214173 B/op        666 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-normal-nob14-8      1354            818199 ns/op           5.01 MB/s     7219760 B/op        665 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-plain-b14-8         1557            784260 ns/op           5.22 MB/s     7243407 B/op        697 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-normal-b14-8        1316            811760 ns/op           5.05 MB/s     7241275 B/op        699 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-preshared-nob14-8   1299            806369 ns/op           5.08 MB/s     7216648 B/op        666 allocs/op
BenchmarkTunnelTCPSmallMTU/4096-preshared-b14-8     1278            858201 ns/op           4.77 MB/s     7242324 B/op        703 allocs/op
```
