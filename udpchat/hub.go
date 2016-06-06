package udpchat

import (
	"errors"
	"log"
	"net"
	"strings"
	"sync"
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
//

type Hub struct {
	// The records in history have variable length, each of them
	// has the format of:
	//
	history []string
	mu      sync.Mutex
	conn    *net.UDPConn
}

type RequestHandler struct {
	recv []byte
	ra   *net.UDPAddr
	hub  *Hub
}

func NewRequestHandler(ra *net.UDPAddr, hub *Hub) *RequestHandler {
	handler := new(RequestHandler)
	handler.recv = make([]byte, 512)
	handler.ra = ra
	handler.hub = hub
	return handler
}

// Serve for the requests from the client.
func (h *RequestHandler) Handle() {
	var err error

	reqType := RequestType(h.recv[0])

	switch reqType {
	case ReqSendChatMsg:
		err = h.handleSndMsg(h.recv[1:])
	case ReqGetHistory:
		err = h.handleHisReq()
	}

	if err != nil {
		log.Println("Server " + err.Error())
		return
	}
}

func (h *RequestHandler) handleSndMsg(msg []byte) error {
	if len(msg) == 0 {
		return errors.New("Empty Message from " + h.ra.String())
	}

	record := string(msg)
	record = time.Now().Format(time.UnixDate) + ": " + record

	h.hub.history = append(h.hub.history, record)
	return nil
}

func (h *RequestHandler) handleHisReq() error {
	logs := strings.Join(h.hub.history, "; ")
	if len(logs) == 0 {
		logs = "No history now."
	}

	log.Println("Sending: [" + logs + "] to " + h.ra.String())
	_, err := h.hub.conn.WriteToUDP([]byte(logs), h.ra)
	return err
}

func (h *Hub) listen() {
	recv := make([]byte, 512)
	for {
		n, ra, err := h.conn.ReadFromUDP(recv)
		if err != nil {
			log.Println("Server " + err.Error())
			continue
		}
		handler := NewRequestHandler(ra, h)
		copy(handler.recv, recv[:n])
		// handle the request in background.
		go handler.Handle()
	}
}

func NewHub() (*Hub, error) {
	hub := new(Hub)
	err := hub.startServer(ServiceHost, ServicePort)
	return hub, err
}

func (h *Hub) startServer(host string, port int) error {
	var err error
	udpaddr := &net.UDPAddr{IP: net.ParseIP(host), Port: port}
	h.conn, err = net.ListenUDP("udp", udpaddr)
	if err != nil {
		return err
	}
	log.Println("Server starts.")
	return nil
}

func (h *Hub) RunLoop() {
	h.listen()
	defer h.conn.Close()
}
