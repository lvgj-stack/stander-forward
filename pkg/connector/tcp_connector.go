package connector

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"

	"github.com/Mr-LvGJ/stander/pkg/client"
	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/service/req"
)

type TCPConnector struct {
	Src *net.TCPAddr

	// As Inbound
	Chain *net.TCPAddr
	RAddr string

	listener *net.TCPListener

	curTraffic atomic.Int64
	listenPort int
}

func (t *TCPConnector) InitConfig(src, chain, raddr string) error {

	hlog.Infof("TCPConnector InitConfig, src: %s, chain: %s, raddr: %s", src, chain, raddr)
	srcAddr, err := net.ResolveTCPAddr("tcp", src)
	if err != nil {
		return err
	}
	conn, err := net.ListenTCP("tcp", srcAddr)
	if err != nil {
		return err
	}
	conn.Close()
	defer conn.Close()
	if chain != "" {
		chainAddr, err := net.ResolveTCPAddr("tcp", chain)
		if err != nil {
			return err
		}
		t.Chain = chainAddr
	}
	t.RAddr = raddr
	t.Src = srcAddr
	t.reportCurTraffic()
	return nil
}

func (t *TCPConnector) Listen() error {
	listener, err := net.ListenTCP("tcp", t.Src)
	if err != nil {
		return err
	}
	t.listener = listener
	ss := strings.Split(t.listener.Addr().String(), ":")
	t.listenPort, _ = strconv.Atoi(ss[len(ss)-1])
	for {
		conn, err := listener.Accept()
		if err != nil {
			hlog.Errorf("accept failed, err: %v", err)
			return err
		}
		hlog.Debugf("%s ====> %s =====> %s", listener.Addr().String(), t.Chain.String(), t.RAddr)
		go t.handleConn(conn)
	}
}

func (t *TCPConnector) Close() error {
	return t.listener.Close()
}

func (t *TCPConnector) estabFromHeader(conn net.Conn) (*net.TCPConn, io.Reader, error) {
	reader := bufio.NewReader(conn)
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, nil, err
	}
	hlog.Infof("firstLine: %s", firstLine)
	if len(firstLine) == 0 {
		return nil, nil, errors.New("read empty first line")
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", firstLine[:len(firstLine)-1])
	if err != nil {
		return nil, nil, err
	}

	destConn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, nil, err
	}
	if destConn == nil {
		return nil, nil, err
	}
	return destConn, reader, nil
}

func (t *TCPConnector) reportCurTraffic() {
	c := config.GetAgentConfig()
	go func() {
		for {
			if t.curTraffic.Load() == 0 {
				time.Sleep(60 * time.Second)
				continue
			}
			hlog.Infof("reportCurTraffic, curTraffic: %d", t.curTraffic.Load())
			_, err := client.DoRequest(c.ControllerAddr, "data", "ReportNetworkTraffic", c.NodeKey, &req.ReportNetworkTrafficReq{
				Port:    int32(t.listenPort),
				Traffic: t.curTraffic.Load(),
			})
			if err == nil {
				t.curTraffic.Store(0)
			} else {
				hlog.Errorf("ReportNetworkTraffic failed, localAddr: %s, err: %v", t.listener.Addr().String(), err)
			}
			time.Sleep(30 * time.Second)
		}
	}()
}

func (t *TCPConnector) handleConn(rawConn net.Conn) error {
	var (
		destConn net.Conn
		err      error
	)
	conn := rawConn.(*net.TCPConn)
	errChan := make(chan error, 10)
	onClose := func() {
		if conn != nil && conn.Close != nil {
			conn.Close()
		}
		if destConn != nil && destConn.Close != nil {
			destConn.Close()
		}
	}

	// as inbound
	if t.Chain != nil && t.RAddr != "" {
		hlog.Debugf("chain: %s, raddr: %s", t.Chain.String(), t.RAddr)
		destConn, err = net.DialTCP("tcp", nil, t.Chain)
		if err != nil {
			hlog.Error("connect to chain failed", "err", err.Error())
			return err
		}
		if _, err := fmt.Fprintf(destConn, t.RAddr+"\n"); err != nil {
			return err
		}
		go func() {
			errChan <- copyConn(conn, destConn, nil)
		}()
		go func() {
			errChan <- copyConn(destConn, conn, func(i int64) { t.curTraffic.Add(i) })
		}()
	} else if t.Chain == nil && t.RAddr == "" {
		var reader io.Reader
		destConn, reader, err = t.estabFromHeader(conn)
		if err != nil {
			hlog.Error(err)
			return err
		}
		go func() {
			_, err := io.Copy(destConn, conn)
			errChan <- err
		}()
		go func() {
			_, err := io.Copy(conn, destConn)
			errChan <- err
		}()
		_, _ = io.Copy(destConn, reader)
	} else {
		destConn, err = net.Dial("tcp", t.RAddr)
		if err != nil {
			return err
		}
		if destConn == nil {
			return err
		}
		hlog.Debugf("direct send bytes, %s =====> %s =====> %s", conn.RemoteAddr(), conn.LocalAddr(), destConn.RemoteAddr())
		go func() {
			errChan <- copyConn(conn, destConn, nil)
		}()
		go func() {
			errChan <- copyConn(destConn, conn, func(i int64) {
				t.curTraffic.Add(i)
			})
		}()
	}

	select {
	case err := <-errChan:
		onClose()
		if err != nil {
			hlog.Errorf("errChan to chain failed, localAddr: %s, err: %v", t.listener.Addr().String(), err)
			return err
		}
	}
	return nil
}

func (t *TCPConnector) StoreConn(m *sync.Map) {
	for {
		if t.listener == nil {
			continue
		} else {
			hlog.Infof("store src conn to map, port: %d", t.Src.Port)
			m.Store(strconv.Itoa(t.Src.Port), t.listener)
			break
		}
	}
}
