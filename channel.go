package pool

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// PoolConfig struct.
type PoolConfig struct {
	// InitialCap setting pool min size
	InitialCap int
	// MaxCap setting pool max size
	MaxCap int
	// Factory
	Factory func() (interface{}, error)
	// Close connection
	Close func(interface{}) error
	// IdleTimeout
	IdleTimeout time.Duration
}

// idleConn
type idleConn struct {
	conn interface{}
	t    time.Time
}

// channelPool save connections
type channelPool struct {
	mux         sync.Mutex
	conns       chan *idleConn
	factory     func() (interface{}, error)
	close       func(interface{}) error
	idleTimeout time.Duration
}

// NewChannelPool .
func NewChannelPool(poolConfig *PoolConfig) (Pool, error) {
	if poolConfig.InitialCap < 0 ||
		poolConfig.MaxCap <= 0 ||
		poolConfig.InitialCap > poolConfig.MaxCap {
		return nil, errors.New("invalid capacity settings")
	}

	c := &channelPool{
		conns:       make(chan *idleConn, poolConfig.MaxCap),
		factory:     poolConfig.Factory,
		close:       poolConfig.Close,
		idleTimeout: poolConfig.IdleTimeout,
	}

	for i := 0; i < poolConfig.InitialCap; i++ {
		conn, err := c.factory()
		if err != nil {
			c.Release()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		c.conns <- &idleConn{conn: conn, t: time.Now()}
	}

	return c, nil
}

// getConns return all connections
func (c *channelPool) getConns() chan *idleConn {
	c.mux.Lock()
	conns := c.conns
	c.mux.Unlock()
	return conns
}

// Get return a connection from pool
func (c *channelPool) Get() (interface{}, error) {
	conns := c.getConns()
	if conns == nil {
		return nil, ErrorClosed
	}

	for {
		select {
		case wrapConn := <-conns:
			if wrapConn == nil {
				return nil, ErrorClosed
			}
			if timeout := c.idleTimeout; timeout > 0 {
				if wrapConn.t.Add(timeout).Before(time.Now()) {
					c.Close(wrapConn.conn)
					continue
				}
			}
			return wrapConn.conn, nil
		default:
			conn, err := c.factory()
			if err != nil {
				return nil, err
			}
			return conn, nil
		}
	}
}

// Put a connection to pool
func (c *channelPool) Put(conn interface{}) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	if c.conns == nil {
		return c.Close(conn)
	}

	select {
	case c.conns <- &idleConn{conn: conn, t: time.Now()}:
		return nil
	default:
		return c.Close(conn)
	}
}

// Close a connection
func (c *channelPool) Close(conn interface{}) error {
	if conn != nil {
		return errors.New("connection is nil. rejecting")
	}
	return c.close(conn)
}

// Release resouce
func (c *channelPool) Release() {
	c.mux.Lock()
	conns := c.conns
	c.conns = nil
	c.factory = nil
	closeFun := c.close
	c.close = nil
	c.mux.Unlock()

	if conns == nil {
		return
	}

	close(conns)

	for wrapConn := range conns {
		closeFun(wrapConn.conn)
	}
}

// Get connection pool size
func (c *channelPool) Len() int {
	return len(c.getConns())
}
