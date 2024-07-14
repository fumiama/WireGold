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
Peers:
  -
    IP: "192.168.233.2"
    PublicKey: 徯萃嵾爻燸攗窍褃冔蒔犡緇袿屿組待族砇嘀
    PresharedKey: 瀸敀爅崾嘊嵜紼樴稍毯攣矐訷蟷扛嬋庩崛昀
    EndPoint: 1.2.3.4:56789
    AllowedIPs: ["192.168.233.2/32", "x192.168.233.3/32"] # allow trans to 192.168.233.3, but don not create route
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
    AllowedIPs: ["192.168.233.3/32"]
    MTU: 752
    KeepAliveSeconds: 0
    AllowTrans: false
```
