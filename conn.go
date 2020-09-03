package websocket

import (
	"encoding/json"
	"github.com/gobwas/ws"
	"net"
	"net/url"
	"sync"
	"time"
)

// Conn websocket connection
type Conn struct {
	id     string
	conn   net.Conn
	params url.Values
	done   chan bool
	mu     sync.Mutex
}

var pingHeader = ws.Header{
	Fin:    true,
	OpCode: ws.OpPing,
	Masked: false,
	Length: 0,
}

var PingInterval = time.Second * 5

// ID return an connection identifier (may be not unique)
func (c *Conn) ID() string {
	return c.id
}

// Emit emit message to connection.
func (c *Conn) Emit(name string, data interface{}) error {
	msg := Message{
		Name: name,
		Data: data,
	}
	b, _ := json.Marshal(msg)

	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpText,
		Masked: false,
		Length: int64(len(b)),
	}

	return c.Write(h, b)
}

// Write write byte array to connection.
func (c *Conn) Write(h ws.Header, b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.conn.SetWriteDeadline(time.Now().Add(15000 * time.Millisecond))
	err := ws.WriteHeader(c.conn, h)
	if err != nil {
		return err
	}

	_, err = c.conn.Write(b)
	return err
}

// Send send data to connection.
func (c *Conn) Send(data interface{}) error {
	var b []byte
	switch data.(type) {
	case []byte:
		b = data.([]byte)
	default:
		b, _ = json.Marshal(data)
	}

	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpText,
		Masked: false,
		Length: int64(len(b)),
	}

	err := c.Write(h, b)
	return err
}

// Close closing websocket connection.
func (c *Conn) Close() error {
	c.done <- true

	err := c.conn.Close()
	return err
}

// Param gets the value from url params.
// If there are no values associated with the key, Get returns
// the empty string. To access multiple values, use the map
// directly.
func (c *Conn) Param(key string) string {
	return c.params.Get(key)
}

func (c *Conn) startPing() {
	ticker := time.NewTicker(PingInterval)

	go func() {
		for {
			select {
			case <-ticker.C:
				_ = ws.WriteHeader(c.conn, pingHeader)
			case <-c.done:
				ticker.Stop()
				return
			}
		}
	}()
}
