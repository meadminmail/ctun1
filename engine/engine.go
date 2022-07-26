package engine

import (
	"ctun1/component/dialer"
	"ctun1/core"
	"ctun1/core/option"
	"ctun1/engine/mirror"
	"ctun1/proxy"
	"ctun1/restapi"
	"ctun1/tunnel"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/docker/go-units"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var (
	_engineMu sync.Mutex

	// _defaultKey holds the default key for the engine.
	_defaultKey *Key

	// _defaultStack holds the default stack for the engine.
	_defaultStack *stack.Stack
)

// 加载参数给默认引擎
func Insert(k *Key) {
	_engineMu.Lock()
	_defaultKey = k
	_engineMu.Unlock()
}

// 启动默认引擎
func Start() {
	err := start()
	if err != nil {
		fmt.Println(err)
	}
}

// 停止默认引擎
func Stop() {

	err := stop()
	if err != nil {
		fmt.Println(err)
	}
}

func start() error {
	_engineMu.Lock()
	if _defaultKey == nil {
		return errors.New("empty key")
	}

	err := general(_defaultKey)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = restAPI(_defaultKey)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = netstack(_defaultKey)
	if err != nil {
		fmt.Println(err)
		return err
	}

	_engineMu.Unlock()
	return nil
}

func stop() (err error) {
	err = errors.New("stop err")
	return err
}

func general(k *Key) error {
	if k.Interface != "" {
		iface, err := net.InterfaceByName(k.Interface)
		if err != nil {
			fmt.Println(err)
			return err
		}
		dialer.DefaultInterfaceName.Store(iface.Name)
		dialer.DefaultInterfaceIndex.Store(int32(iface.Index))

	}
	if k.Mark != 0 {
		dialer.DefaultRoutingMark.Store(int32(k.Mark))
	}
	if k.UDPTimeout > 0 {
		if k.UDPTimeout < time.Second {
			return errors.New("无效的udp超时时间")
		}
		tunnel.SetUDPTimeout(k.UDPTimeout)
	}
	return nil
}

func restAPI(k *Key) error {
	if k.RestAPI != "" {
		u, err := parseRestAPI(k.RestAPI)
		if err != nil {
			fmt.Println(err)
			return err
		}
		host, token := u.Host, u.User.String()
		restapi.SetStatsFunc(func() tcpip.Stats {
			_engineMu.Lock()
			defer _engineMu.Unlock()

			// default stack is not initialized.
			if _defaultStack == nil {
				return tcpip.Stats{}
			}
			return _defaultStack.Stats()
		})

		go func() {
			err := restapi.Start(host, token)
			if err != nil {
				fmt.Println(err)
			}
		}()
	}
	return nil
}

func netstack(k *Key) error {
	if k.Proxy == "" {
		return errors.New("empty proxy")
	}
	if k.Device == "" {
		return errors.New("empty device")
	}
	_defaultProxy, err := parseProxy(k.Proxy)
	if err != nil {
		return err
	}
	proxy.SetDialer(_defaultProxy)

	_defaultDevice, err := parseDevice(k.Device, uint32(k.MTU))
	if err != nil {
		return err
	}

	var opts []option.Option
	if k.TCPModerateReceiveBuffer {
		opts = append(opts, option.WithTCPModerateReceiveBuffer(true))
	}

	if k.TCPSendBufferSize != "" {
		size, err := units.RAMInBytes(k.TCPSendBufferSize)
		if err != nil {
			return err
		}
		opts = append(opts, option.WithTCPSendBufferSize(int(size)))
	}

	if k.TCPReceiveBufferSize != "" {
		size, err := units.RAMInBytes(k.TCPReceiveBufferSize)
		if err != nil {
			return err
		}
		opts = append(opts, option.WithTCPReceiveBufferSize(int(size)))
	}

	if _defaultStack, err = core.CreateStack(&core.Config{
		LinkEndpoint:     _defaultDevice,
		TransportHandler: &mirror.Tunnel{},
		PrintFunc: func(format string, v ...any) {
			fmt.Printf("[STACK] %s", fmt.Sprintf(format, v...))
		},
		Options: opts,
	}); err != nil {
		return nil
	}

	fmt.Printf(
		"[STACK] %s://%s <-> %s://%s",
		_defaultDevice.Type(), _defaultDevice.Name(),
		_defaultProxy.Proto(), _defaultProxy.Addr(),
	)
	return nil
}
