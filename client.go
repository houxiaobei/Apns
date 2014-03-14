package apns

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

func NewClient(server, certFilename, keyFilename string, timeout time.Duration) (*Client, error) {

	cert, err := tls.LoadX509KeyPair(certFilename, keyFilename)
	if err != nil {
		return nil, err
	}

	certificate := []tls.Certificate{cert}
	conf := &tls.Config{
		Certificates: certificate,
	}

	c := &Client{
		server:  server,
		conf:    conf,
		timeout: timeout,
	}

	err = c.connect()

	if err != nil {
		return nil, err
	}

	c.queue = make(chan *Notification, 1000)
	c.errors = make(chan *NotificationError, 1000)
	c.sendLoop()
	c.readErrors()
	return c, err
}

type Client struct {
	server  string
	conf    *tls.Config
	conn    *tls.Conn
	timeout time.Duration

	queue  chan *Notification
	errors chan *NotificationError
}

func (c *Client) Send(n *Notification) {

	c.queue <- n

}

func (c *Client) GetError() *NotificationError {
	return <-c.errors
}

func (c *Client) connect() error {

	err := c.close()
	if err != nil {
		return fmt.Errorf("close last connection failed: %s", err)
	}

	conn, err := net.Dial("tcp", c.server)
	if err != nil {
		return fmt.Errorf("connect to server error: %d", err)
	}

	c.conn = tls.Client(conn, c.conf)
	err = c.conn.Handshake()
	if err != nil {
		return fmt.Errorf("handshake server error: %s", err)
	}

	return nil

}

func (c *Client) close() error {

	if c.conn == nil {
		return nil
	}

	conn := c.conn
	c.conn = nil
	return conn.Close()

}

func (c *Client) sendLoop() {

	go func() {

		for {
			n := <-c.queue

			err := c.send(n)

			if err != nil {
				c.errors <- NewNotificationError([]byte("123"), err)
			}

		}
	}()

}

func (c *Client) send(n *Notification) error {

	if c.conn == nil {
		return fmt.Errorf("apns is not connected")
	}

	bytes, err := n.ToBytes()

	if err != nil {
		return fmt.Errorf("notification to byte error: %s", err)
	}

	_, err = c.conn.Write(bytes)
	if err != nil {
		return fmt.Errorf("write socket error: %s", err)
	}

	return nil

}

func (c *Client) readErrors() {

	go func() {
		p := make([]byte, 8)
		num, err := c.conn.Read(p)
		e := NewNotificationError(p[:num], err)
		c.connect()
		c.errors <- e
	}()

}

//单条发送测试
// func (c *Client) SendSync(n *Notification) error {

// 	if c.conn == nil {
// 		err := c.Connect()

// 		if err != nil {
// 			return err
// 		}
// 	}

// 	bytes, err := n.ToBytes()

// 	if err != nil {
// 		return fmt.Errorf("notification to byte error: %s", err)
// 	}

// 	_, err = c.conn.Write(bytes)
// 	if err != nil {
// 		return fmt.Errorf("write socket error: %s", err)
// 	}

// 	p := make([]byte, 8)

// 	c.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
// 	num, err := c.conn.Read(p)
// 	neterr, ok := err.(net.Error)
// 	if ok && neterr.Timeout() {
// 		return nil // timeout isn't an error in this case
// 	}

// 	e := NewNotificationError(p[:num], err)
// 	return e

// }
