package udpchat

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"os"
	"strconv"
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
// TODO the client should keep a local cache of the chat history, so that it
// do not have to make requests every time.
// TODO connect in TCP but chat in UDP.
// TODO (HARD) each client can subscribe and publish for multiple channels,
// they can type "channels" for information of the current existing channels,
// and type "subscribe: <channel>" for subscribing specific channels so that
// server can reply with the histories in those channels when users request for.
// To complete this requirement, messages to be sent must start with the
// destination channel, like: "send: <channel-id>/<channel-name> <message>".
// Furthermore, users can type "mkchannel: <channel-name>" to create a new channel,
// server must return a global unique channel-id if created successfully.
// Also, only users who created the channel has the authorization to delete it (
// "rmchannel: <channel-id>")

type Hub struct {
	// The records in history have arbitrary length, each of them
	// has the format of:
	// <Date> <Message>
	history []string
	mu      sync.Mutex
	conn    *net.UDPConn

	fileHandlers map[uint64]*RequestHandler
}

type RequestHandler struct {
	ra  *net.UDPAddr
	hub *Hub

	receiver *fileReceiver
}

type fileReceiver struct {
	accepted     map[uint32](*fileSegment)
	mu           sync.Mutex
	fname        string
	fileListener chan bool // indicates iff file segments are completely received
	packet_id    uint64
}

// REQUIRE: mutex lock held
func (fr *fileReceiver) isComplete() bool {

	if len(fr.accepted) == 0 {
		return false
	}

	max_id := uint32(0)
	for id, _ := range fr.accepted {
		if id > max_id {
			max_id = id
		}
	}

	l := uint32(len(fr.accepted))
	if l != (max_id + 1) {
		return false
	}

	return len(fr.accepted[l-1].content) == 0
}

// REQUIRE: mutex lock held
func (fr *fileReceiver) insert(seg *fileSegment) {
	fr.accepted[seg.seg_id] = seg
	log.Println("Receiving segment " + seg.toString())
}

type fileSegment struct {
	packet_id uint64
	seg_id    uint32
	content   []byte
	seg_len   int
}

func newFileSegment(seg []byte) *fileSegment {
	fs := new(fileSegment)
	fs.packet_id = binary.LittleEndian.Uint64(seg[1:9])
	fs.seg_id = binary.LittleEndian.Uint32(seg[9:13])
	fs.content = make([]byte, len(seg[13:]))
	copy(fs.content, seg[13:])
	fs.seg_len = len(seg)
	return fs
}

func (fs *fileSegment) toString() string {
	str := "<PACKET_ID: " + strconv.FormatUint(fs.packet_id, 10)
	str += " , SEG_ID: " + strconv.FormatUint(uint64(fs.seg_id), 10)
	str += " , SEG_LEN: " + strconv.FormatInt(int64(fs.seg_len), 10)
	str += ">"
	return str
}

func NewRequestHandler(ra *net.UDPAddr, hub *Hub) *RequestHandler {
	handler := new(RequestHandler)
	handler.ra = ra
	handler.hub = hub
	return handler
}

// Serve for the requests from the client. There may have mutiple clients
// concurrently handling the requests.
func (h *RequestHandler) Handle(recv []byte) {
	var err error

	reqType := RequestType(recv[0])

	switch reqType {
	case kReqSendChatMsg:
		err = h.handleSndMsg(recv)
	case kReqGetHistory:
		err = h.handleHisReq()
	case kReqSendFile:
		err = h.handleSendFile(recv)
	case kReqSendSeg:
		err = h.receiver.handleSendSegment(recv)
	}

	if err != nil {
		log.Println("Server " + err.Error())
		return
	}
}

func (h *RequestHandler) handleSendFile(recv []byte) error {
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(kRespSendFileOK))
	_, err := h.hub.conn.WriteToUDP(buf.Bytes(), h.ra)

	if err == nil {
		go h.collectSegments()

		log.Println("File transferring from remote: " + h.ra.String())
		h.receiver = new(fileReceiver)
		h.receiver.fname = string(recv[9:])
		h.receiver.accepted = make(map[uint32]*fileSegment)
		h.receiver.packet_id = binary.LittleEndian.Uint64(recv[1:9])
		h.receiver.fileListener = make(chan bool)
		h.hub.fileHandlers[h.receiver.packet_id] = h
	}
	return err
}

func (h *RequestHandler) collectSegments() {
loop:
	for {
		select {
		case <-h.receiver.fileListener:
			err := h.writeFile()
			if err == nil {
				delete(h.hub.fileHandlers, h.receiver.packet_id)
				h.appendHistorys("Sending file " + h.receiver.fname)
				//h.receiver = nil // do GC
				break
			}
			log.Println("Failed to write file: " + h.receiver.fname + " " + err.Error())
			break loop
		}
	}
}

func (fr *fileReceiver) handleSendSegment(recv []byte) error {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	fr.insert(newFileSegment(recv))

	if fr.isComplete() {
		log.Println("All segments are received")
		fr.fileListener <- true
		return nil
	}
	return nil
}

// TODO 当前版本为一次性写，应该每段写一个分文件，最终集合成大文件
func (h *RequestHandler) writeFile() error {
	file, err := os.Create(h.receiver.fname)
	defer file.Close()
	if err == nil {
		log.Println("Writing file sended from client")

		writer := bufio.NewWriter(file)
		for _, segment := range h.receiver.accepted {
			_, err = writer.Write(segment.content)

			if err != nil {
				break
			}

		}

		if err == nil {
			err = writer.Flush()
		}
	}
	return err
}

func (h *RequestHandler) handleSndMsg(recv []byte) error {
	msg := recv[1:]
	if len(recv)-1 == 0 {
		return errors.New("Empty Message from " + h.ra.String())
	}

	h.appendHistorys(string(msg))
	return nil
}

func (h *RequestHandler) appendHistorys(msg string) {
	record := string(msg)
	record = time.Now().Format(time.UnixDate) + ": " + record + " from " + h.ra.String()
	h.hub.history = append(h.hub.history, record)
}

func (h *RequestHandler) handleHisReq() error {
	h.hub.mu.Lock()
	defer h.hub.mu.Unlock()

	logs := strings.Join(h.hub.history, "; ")
	if len(logs) == 0 {
		logs = "No history now."
	}

	log.Println("Sending: [" + logs + "] to " + h.ra.String())
	_, err := h.hub.conn.WriteToUDP([]byte(logs), h.ra)
	return err
}

func (h *Hub) listen() {
	for {
		recv := make([]byte, 512)

		n, ra, err := h.conn.ReadFromUDP(recv)
		if err != nil {
			log.Println("Server " + err.Error())
			continue
		}

		recv = recv[:n]

		var handler *RequestHandler
		if RequestType(recv[0]) == kReqSendSeg {
			var has bool
			packet_id := binary.LittleEndian.Uint64(recv[1:9])
			if handler, has = h.fileHandlers[packet_id]; !has {
				log.Println("[Error] Unexpected segment from: " + ra.String())
				continue
			}
		} else {
			handler = NewRequestHandler(ra, h)
		}

		// handle the request in background.
		go handler.Handle(recv)
	}
}

func NewHub() (*Hub, error) {
	hub := new(Hub)
	hub.fileHandlers = make(map[uint64]*RequestHandler)
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
