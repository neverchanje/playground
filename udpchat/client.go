package udpchat

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

type Client struct {
	remote   *net.UDPAddr
	conn     *net.UDPConn
	username string

	// stdin
	input *bufio.Reader

	// indicates iff the user types "quit".
	quitListener chan bool

	recv []byte
}

// Establishes an udp connection to server.
func (c *Client) connect(host string, port int) (err error) {
	c.remote = &net.UDPAddr{IP: net.ParseIP(host), Port: port}
	c.conn, err = net.DialUDP("udp", nil, c.remote)
	println(c.remote.String())
	return err
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) Send(msg []byte, t RequestType) error {
	if t == ReqSendChatMsg && len(msg) == 0 {
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

		if strings.Compare(msg, "quit") == 0 {
			c.quitListener <- true
			break
		} else if strings.Compare(msg, "help") == 0 {
			continue
		} else if strings.Compare(msg, "history") == 0 {
			err := c.Send(nil, ReqGetHistory)
			if err == nil {
				err = c.handleHisResponse()
			}

			if err != nil {
				log.Println("Client " + err.Error())
			}
			continue
		} else if strings.HasPrefix(msg, "send:") {
			msg = strings.TrimLeft(msg, "send:")
			msg = strings.TrimSpace(msg)

			if len(msg) == 0 {
				println("Input message should not be emtpy.")
			} else {
				c.Send([]byte(msg), ReqSendChatMsg)
			}
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
	err = client.connect(ServiceHost, ServicePort)
	if err == nil {
		client.input = bufio.NewReader(os.Stdin)
		client.quitListener = make(chan bool)
		client.username = username
		client.recv = make([]byte, 4096)
	}
	return client, err
}
