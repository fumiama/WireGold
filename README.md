<div align="center">
  <img src=".github/rikka.png" width = "360" height = "360" alt="WireGold-Rikka"><br>
  <h1>WireGold</h1>
  Wire Golang Guard = WireGold<br><br>
</div>

## Usage
> If you are running in windows, remember to select the `wintun.dll` of your arch in `lower/wintun` and place it alongside the compiled exe
```bash
wg [-c config.yaml] [-d|w] [-g] [-h] [-m mtu] [-p]
```
#### Instructions
```bash
  -c string
        specify conf file (default "config.yaml")
  -d    print debug logs
  -g    generate key pair
  -h    display this help
  -m int
        set the mtu of wg (default 32700)
  -p    show my publickey
  -w    only show logs above warn level
```
- *especially*: in macos mojave, max value of mtu is `9159`

## Config file example
```yaml
IP: 192.168.233.1
SubNet: 192.168.233.0/24
PrivateKey: 暲菉斂狧污爉窫擸紈卆帞蔩慈睠庮扝憚瞼縀
EndPoint: 0.0.0.0:56789
Peers:
  -
      IP: "192.168.233.2"
      SubNet: 192.168.233.0/24
      PublicKey: 徯萃嵾爻燸攗窍褃冔蒔犡緇袿屿組待族砇嘀
      EndPoint: that.host2.com:56789
      AllowedIPs: ["192.168.233.2/32"]
      KeepAliveSeconds: 0
      AllowTrans: false
  -
      IP: "192.168.233.3"
      SubNet: 192.168.233.0/24
      PublicKey: 牢喨粷詸衭譛浾蘹櫠砙杹蟫瑳叩刋橋経挵蘀
      EndPoint: that.host3.com:56789
      AllowedIPs: ["192.168.233.3/32"]
      KeepAliveSeconds: 0
      AllowTrans: false
```