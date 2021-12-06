package pingser

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var (
	ipv4Proto    string        = "ip4:icmp"
	defaultDelay time.Duration = time.Millisecond * 100
)

var (
	ErrNoConnection = errors.New("icmp no connection")
)

type Packet struct {
	ID   int
	Seq  int
	Data []byte
	Src  *net.IPAddr
}

type Pingser struct {
	done chan struct{}
	conn *icmp.PacketConn

	id          int
	isServerMod bool

	// client mode uses these
	sequence int
	addr     string
	ipaddr   net.Addr

	// OnRecv is called when Pingser receives and processes a packet
	OnRecv func(*Packet)
}

func NewClient(addr string) (*Pingser, error) {
	client := &Pingser{
		id:   getid(),
		done: make(chan struct{}),
		addr: addr,
	}
	return client, client.resolve()
}

func NewServer() *Pingser {
	return &Pingser{
		id:          getid(),
		done:        make(chan struct{}),
		isServerMod: true,
	}
}

func (p *Pingser) Listen() error {
	conn, err := icmp.ListenPacket(ipv4Proto, "")
	if err != nil {
		return err
	}

	p.conn = conn
	return nil
}

func (p *Pingser) Run() error {
	if p.conn == nil {
		return ErrNoConnection
	}

	recvCh := make(chan *Packet, 5)
	defer close(recvCh)

	go func() {
		defer p.Close()
		if err := p.recvICMP(recvCh); err != nil {
			return
		}
	}()

	for {
		select {
		case <-p.done:
			return nil
		case pkt := <-recvCh:
			if p.OnRecv != nil {
				p.OnRecv(pkt)
			}
		}
	}
}

func (p *Pingser) Send(data []byte, pkt ...*Packet) error {
	if p.conn == nil {
		return ErrNoConnection
	}
	var (
		body   *icmp.Echo
		ipaddr net.Addr
	)

	if !p.isServerMod {
		body = &icmp.Echo{
			ID:   p.id,
			Seq:  p.sequence,
			Data: data,
		}
		ipaddr = p.ipaddr
	} else {
		if len(pkt) <= 0 {
			return errors.New("the 'pkt' parameter is missing from server mode")
		}
		body = &icmp.Echo{
			ID:   pkt[0].ID,
			Seq:  pkt[0].Seq,
			Data: data,
		}
		ipaddr = pkt[0].Src
	}
	return p.sendICMP(body, ipaddr)
}

func (p *Pingser) Close() {
	open := true
	select {
	case _, open = <-p.done:
	default:
	}

	if open {
		close(p.done)
	}

	if p.conn != nil {
		p.conn.Close()
	}
}

func (p *Pingser) sendICMP(body icmp.MessageBody, toaddr net.Addr) error {
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: body,
	}
	if p.isServerMod {
		msg.Type = ipv4.ICMPTypeEchoReply
	}

	msgData, err := msg.Marshal(nil)
	if err != nil {
		return err
	}
	if _, err := p.conn.WriteTo(msgData, toaddr); err != nil {
		if neterr, ok := err.(*net.OpError); ok {
			if neterr.Err == syscall.ENOBUFS {
				return nil
			}
		}
		return err
	}
	p.sequence++
	if p.sequence > 65535 {
		p.sequence = 0
	}
	return nil
}

func (p *Pingser) recvICMP(recv chan<- *Packet) error {
	delay := defaultDelay
	for {
		select {
		case <-p.done:
			return nil
		default:
			if err := p.conn.SetReadDeadline(time.Now().Add(delay)); err != nil {
				return err
			}
			// TODO data size
			data := make([]byte, 1024)
			n, srcaddr, err := p.conn.ReadFrom(data)
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						// Read timeout
						delay = time.Second
						continue
					}
				}
				return err
			} else {
				delay = defaultDelay
			}

			if n <= 0 {
				continue
			}

			pkt, err := p.processPacket(data[:])
			if err != nil || pkt == nil {
				fmt.Printf("recvIcmp - processPacket error:%s\n", err)
				continue
			}
			pkt.Src = srcaddr.(*net.IPAddr)

			select {
			case <-p.done:
				return nil
			case recv <- pkt:
			}
		}
	}
}

func (p *Pingser) processPacket(bytes []byte) (*Packet, error) {
	var (
		m   *icmp.Message
		err error
	)
	if m, err = icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), bytes); err != nil {
		return nil, fmt.Errorf("error parsing icmp message: %w", err)
	}

	if p.isServerMod {
		if m.Type != ipv4.ICMPTypeEcho {
			return nil, nil
		}
	} else {
		if m.Type != ipv4.ICMPTypeEchoReply {
			return nil, nil
		}
	}

	switch pkt := m.Body.(type) {
	case *icmp.Echo:
		if !p.isServerMod && !p.matchID(pkt.ID) {
			return nil, nil
		}

		return &Packet{
			ID:   pkt.ID,
			Seq:  pkt.Seq,
			Data: pkt.Data,
		}, nil
	default:
		return nil, fmt.Errorf("invalid ICMP echo reply; type: '%T', '%v'", pkt, pkt)
	}
}

func (p *Pingser) resolve() error {
	if len(p.addr) == 0 {
		return errors.New("addr cannot be empty")
	}
	ipaddr, err := net.ResolveIPAddr("ip", p.addr)
	if err != nil {
		return err
	}

	p.ipaddr = ipaddr
	return nil
}

func (p *Pingser) matchID(ID int) bool {
	return ID == p.id
}

func getid() int {
	return os.Getegid() & 0xffff
}
