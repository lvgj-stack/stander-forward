package connector

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

var (
	// KeepAliveTime is the keep alive time period for TCP connection.
	KeepAliveTime = 180 * time.Second
	// DialTimeout is the timeout of dial.
	DialTimeout = 5 * time.Second
	// HandshakeTimeout is the timeout of handshake.
	HandshakeTimeout = 5 * time.Second
	// ConnectTimeout is the timeout for connect.
	ConnectTimeout = 5 * time.Second
	// ReadTimeout is the timeout for reading.
	ReadTimeout = 10 * time.Second
	// WriteTimeout is the timeout for writing.
	WriteTimeout = 10 * time.Second
	// PingTimeout is the timeout for pinging.
	PingTimeout = 30 * time.Second
	// PingRetries is the reties of ping.
	PingRetries = 1
	// default udp node TTL in second for udp port forwarding.
	defaultTTL       = 10 * time.Second
	defaultBacklog   = 128
	defaultQueueSize = 128
	tinyBufferSize   = 512
	smallBufferSize  = 2 * 1024  // 2KB small buffer
	mediumBufferSize = 8 * 1024  // 8KB medium buffer
	largeBufferSize  = 32 * 1024 // 32KB large buffer
	Debug            = false
)

var mPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, mediumBufferSize)
	},
}

var lPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, largeBufferSize)
	},
}

type UDPConnector struct {
	Src *net.UDPAddr

	// As Inbound
	Chain      *net.UDPAddr
	RAddr      string
	ln         *net.UDPConn
	listenPort int

	errChan  chan error
	connChan chan net.Conn
	connMap  *udpConnMap
	config   *UDPListenConfig
}

type UDPListenConfig struct {
	TTL       time.Duration // timeout per connection
	Backlog   int           // connection backlog
	QueueSize int           // recv queue size per connection
}

type udpServerConn struct {
	conn       net.PacketConn
	raddr      net.Addr
	rChan      chan []byte
	closed     chan struct{}
	closeMutex sync.Mutex
	nopChan    chan int
	config     *udpServerConnConfig
}

type udpServerConnConfig struct {
	ttl     time.Duration
	qsize   int
	onClose func()
}

type udpConnMap struct {
	size int64
	m    sync.Map
}

func (t *UDPConnector) InitConfig(src, chain, raddr string) error {
	hlog.Infof("UDPConnector InitConfig, src: %s, chain: %s, raddr: %s", src, chain, raddr)
	srcAddr, err := net.ResolveUDPAddr("udp", src)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if chain != "" {
		chainAddr, err := net.ResolveUDPAddr("udp", chain)
		if err != nil {
			return err
		}
		t.Chain = chainAddr
	}
	t.RAddr = raddr
	t.Src = srcAddr
	t.connMap = &udpConnMap{}
	t.connChan = make(chan net.Conn, defaultBacklog)
	t.errChan = make(chan error)
	t.config = &UDPListenConfig{}

	return nil
}

func (t *UDPConnector) Listen() error {
	listener, err := net.ListenUDP("udp", t.Src)
	if err != nil {
		return err
	}
	t.ln = listener
	hlog.Debugf("UDPConnector Listen: %s, is null? : %v", t.Src, t.ln == nil)
	t.listenPort = listener.LocalAddr().(*net.UDPAddr).Port
	go t.listen()
	err = t.Serve()
	if err != nil {
		hlog.Errorf("UDPConnector Listen: %s, %v", t.Src, err)
		return err
	}
	return nil
}

func (l *UDPConnector) Serve() error {
	var tempDelay time.Duration
	for {
		conn, e := l.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				hlog.Infof("server: Accept error: %v; retrying in %v\n", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0

		go l.Handle(conn)
	}
}

func (l *UDPConnector) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-l.connChan:
	case err, ok = <-l.errChan:
		if !ok {
			err = errors.New("accpet on closed listener")
		}
	}
	return
}

func (t *UDPConnector) Handle(conn net.Conn) {
	defer func() {
		if e := recover(); e != nil {
			hlog.Errorf("panic UDPConnector Handle: %s, %v", t.Src, e)
		}
	}()

	var (
		destConn net.Conn
		err      error
	)

	// as inbound
	if t.Chain != nil && t.RAddr != "" {
		hlog.Debugf("chain: %s, raddr: %s", t.Chain.String(), t.RAddr)
		destConn, err = net.DialUDP("udp", nil, t.Chain)
		if err != nil {
			hlog.Error("connect to chain failed", "err", err.Error())
			return
		}
		if _, err := fmt.Fprintf(destConn, t.RAddr+"\n"); err != nil {
			return
		}
		transport(conn, destConn)
	} else if t.Chain == nil && t.RAddr == "" {
		var reader io.Reader
		destConn, reader, err = t.estabFromHeader(conn)
		if err != nil {
			hlog.Error(err)
			return
		}
		transport(conn, destConn)
		_, _ = io.Copy(destConn, reader)
	} else {
		destConn, err = net.Dial("udp", t.RAddr)
		if err != nil {
			return
		}
		if destConn == nil {
			return
		}
		hlog.Debugf("direct send bytes, %s =====> %s =====> %s", conn.RemoteAddr(), conn.LocalAddr(), destConn.RemoteAddr())
		transport(conn, destConn)
	}
	hlog.Debugf("[udp] end %s - %s\n", conn.RemoteAddr(), conn.LocalAddr())
	conn.Close()
	if destConn != nil && destConn.Close != nil {
		destConn.Close()
	}
}

