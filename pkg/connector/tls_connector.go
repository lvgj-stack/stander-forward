package connector

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
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
	"github.com/Mr-LvGJ/stander/pkg/common"
	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/service/req"
)

type TLSConnector struct {
	Src string

	// As Inbound
	Chain string
	RAddr string

	// TLS
	TLSConfig *tls.Config

	listener net.Listener

	curTraffic  atomic.Int64
	listenPort  int
	releaseLock sync.Mutex
	timeout     time.Duration
}

func (t *TLSConnector) InitConfig(src, chain, raddr, serverName string, certBytes, keyBytes, caBytes []byte) error {
	// hack
	certBytes = []byte(common.TmpCert)
	keyBytes = []byte(common.TmpKey)
	caBytes = []byte(common.TmpCA)
	// hack

	hlog.Infof("TLSConnector InitConfig, src: %s, chain: %s, raddr: %s, serverName: %s", src, chain, raddr, serverName)
	srcAddr, err := net.ResolveTCPAddr("tcp", src)
	if err != nil {
		return err
	}
	conn, err := net.ListenTCP("tcp", srcAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return err
	}
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(caBytes)
	if !ok {
		return errors.New("failed to parse root certificate")
	}

	t.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCertPool,

		ServerName: serverName,
		RootCAs:    clientCertPool,
	}

	t.Chain = chain
	t.RAddr = raddr
	t.Src = src
	if !(t.Chain == "" && t.RAddr == "") {
		t.reportCurTraffic()
	}
	t.timeout = 10 * time.Minute
	return nil
}

func (t *TLSConnector) Listen() error {
	var listener net.Listener
	var err error

	if t.Chain == "" && t.RAddr == "" {
		listener, err = tls.Listen("tcp", t.Src, t.TLSConfig)
	} else {
		listener, err = net.Listen("tcp", t.Src)
	}
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
		hlog.Debugf("%s ====> %s =====> %s", listener.Addr().String(), t.Chain, t.RAddr)
		go t.handleConn(conn)
	}
}

func (t *TLSConnector) Close() error {
	return t.listener.Close()
}

func (t *TLSConnector) writeWithHeader(conn net.Conn, destConn net.Conn) error {
	if _, err := fmt.Fprintf(destConn, "%s\n", t.RAddr); err != nil {
		return err
	}
	return nil
}

func (t *TLSConnector) estabFromHeader(conn net.Conn) (net.Conn, io.Reader, error) {
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

func (t *TLSConnector) reportCurTraffic() {
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
			time.Sleep(time.Minute)
		}
	}()
}

func (t *TLSConnector) handleConn(conn net.Conn) error {
	var (
		destConn net.Conn
		err      error
		//br       *bufio.Reader
	)
	//if !(t.Chain == "" && t.RAddr == "") {
	//	br, err = blockCheck(conn)
	//	if err != nil {
	//		hlog.Errorf("block conn, err: %v", err)
	//		return err
	//	}
	//}

	errChan := make(chan error, 10)

	defer func() {
		for {
			lock := t.releaseLock.TryLock()
			if !lock {
				time.Sleep(time.Second)
			} else {
				break
			}
		}
		if destConn != nil && destConn.Close != nil {
			_ = destConn.Close()
		}
		_ = conn.Close()
		t.releaseLock.Unlock()
	}()

	// as inbound
	if t.Chain != "" && t.RAddr != "" {
		destConn, err = tls.Dial("tcp", t.Chain, t.TLSConfig)
		if err != nil {
			hlog.Error("connect to chain failed", "err", err.Error())
			destConn = nil
			return err
		}
		var wg sync.WaitGroup
		wg.Add(2)
		if err := t.writeWithHeader(conn, destConn); err != nil {
			return err
		}
		//destConn.Write(flushBuffered(br))
		go func() {
			defer wg.Done()
			go func() {
				errChan <- copyConn(conn, destConn, func(n int64) {
					t.curTraffic.Add(n)
				})
			}()
			select {
			case err = <-errChan:
				return
			case <-time.After(t.timeout):
				return
			}
		}()
		go func() {
			defer wg.Done()
			go func() {
				errChan <- copyConn(destConn, conn, func(n int64) {
					t.curTraffic.Add(n)
				})
			}()
			select {
			case err = <-errChan:
				return
			case <-time.After(t.timeout):
				return
			}
		}()
		wg.Wait()
		return err
	} else if t.Chain == "" && t.RAddr == "" {
		var _ io.Reader
		destConn, _, err = t.estabFromHeader(conn)
		if err != nil {
			return err
		}
		if destConn == nil {
			return err
		}
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			//_, err = io.Copy(conn, destConn)
			//errChan <- err
			errChan <- copyConn(conn, destConn, nil)
		}()
		go func() {
			defer wg.Done()
			//_, err = io.Copy(destConn, conn)
			//errChan <- err
			errChan <- copyConn(destConn, conn, nil)
		}()
		//_, _ = io.Copy(destConn, reader)
		wg.Wait()
		return err
	} else {
		destConn, err = net.Dial("tcp", t.RAddr)
		if err != nil {
			destConn = nil
			return err
		}
		if destConn == nil {
			return err
		}
		go func() {
			//_, err = io.Copy(conn, destConn)
			errChan <- copyConn(conn, destConn, func(n int64) { t.curTraffic.Add(n) })
			errChan <- err
		}()
		go func() {
			//_, err = io.Copy(destConn, conn)
			errChan <- copyConn(destConn, conn, func(n int64) { t.curTraffic.Add(n) })
			errChan <- err
		}()
		err = <-errChan
		return err
	}
}

func blockCheck(conn net.Conn) (*bufio.Reader, error) {
	br := bufio.NewReaderSize(conn, 4096)
	peekData, err := br.Peek(4)
	if err != nil && err != io.EOF {
		hlog.Errorf("读取数据失败:%v", err)
		return br, nil
	}
	// 判断是否是 HTTP 请求
	if isHTTPRequest(string(peekData)) {
		// 处理 HTTP 请求（这里只是示范，可以在这里进行更多处理）
		return nil, errors.New("http forbidden")
	}
	return br, nil
}

const httpMethods = "GET,POST,PUT"

func isHTTPRequest(data string) bool {
	// 检查HTTP方法
	method := strings.Split(string(data), " ")[0]
	return strings.Contains(httpMethods, method)
}

func flushBuffered(br *bufio.Reader) []byte {
	var buf bytes.Buffer
	if n := br.Buffered(); n > 0 {
		buf.Grow(n)
		buf.ReadFrom(io.LimitReader(br, int64(n)))
	}
	return buf.Bytes()
}

func (t *TLSConnector) StoreConn(m *sync.Map) {
	for {
		if t.listener == nil {
			continue
		} else {
			ss := strings.Split(t.Src, ":")
			hlog.Infof("store src conn to map, port: %s", ss[len(ss)-1])
			m.Store(ss[len(ss)-1], t.listener)
			break
		}
	}
}
