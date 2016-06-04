/**
 * Copyright (C) 2016, Wu Tao All rights reserved.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	// we are not allowed to use iana.ProtocolICMP since it's in
	// internal package
	ProtocolICMP = 1
)

type Pinger struct {
	// The identifier and sequence number can be used by the client to
	// determine which echo requests are associated with the echo replies.
	id  int
	seq int

	peer   *net.IPAddr
	MaxRTT time.Duration
}

type packet struct {
	bytes []byte
	addr  *net.Addr
}

func NewPinger(hostname string) (*Pinger, error) {
	pinger := new(Pinger)
	pinger.MaxRTT = time.Second
	err := pinger.Register(hostname)
	return pinger, err
}

func timeToBytes(t time.Time) []byte {
	nsec := t.UnixNano()
	b := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return b
}

func bytesToTime(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < 8; i++ {
		nsec += int64(b[i]) << ((7 - i) * 8)
	}
	return time.Unix(nsec/1000000000, nsec%1000000000)
}

// Loops until the user manually terminates it.
// The main loop processes received packets and timing as events(aka goroutines).
// It sends an ICMP packet to host once per second(MaxRTT), and calculates the
// RTT when the packet received.
func (p *Pinger) RunLoop() error {
	var conn *icmp.PacketConn
	var err error

	if conn, err = icmp.ListenPacket("udp4", ""); conn == nil || err != nil {
		return err
	}
	defer conn.Close()

	// register a channel, that signals once packet received.
	// the "receiver" runs in another loop.
	recv := make(chan *packet, 1)
	go p.recvICMP(conn, recv)

	// TODO what error could happen?
	err = p.sendICMP(conn)

	ticker := time.NewTicker(p.MaxRTT)

	for {
		select {
		case <-ticker.C:
			// time exceeds
			p.sendICMP(conn)
		case r := <-recv:
			// packet received
			// procesing packet
			p.procICMP(r)
		}
	}
	return nil
}

func (p *Pinger) Register(peerHostName string) (err error) {
	p.peer, err = net.ResolveIPAddr("ip4", peerHostName)
	fmt.Println("Registering: " + p.peer.String())
	return err
}

func (p *Pinger) RegisterIPAddr(peer *net.IPAddr) {
	p.peer = peer
}

func (p *Pinger) RegisterIP(peer string) {
	ip := net.ParseIP(peer)
	if ip != nil {
		p.peer = &net.IPAddr{IP: ip}
	}
}

func (p *Pinger) sendICMP(conn *icmp.PacketConn) (err error) {

	var b []byte

	p.id = rand.Intn(0xffff)
	p.seq = rand.Intn(0xffff)

	// icmp.Echo has implemented the interface of icmp.MessageBody
	// checksum field of icmp.Message will be calculated in icmp.Message.Mashal()
	b, err = (&icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{ID: p.id, Seq: p.seq, Data: timeToBytes(time.Now())},
	}).Marshal(nil)
	if err != nil {
		return
	}

	dst := &net.UDPAddr{IP: p.peer.IP}
	_, err = conn.WriteTo(b, dst)

	fmt.Println("Sending packet to " + dst.String())
	return
}

// chan<-: can only be used to send
func (p *Pinger) recvICMP(conn *icmp.PacketConn, recv chan<- *packet) (err error) {

	for {
		var remote net.Addr
		// read buffer
		b := make([]byte, 512)

		// wait for 3 seconds if no packets are come
		conn.SetReadDeadline(time.Now().Add(time.Second * 3))
		if _, remote, err = conn.ReadFrom(b); err != nil {
			return
		}

		fmt.Println("Reading from remote address: " + remote.String())

		// signal the receiver
		recv <- &packet{bytes: b, addr: &remote}
	}

	return
}

func (p *Pinger) procICMP(pac *packet) (err error) {
	var m *icmp.Message
	if m, err = icmp.ParseMessage(ProtocolICMP, pac.bytes); err != nil {
		log.Println(err.Error())
		return
	}

	fmt.Println("Processing packet")

	var rtt time.Duration
	// .(type) can only be used within switch cace.
	switch pkt := m.Body.(type) {
	case *icmp.Echo:
		if pkt.ID == p.id && pkt.Seq == p.seq {
			rtt = time.Since(bytesToTime(pkt.Data[:8]))
		}
		fmt.Printf("%s : %v %v\n\n", p.peer.String(), rtt, time.Now())
	default:
		return
	}

	return
}

func main() {

	hostname := flag.String("hostname", "",
		"Example: ./ping -hostname=www.example.com")

	flag.Parse()

	// hostname must be provided
	if len(*hostname) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	pinger, err := NewPinger(*hostname)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	pinger.RunLoop()
}
