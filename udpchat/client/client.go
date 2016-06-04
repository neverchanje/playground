package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	uc "github.com/neverchanje/unplayground/udpchat"
)

type Client struct {
	remote   *net.UDPAddr
	conn     *net.UDPConn
	username string

	// stdin
	input *bufio.Reader

	// indicates iff the user types "quit".
	quitListener chan bool
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

func (c *Client) send(msg []byte, t uc.RequestType) {
	if t == uc.ReqSendChatMsg && len(msg) == 0 {
		log.Println("Sending empty messages is not allowed.")
		return
	}

	// Fixed-length header reserved for providing information of client
	// requests.
	header := make([]byte, 1)
	header[0] = byte(t)

	msg = append(header, msg...)

	n, err := c.conn.Write(msg)
	if err != nil || n != len(msg) {
		log.Println(err)
	}
}

// Asynchronously checks the user input.
func (c *Client) checkInput() {
	for {

		fmt.Print(">>> ")

		line, _, err := c.input.ReadLine()

		// TODO input error?
		if err != nil {
			log.Fatal(err)
		}

		msg := strings.TrimSpace(string(line))

		if strings.Compare(msg, "quit") == 0 {
			c.quitListener <- true
			break
		} else if strings.Compare(msg, "help") == 0 {
			continue
		} else if strings.Compare(msg, "history") == 0 {
			c.send(nil, uc.ReqGetHistory)
			continue
		} else if strings.HasPrefix(msg, "send:") {
			msg = strings.TrimLeft(msg, "send:")
			msg = strings.TrimSpace(msg)
			c.send([]byte(msg), uc.ReqSendChatMsg)
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
		case <-c.quitListener:
			break loop
		}
	}
}

// Create a new client with connection to server.
// NOTE Close the opened client when no longer used.
func NewClient(username string) (client *Client, err error) {
	client = new(Client)
	err = client.connect(uc.ServiceHost, uc.ServicePort)
	if err == nil {
		client.input = bufio.NewReader(os.Stdin)
		client.quitListener = make(chan bool)
		client.username = username
	}
	return client, err
}
