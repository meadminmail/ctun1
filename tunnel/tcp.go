package tunnel

import (
	"ctun1/common/pool"
	"ctun1/core/adapter"
	M "ctun1/metadata"
	"ctun1/proxy"
	"ctun1/tunnel/statistic"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	tcpWaitTimeout = 5 * time.Second
)

func newTCPTracker(conn net.Conn, metadata *M.Metadata) net.Conn {
	return statistic.NewTCPTracker(conn, metadata, statistic.DefaultManager)
}

func handleTCPConn(localConn adapter.TCPConn) {
	defer localConn.Close()

	id := localConn.ID()
	metadata := &M.Metadata{
		Network: M.TCP,
		SrcIP:   net.IP(id.RemoteAddress),
		SrcPort: id.RemotePort,
		DstIP:   net.IP(id.LocalAddress),
		DstPort: id.LocalPort,
	}

	targetConn, err := proxy.Dial(metadata)
	if err != nil {
		fmt.Printf("[TCP] dial %s: %v", metadata.DestinationAddress(), err)
		return
	}
	metadata.MidIP, metadata.MidPort = parseAddr(targetConn.LocalAddr())

	targetConn = newTCPTracker(targetConn, metadata)
	defer targetConn.Close()

	fmt.Printf("[TCP] %s <-> %s", metadata.SourceAddress(), metadata.DestinationAddress())
	relay(localConn, targetConn) /* relay connections */
}

// relay copies between left and right bidirectionally.
func relay(left, right net.Conn) {
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := copyBuffer(right, left); err != nil {
			fmt.Printf("[TCP] %v", err)
		}
		right.SetReadDeadline(time.Now().Add(tcpWaitTimeout))
	}()

	go func() {
		defer wg.Done()
		if err := copyBuffer(left, right); err != nil {
			fmt.Printf("[TCP] %v", err)
		}
		left.SetReadDeadline(time.Now().Add(tcpWaitTimeout))
	}()

	wg.Wait()
}

func copyBuffer(dst io.Writer, src io.Reader) error {
	buf := pool.Get(pool.RelayBufferSize)
	defer pool.Put(buf)

	_, err := io.CopyBuffer(dst, src, buf)
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return nil /* ignore I/O timeout */
	}
	return err
}
