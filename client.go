package apns

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
)

func NewClient(server, certFilename, keyFilename string) (*Client, error) {

	cert, err := tls.LoadX509KeyPair(certFilename, keyFilename)
	if err != nil {
		return nil, err
	}

	certificate := []tls.Certificate{cert}
	conf := &tls.Config{
		Certificates: certificate,
	}

	c := &Client{
		server: server,
		conf:   conf,
	}

	if err != nil {
		return nil, err
	}

	c.queue = make(chan *Notification, 1000)
	c.errors = make(chan *NotificationError, 1000)
	c.buffer = make([]*Notification, IDENTIFIER_UBOUND+1)
	c.sendLoop()
	return c, err
}

type Client struct {
	server string
	conf   *tls.Config
	conn   *tls.Conn

	queue  chan *Notification
	errors chan *NotificationError
	buffer []*Notification
}

func (c *Client) Connect() error {

	err := c.Close()
	if err != nil {
		return fmt.Errorf("close last connection failed: %s", err)
	}

	conn, err := net.Dial("tcp", c.server)
	if err != nil {
		return fmt.Errorf("connect to server error: %s", err)
	}

	c.conn = tls.Client(conn, c.conf)
	err = c.conn.Handshake()
	if err != nil {
		return fmt.Errorf("handshake server error: %s", err)
	}

	c.readErrors()

	return nil

}

func (c *Client) Close() error {

	if c.conn == nil {
		return nil
	}

	conn := c.conn
	c.conn = nil
	return conn.Close()

}

func (c *Client) Send(n *Notification) {

	c.queue <- n

}

func (c *Client) GetError() *NotificationError {
	return <-c.errors
}

func (c *Client) sendLoop() {

	go func() {

		for {

			n := <-c.queue
			var err error

			for i := 0; i < 3; i++ {

				log.Printf("SEND (%d):%d-%s\n", i, n.Identifier, n.DeviceToken)

				err := c.send(n)

				if err == nil {
					break
				}
				c.Connect()
			}

			if err != nil {
				log.Printf("ERROR send error with retry:%s-%s\n", n.DeviceToken, err)
			}
		}
	}()

}

func (c *Client) send(n *Notification) error {

	if c.conn == nil {
		return fmt.Errorf("Apns is not connected")
	}

	bytes, err := n.ToBytes()

	if err != nil {
		return fmt.Errorf("Notification to byte error: %s", err)
	}

	_, err = c.conn.Write(bytes)
	if err != nil {
		return fmt.Errorf("Write socket error: %s", err)
	}

	return nil

}

func (c *Client) readErrors() {

	go func() {

		if c.conn != nil {

			p := make([]byte, 8)
			num, err := c.conn.Read(p)

			if err != nil {
				log.Println("ERROR read:", err)
				c.Connect()
			} else {
				e := NewNotificationError(p[:num], err)
				c.errors <- e
			}
		}

	}()

}
