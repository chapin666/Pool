package pool

import "errors"

var (
	// ErrorClosed constant
	ErrorClosed = errors.New("pool is closed")
)

// Pool interface
type Pool interface {

	// Get method
	Get() (interface{}, error)

	// Put method
	Put(interface{}) error

	// Close method
	Close(interface{}) error

	// Release resource
	Release()

	// Len return pool size
	Len() int
}
