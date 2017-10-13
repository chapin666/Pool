package main

import (
	"fmt"
	"net"
	"time"

	"github.com/chapin/pool"
)

func main() {

	factory := func() (interface{}, error) {
		return net.Dial("tcp", "127.0.0.1:80")
	}

	close := func(v interface{}) error {
		return v.(net.Conn).Close()
	}

	poolConfig := &pool.PoolConfig{
		InitialCap:  5,
		MaxCap:      30,
		Factory:     factory,
		Close:       close,
		IdleTimeout: 15 * time.Second,
	}

	p, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		fmt.Println("err=", err)
	}
	v, err := p.Get()

	p.Put(v)

	current := p.Len()
	fmt.Println("len=", current)
}
