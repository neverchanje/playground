package udpchat

import (
	"log"
	"net"
	"strings"
	"time"
)

// One goroutine for each connection.
// For each message the server receives, logs it to history
// that's maintained by the server. The clients will not obtain the realtime
// chat history until it requests for it.
// TODO use some persistent database to store chat history, the server
// should only keep the latest logs in memory, and stores the older logs
// into disks. But on the other hand it could also brings many problems like
// indexing of logs.
// TODO the client should store a local cache of the chat history, so that it
// do not have to make requests every time.
// TODO connect in TCP but chat in UDP.
// TODO limit the allowed number of bytes to send.
//

type Conn struct {
	udpconn *net.UDPConn
	recv    []byte
	hub     *Hub
}

type Hub struct {
	conns   map[*Conn]bool
	history []string
}

func NewConn(c *net.UDPConn) *Conn {
	conn := new(Conn)
	conn.recv = make([]byte, 512)
	conn.udpconn = c
	return conn
}

// abandon this connection, it doesn't affect
// other clients.
func (h *Hub) abandon(c *Conn) {
	c.Close()
	delete(c.hub.conns, c)
}

// Serve for the requests from the client.
func (c *Conn) handleClient() {
	for {
		n, err := c.udpconn.Read(c.recv)
		if err != nil {
			log.Println(err)
			c.hub.abandon(c)
			break
		}

		switch RequestType(c.recv[0]) {
		case ReqSendChatMsg:
			err = c.handleSndMsg(c.recv[1:n])
		case ReqGetHistory:
			err = c.handleHisReq()
		}

		if err != nil {
			log.Println(err)
			c.hub.abandon(c)
			break
		}
	}
}

func (c *Conn) handleSndMsg(msg []byte) error {
	record := string(msg)
	record = time.Now().Format(time.UnixDate) + record
	c.hub.history = append(c.hub.history, record)
	return nil
}

func (c *Conn) handleHisReq() error {
	logs := strings.Join(c.hub.history, "\n")
	_, err := c.udpconn.Write([]byte(logs))

	return err
}

func (c *Conn) Close() {
	c.udpconn.Close()
}

func (h *Hub) listen(host string, port int) error {
	udpaddr := &net.UDPAddr{IP: net.ParseIP(host), Port: port}
	c, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		return err
	}

	conn := NewConn(c)
	h.conns[conn] = true
	// per goroutine per connection
	go conn.handleClient()
	return nil
}

func NewHub() (*Hub, error) {
	hub := new(Hub)
	hub.conns = make(map[*Conn]bool)
	hub.history = make([]string, 5)
	return hub, nil
}

// Close all opening connection.
func (h *Hub) Close() {
	for c, _ := range h.conns {
		h.abandon(c)
	}
}

func (h *Hub) RunLoop() {
	for {
		h.listen(ServiceHost, ServicePort)
	}
}
