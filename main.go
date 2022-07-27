package main

import (
	_ "ctun1/dns"
	"ctun1/engine"
	"flag"
	"fmt"
)

var (
	key = new(engine.Key)

	configFile  string
	versionFlag bool
)

/** 初始化 */
func init() {

	flag.IntVar(&key.Mark, "fwmark", 0, "Set firewall MARK (Linux only)")
	flag.IntVar(&key.MTU, "mtu", 0, "Set device maximum transmission unit (MTU)")
	flag.DurationVar(&key.UDPTimeout, "udp-timeout", 0, "Set timeout for each UDP session")
	flag.StringVar(&configFile, "config", "", "YAML format configuration file")
	flag.StringVar(&key.Device, "device", "wintun", "Use this device [driver://]name")
	flag.StringVar(&key.Interface, "interface", "", "Use network INTERFACE (Linux/MacOS only)")
	flag.StringVar(&key.LogLevel, "loglevel", "info", "Log level [debug|info|warning|error|silent]")
	flag.StringVar(&key.Proxy, "proxy", "192.168.74.128:8000", "Use this proxy [protocol://]host[:port]")
	flag.StringVar(&key.RestAPI, "restapi", "", "HTTP statistic server listen address")
	flag.StringVar(&key.TCPSendBufferSize, "tcp-sndbuf", "", "Set TCP send buffer size for netstack")
	flag.StringVar(&key.TCPReceiveBufferSize, "tcp-rcvbuf", "", "Set TCP receive buffer size for netstack")
	flag.BoolVar(&key.TCPModerateReceiveBuffer, "tcp-auto-tuning", false, "Enable TCP receive buffer auto-tuning")
	flag.BoolVar(&versionFlag, "version", false, "Show version and then quit")
	flag.Parse()
}

/** 入口函数 */
func main() {
	fmt.Println("ctun1 start ...")

	engine.Insert(key)

	engine.Start()
	defer engine.Stop()
}