func transport(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 1)
	go func() {
		errc <- copyBuffer(rw1, rw2)
	}()

	go func() {
		errc <- copyBuffer(rw2, rw1)
	}()

	if err := <-errc; err != nil && err != io.EOF {
		return err
	}

	return nil
}

func copyBuffer(dst io.Writer, src io.Reader) error {
	buf := lPool.Get().([]byte)
	defer lPool.Put(buf)

	_, err := io.CopyBuffer(dst, src, buf)
	return err
}

func (l *UDPConnector) listen() {
	for {
		// NOTE: this buffer will be released in the udpServerConn after read.
		b := mPool.Get().([]byte)

		n, raddr, err := l.ln.ReadFrom(b)
		if err != nil {
			hlog.Infof("[udp] peer -> %s : %s\n", l.ln.LocalAddr(), err)
			l.ln.Close()
			l.errChan <- err
			close(l.errChan)
			return
		}

		hlog.Infof("[udp] ============= -> %s : %s\n", l.ln.LocalAddr(), raddr)
		conn, ok := l.connMap.Get(raddr.String())
		if !ok {
			hlog.Infof("[new udp] ============= -> %s : %s\n", l.ln.LocalAddr(), raddr)
			conn = newUDPServerConn(l.ln, raddr, &udpServerConnConfig{
				ttl:   l.config.TTL,
				qsize: l.config.QueueSize,
				onClose: func() {
					l.connMap.Delete(raddr.String())
					hlog.Debugf("[udp] %s ============= closed (%d)\n", raddr, l.connMap.Size())
				},
			})

			select {
			case l.connChan <- conn:
				l.connMap.Set(raddr.String(), conn)
				hlog.Debugf("[udp] %s -> %s (%d)\n", raddr, l.ln.LocalAddr(), l.connMap.Size())
			default:
				conn.Close()
				hlog.Debugf("[udp] %s - %s: connection queue is full (%d)\n", raddr, l.ln.LocalAddr(), cap(l.connChan))
			}
		}

		select {
		case conn.rChan <- b[:n]:
			if Debug {
				hlog.Debugf("[udp] %s >>> %s : length %d\n", raddr, l.ln.LocalAddr(), n)
			}
		default:
			hlog.Infof("[udp] %s -> %s : recv queue is full (%d)\n", raddr, l.ln.LocalAddr(), cap(conn.rChan))
		}
	}

}

func (t *UDPConnector) Close() error {
	return t.ln.Close()
}

func (t *UDPConnector) writeWithHeader(conn net.Conn, destConn net.Conn) error {
	header := t.RAddr
	hlog.Debugf("UDP write with header, %s =====> %s ======> %s =====> %s, headerLen: %d", conn.RemoteAddr(), conn.LocalAddr(), destConn.RemoteAddr(), t.RAddr, len([]byte(header)))
	_, err := destConn.Write([]byte(header))
	if err != nil {
		return err
	}
	return nil
}

func (t *UDPConnector) estabFromHeader(conn net.Conn) (*net.UDPConn, io.Reader, error) {
	reader := bufio.NewReader(conn)
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, nil, err
	}
	hlog.Debugf("firstLine: %s", firstLine)
	if len(firstLine) == 0 {
		return nil, nil, errors.New("read empty first line")
	}

	udpAddr, err := net.ResolveUDPAddr("udp", firstLine[:len(firstLine)-1])
	if err != nil {
		return nil, nil, err
	}

	destConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, nil, err
	}
	if destConn == nil {
		return nil, nil, err
	}
	return destConn, reader, nil
}

func (t *UDPConnector) StoreConn(m *sync.Map) {
	for {
		if t.ln == nil {
			hlog.Warnf("udp listener is nil, port: %d", t.Src.Port)
			time.Sleep(3 * time.Second)
			continue
		} else {
			hlog.Infof("store src conn to map, port: %d", t.Src.Port)
			m.Store(t.GetStoreConnKey(t.Src.Port), t.ln)
			break
		}
	}
}

