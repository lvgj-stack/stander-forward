package connector

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type Connector interface {
	Listen() error
	handleConn(net.Conn) error
}

type Close interface {
	Close() error
}

const (
	connTimeout = 10 * time.Second // 连接超时
	bufferSize  = 32 * 1024        // 32KB 缓冲区
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, bufferSize)
	},
}

func copyConn(conn, destConn net.Conn, mid func(int64)) error {
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)
	for {
		n, err := conn.Read(buf)
		if err == io.EOF {
			return err
		}
		if err != nil {
			hlog.Errorf("Error reading from connection: %v", err)
			return err
		}
		if mid != nil {
			mid(int64(n))
		}
		_, err = destConn.Write(buf[:n])
		if err != nil {
			return err
		}
	}
}
