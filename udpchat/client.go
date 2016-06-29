package udpchat

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

// The client serves as a application that executes commands in
// sequential order.
type Client struct {
	remote   *net.UDPAddr
	conn     *net.UDPConn
	username string

	// stdin
	input *bufio.Reader

	// indicates iff the user types "quit".
	quitListener chan bool

	recv []byte

	fsender *fileSender
}

// Establishes an udp connection to server.
func (c *Client) connect(host string, port int) (err error) {
	c.remote = &net.UDPAddr{IP: net.ParseIP(host), Port: port}
	c.conn, err = net.DialUDP("udp", nil, c.remote)
	return err
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) Send(msg []byte, t RequestType) error {
	if t == kReqSendChatMsg && len(msg) == 0 {
		return errors.New("Sending empty messages is not allowed.")
	}

	// Fixed-length header reserved for providing information of client
	// requests.
	header := make([]byte, 1)
	header[0] = byte(t)

	msg = append(header, msg...)

	_, err := c.conn.Write(msg)
	return err
}

func (c *Client) handleHisResponse() error {
	n, err := c.conn.Read(c.recv)
	if err != nil {
		return err
	}
	if n != 0 {
		histories := strings.Split(string(c.recv), "; ")
		for i := range histories {
			println(histories[i])
		}
	}
	return nil
}

func (c *Client) SendHisReq() {
	err := c.Send(nil, kReqGetHistory)
	if err == nil {
		err = c.handleHisResponse()
	}
	if err != nil {
		log.Println("[Error] Sending history request: " + err.Error())
	}
}

func (c *Client) SendChatMsg(msg string) {
	if len(msg) == 0 {
		println("Input message should not be empty.")
	} else if len(msg) > 250 {
		println("Length of message should not be larger than 250")
	} else {
		err := c.Send([]byte(msg), kReqSendChatMsg)
		if err != nil {
			log.Println("[Error] Sending chat message: " + err.Error())
		}
	}
}

type fileSender struct {
	fname  string
	client *Client

	// segments that's not accepted
	// TODO use offset as value
	unaccepted map[uint32]([]byte)

	packet_id []byte
}

func (c *Client) newFileSender(fname string) *fileSender {
	f := new(fileSender)
	f.unaccepted = make(map[uint32]([]byte))
	f.client = c
	f.fname = fname

	// TODO use better packet_id, like hash64(ra.String() + fname)
	f.packet_id = make([]byte, 8)
	hs := fnv.New64()
	hs.Write([]byte(f.fname))
	binary.LittleEndian.PutUint64(f.packet_id, hs.Sum64())

	log.Println("Generating packet_id: ", hs.Sum64())

	return f
}

func (c *Client) SendFile(file string) {
	if len(file) == 0 {
		println("Input file name should not be empty.")
	} else {
		c.fsender = c.newFileSender(file)

		send_file_packet := make([]byte, len(file)+8)
		copy(send_file_packet, c.fsender.packet_id)
		copy(send_file_packet[8:], []byte(file))
		err := c.Send(send_file_packet, kReqSendFile)

		if err != nil {
			log.Println("[Error] Sending file: " + err.Error())
			return
		}

		_, err = c.conn.Read(c.recv)
		if err != nil {
			log.Println("[Error] Sending file: " + err.Error())
			return
		}

		switch ResponseType(c.recv[0]) {
		case kRespSendFileOK:
			println("Start file transferring")
			c.fsender.sendFileImpl(file)
			c.fsender = nil
		case kRespSendFileFailed:
			log.Println("[Error] Sending file is not permitted")
			return
		}
	}
}

func (fs *fileSender) sendFileImpl(fname string) {
	file, err := os.Open(fname)
	if err != nil {
		log.Println("[Error] " + err.Error())
		return
	}

	reader := bufio.NewReader(file)
	var sid uint32 = 0
	hitsEOF := false
	content := make([]byte, 512)

	for {
		if hitsEOF {
			//TODO resend segment
			break
		} else {
			n, err := reader.Read(content)
			if err == io.EOF {
				hitsEOF = true
			} else if err != nil {
				log.Println("[Error] File reading " + err.Error())
				return
			}
			content = content[:n]
		}

		err = fs.sendSegment(content, sid)
		if err != nil {
			log.Println("[Error] Segment sending " + err.Error())
		}

		sid += 1
	}
}

func (fs *fileSender) sendSegment(content []byte, seg_id uint32) error {
	packet := new(bytes.Buffer)

	packet.WriteByte(byte(kReqSendSeg))

	// PACKET_ID
	packet.Write(fs.packet_id)

	// SEG_ID
	seg_id_buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(seg_id_buf, seg_id)
	packet.Write(seg_id_buf)

	// SEG_CONTENT
	packet.Write(content)
	fs.unaccepted[seg_id] = content

	_, err := fs.client.conn.Write(packet.Bytes())

	log.Println("Writing segment "+strconv.FormatUint(uint64(seg_id), 10)+" in length ", packet.Len()-13)
	return err
}

// Asynchronously checks the user input.
func (c *Client) checkInput() {
	for {

		fmt.Print(">>> ")

		line, _, err := c.input.ReadLine()

		// TODO input error?
		if err != nil {
			log.Fatal("Input Error: " + err.Error())
		}

		msg := strings.TrimSpace(string(line))

		// "quit" and "history" are case-insensitive, which means
		// commands like "QuIT" and "HiSTOry" are legal.
		if strings.EqualFold(msg, "quit") {
			c.quitListener <- true
			break
		} else if strings.EqualFold(msg, "help") {
			PrintHelpInfo()
			continue
		} else if strings.EqualFold(msg, "history") {
			c.SendHisReq()
			continue
		} else if strings.HasPrefix(msg, "send:") {
			msg = strings.TrimLeft(msg, "send:")
			msg = strings.TrimSpace(msg)
			c.SendChatMsg(msg)
			continue
		} else if strings.HasPrefix(msg, "sendfile:") {
			file := strings.TrimLeft(msg, "sendfile:")
			file = strings.TrimSpace(file)
			c.SendFile(file)
			continue
		}

		fmt.Println("Unsupported command, type \"help\" for more information.")
	}
}

// The main loop takes charge for receiving messages from the server,
// and sending messages to the server.
func (c *Client) RunLoop() {
	go c.checkInput()

loop:
	for {
		select {
		// 这么写没有为什么，就是为了装逼
		case <-c.quitListener:
			break loop
		}
	}
}

// Create a new client with connection to server.
// NOTE Close the opened client when no longer used.
func NewClient(username string) (client *Client, err error) {
	client = new(Client)
	err = client.connect(ServiceHost, ServicePort)
	if err == nil {
		client.input = bufio.NewReader(os.Stdin)
		client.quitListener = make(chan bool)
		client.username = username
		client.recv = make([]byte, 4096)
		client.fsender = nil
	}
	return client, err
}