func (t *UDPConnector) GetStoreConnKey(port int) string {
	return "udp#" + strconv.Itoa(port)
}

func newUDPServerConn(conn net.PacketConn, raddr net.Addr, cfg *udpServerConnConfig) *udpServerConn {
	if conn == nil || raddr == nil {
		return nil
	}

	if cfg == nil {
		cfg = &udpServerConnConfig{}
	}
	qsize := cfg.qsize
	if qsize <= 0 {
		qsize = defaultQueueSize
	}
	c := &udpServerConn{
		conn:    conn,
		raddr:   raddr,
		rChan:   make(chan []byte, qsize),
		closed:  make(chan struct{}),
		nopChan: make(chan int),
		config:  cfg,
	}
	go c.ttlWait()
	return c
}

func (c *udpServerConn) Close() error {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	select {
	case <-c.closed:
		return errors.New("connection is closed")
	default:
		if c.config.onClose != nil {
			c.config.onClose()
		}
		close(c.closed)
	}
	return nil
}

func (c *udpServerConn) ttlWait() {
	ttl := c.config.ttl
	if ttl == 0 {
		ttl = defaultTTL
	}
	timer := time.NewTimer(ttl)
	defer timer.Stop()

	for {
		select {
		case <-c.nopChan:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(ttl)
		case <-timer.C:
			c.Close()
			return
		case <-c.closed:
			return
		}
	}
}

func (c *udpServerConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *udpServerConn) RemoteAddr() net.Addr {
	return c.raddr
}

func (c *udpServerConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *udpServerConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *udpServerConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *udpServerConn) Read(b []byte) (n int, err error) {
	n, _, err = c.ReadFrom(b)
	return
}

func (c *udpServerConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	select {
	case bb := <-c.rChan:
		n = copy(b, bb)
		if cap(bb) == mediumBufferSize {
			mPool.Put(bb[:cap(bb)])
		}
	case <-c.closed:
		err = errors.New("read from closed connection")
		return
	}

	select {
	case c.nopChan <- n:
	default:
	}

	addr = c.raddr

	return
}

func (c *udpServerConn) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.raddr)
}

func (c *udpServerConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	n, err = c.conn.WriteTo(b, addr)

	if n > 0 {
		if Debug {
			hlog.Debugf("[udp] %s <<< %s : length %d\n", addr, c.LocalAddr(), n)
		}

		select {
		case c.nopChan <- n:
		default:
		}
	}

	return
}

func (m *udpConnMap) Get(key interface{}) (conn *udpServerConn, ok bool) {
	v, ok := m.m.Load(key)
	if ok {
		conn, ok = v.(*udpServerConn)
	}
	return
}

func (m *udpConnMap) Set(key interface{}, conn *udpServerConn) {
	m.m.Store(key, conn)
	atomic.AddInt64(&m.size, 1)
}

func (m *udpConnMap) Delete(key interface{}) {
	m.m.Delete(key)
	atomic.AddInt64(&m.size, -1)
}

func (m *udpConnMap) Range(f func(key interface{}, value *udpServerConn) bool) {
	m.m.Range(func(k, v interface{}) bool {
		return f(k, v.(*udpServerConn))
	})
}

func (m *udpConnMap) Size() int64 {
	return atomic.LoadInt64(&m.size)
}

// UDPForwarder 结构体用于存储转发器的配置和状态
type UDPForwarder struct {
	localAddr    *net.UDPAddr
	targetAddr   *net.UDPAddr
	conn         *net.UDPConn
	clients      sync.Map
	timeout      time.Duration
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

// Client 结构体用于存储客户端连接信息
type Client struct {
	targetConn *net.UDPConn
	lastActive time.Time
	wg         *sync.WaitGroup // 每个客户端使用独立的 WaitGroup
}

// NewUDPForwarder 创建一个新的 UDP 转发器
func NewUDPForwarder(localAddrStr, targetAddrStr string, timeout time.Duration) (*UDPForwarder, error) {
	// 解析本地地址
	localAddr, err := net.ResolveUDPAddr("udp", localAddrStr)
	if err != nil {
		return nil, fmt.Errorf("解析本地地址失败: %v", err)
	}

	// 解析目标地址
	targetAddr, err := net.ResolveUDPAddr("udp", targetAddrStr)
	if err != nil {
		return nil, fmt.Errorf("解析目标地址失败: %v", err)
	}

	return &UDPForwarder{
		localAddr:    localAddr,
		targetAddr:   targetAddr,
		timeout:      timeout,
		shutdownChan: make(chan struct{}),
	}, nil
}

func (f *UDPForwarder) GetStoreConnKey(port int) string {
	return "udp#" + strconv.Itoa(port)
}

// Start 启动转发器
func (f *UDPForwarder) Start() error {
	// 创建本地监听连接
	var err error
	f.conn, err = net.ListenUDP("udp", f.localAddr)
	if err != nil {
		return fmt.Errorf("创建本地监听失败: %v", err)
	}

	// 启动清理过期连接的协程
	f.wg.Add(1)
	go f.cleanupExpiredClients()

	// 创建缓冲区
	buffer := make([]byte, 65507) // UDP最大包大小

	// 开始监听和转发
	for {
		select {
		case <-f.shutdownChan:
			return nil
		default:
			// 设置读取超时，以便能够检查关闭信号
			f.conn.SetReadDeadline(time.Now().Add(time.Second))
			n, remoteAddr, err := f.conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				hlog.Errorf("读取数据失败: %v", err)
				continue
			}

			// 处理接收到的数据
			f.handlePacket(remoteAddr, buffer[:n])
		}
	}
}

