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

// _udpSessionTimeout is the default timeout for each UDP session.
var _udpSessionTimeout = 60 * time.Second

func SetUDPTimeout(t time.Duration) {
	_udpSessionTimeout = t
}

func newUDPTracker(conn net.PacketConn, metadata *M.Metadata) net.PacketConn {
	return statistic.NewUDPTracker(conn, metadata, statistic.DefaultManager)
}

// TODO: Port Restricted NAT support.
func handleUDPConn(uc adapter.UDPConn) {
	defer uc.Close()

	id := uc.ID()
	metadata := &M.Metadata{
		Network: M.UDP,
		SrcIP:   net.IP(id.RemoteAddress),
		SrcPort: id.RemotePort,
		DstIP:   net.IP(id.LocalAddress),
		DstPort: id.LocalPort,
	}

	pc, err := proxy.DialUDP(metadata)
	if err != nil {
		fmt.Printf("[UDP] dial %s: %v", metadata.DestinationAddress(), err)
		return
	}
	metadata.MidIP, metadata.MidPort = parseAddr(pc.LocalAddr())

	pc = newUDPTracker(pc, metadata)
	defer pc.Close()

	var remote net.Addr
	if udpAddr := metadata.UDPAddr(); udpAddr != nil {
		remote = udpAddr
	} else {
		remote = metadata.Addr()
	}
	pc = newSymmetricNATPacketConn(pc, metadata)

	fmt.Printf("[UDP] %s <-> %s", metadata.SourceAddress(), metadata.DestinationAddress())
	relayPacket(uc, pc, remote)
}

func relayPacket(left net.PacketConn, right net.PacketConn, to net.Addr) {
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := copyPacketBuffer(right, left, to, _udpSessionTimeout); err != nil {
			fmt.Printf("[UDP] %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := copyPacketBuffer(left, right, nil, _udpSessionTimeout); err != nil {
			fmt.Printf("[UDP] %v", err)
		}
	}()

	wg.Wait()
}

func copyPacketBuffer(dst net.PacketConn, src net.PacketConn, to net.Addr, timeout time.Duration) error {
	buf := pool.Get(pool.MaxSegmentSize)
	defer pool.Put(buf)

	for {
		src.SetReadDeadline(time.Now().Add(timeout))
		n, _, err := src.ReadFrom(buf)
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return nil /* ignore I/O timeout */
		} else if err == io.EOF {
			return nil /* ignore EOF */
		} else if err != nil {
			return err
		}

		if _, err = dst.WriteTo(buf[:n], to); err != nil {
			return err
		}
		dst.SetReadDeadline(time.Now().Add(timeout))
	}
}

type symmetricNATPacketConn struct {
	net.PacketConn
	src string
	dst string
}

func newSymmetricNATPacketConn(pc net.PacketConn, metadata *M.Metadata) *symmetricNATPacketConn {
	return &symmetricNATPacketConn{
		PacketConn: pc,
		src:        metadata.SourceAddress(),
		dst:        metadata.DestinationAddress(),
	}
}

func (pc *symmetricNATPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	for {
		n, from, err := pc.PacketConn.ReadFrom(p)

		if from != nil && from.String() != pc.dst {
			fmt.Printf("[UDP] symmetric NAT %s->%s: drop packet from %s", pc.src, pc.dst, from)
			continue
		}

		return n, from, err
	}
}