// Close 停止转发器
func (f *UDPForwarder) Close() error {
	close(f.shutdownChan)

	// 关闭所有客户端连接
	f.clients.Range(func(key, value interface{}) bool {
		client := value.(*Client)
		client.targetConn.Close()
		return true
	})

	// 关闭主监听连接
	f.conn.Close()

	// 等待所有goroutine结束
	f.wg.Wait()
	return nil
}

// handlePacket 处理接收到的数据包
func (f *UDPForwarder) handlePacket(clientAddr *net.UDPAddr, data []byte) {
	// 获取或创建客户端连接
	clientKey := clientAddr.String()
	clientValue, loaded := f.clients.Load(clientKey)

	var client *Client
	if !loaded {
		// 创建到目标的新连接
		targetConn, err := net.DialUDP("udp", nil, f.targetAddr)
		if err != nil {
			hlog.Errorf("创建到目标的连接失败: %v", err)
			return
		}

		// 为新客户端创建WaitGroup
		clientWg := new(sync.WaitGroup)
		client = &Client{
			targetConn: targetConn,
			lastActive: time.Now(),
			wg:         clientWg,
		}

		// 添加goroutine计数
		clientWg.Add(1)
		f.wg.Add(1)

		// 存储客户端信息
		f.clients.Store(clientKey, client)

		// 启动从目标返回到客户端的转发
		go f.handleTargetReturn(clientAddr, client, clientKey)
	} else {
		client = clientValue.(*Client)
	}

	client.lastActive = time.Now()

	// 转发数据到目标
	_, err := client.targetConn.Write(data)
	if err != nil {
		hlog.Errorf("转发数据到目标失败: %v", err)
		f.closeClient(clientKey, client)
	}
}

// handleTargetReturn 处理从目标返回的数据
func (f *UDPForwarder) handleTargetReturn(clientAddr *net.UDPAddr, client *Client, clientKey string) {
	defer func() {
		client.wg.Done() // 减少客户端的WaitGroup计数
		f.wg.Done()      // 减少forwarder的WaitGroup计数
	}()

	buffer := make([]byte, 65507)

	for {
		select {
		case <-f.shutdownChan:
			return
		default:
			client.targetConn.SetReadDeadline(time.Now().Add(time.Second))
			n, err := client.targetConn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				hlog.Errorf("从目标读取数据失败: %v", err)
				f.closeClient(clientKey, client)
				return
			}

			// 将数据发送回客户端
			_, err = f.conn.WriteToUDP(buffer[:n], clientAddr)
			if err != nil {
				hlog.Errorf("发送数据回客户端失败: %v", err)
				f.closeClient(clientKey, client)
				return
			}

			client.lastActive = time.Now()
		}
	}
}

// closeClient 关闭并清理客户端连接
func (f *UDPForwarder) closeClient(clientKey string, client *Client) {
	f.clients.Delete(clientKey)
	client.targetConn.Close()
}

// cleanupExpiredClients 清理过期的客户端连接
func (f *UDPForwarder) cleanupExpiredClients() {
	defer f.wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-f.shutdownChan:
			return
		case <-ticker.C:
			now := time.Now()
			var expiredKeys []string

			// 首先收集过期的连接
			f.clients.Range(func(key, value interface{}) bool {
				client := value.(*Client)
				if now.Sub(client.lastActive) > f.timeout {
					expiredKeys = append(expiredKeys, key.(string))
				}
				return true
			})

			// 然后清理过期的连接
			for _, key := range expiredKeys {
				if clientValue, ok := f.clients.Load(key); ok {
					client := clientValue.(*Client)
					hlog.Infof("清理过期连接: %s", key)
					f.closeClient(key, client)
				}
			}
		}
	}
}
