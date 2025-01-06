package udp_chain_connector

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// ProxyServer 代理服务器
type ProxyServer struct {
	mode         Mode
	localAddr    *net.UDPAddr
	proxyAddr    *net.UDPAddr
	targetAddr   *net.UDPAddr
	conn         *net.UDPConn
	clients      sync.Map
	timeout      time.Duration
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

// Client 客户端连接信息
type Client struct {
	targetConn *net.UDPConn
	targetAddr *net.UDPAddr
	lastActive time.Time
	wg         *sync.WaitGroup
}

const (
	// 包头格式:
	// [1字节版本和标志位][1字节IP长度][N字节IP地址][2字节端口][数据...]
	// 版本和标志位:
	// - 最高位(bit7): 0=IPv4, 1=IPv6
	// - 其他位保留为将来使用
	VersionIPv4     byte = 0x00
	VersionIPv6     byte = 0x80
	MinHeaderSize        = 4         // 版本(1) + IP长度(1) + 最小IP长度(0) + 端口(2)
	largeBufferSize      = 32 * 1024 // 32KB large buffer
)

type Mode int

const (
	ClientMode Mode = iota
	ServerMode
)

var lPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, largeBufferSize)
	},
}

func main() {
	mode := flag.String("mode", "first", "代理模式: first(客户端) 或 second(服务器)")
	localAddr := flag.String("local", "", "本地监听地址")
	proxyAddr := flag.String("proxy", "", "代理服务器地址(仅客户端模式需要)")
	timeout := flag.Duration("timeout", 5*time.Minute, "客户端超时时间")
	flag.Parse()

	if *localAddr == "" {
		log.Fatal("请指定本地监听地址(-local)")
	}

	if *mode == "first" && *proxyAddr == "" {
		log.Fatal("客户端模式需要指定代理服务器地址(-proxy)")
	}

	server, err := NewProxyServer(*localAddr, *proxyAddr, "", *timeout)
	if err != nil {
		log.Fatalf("创建代理服务器失败: %v", err)
	}

	log.Printf("UDP代理服务器已启动\n模式: %s\n监听地址: %s\n", *mode, *localAddr)
	if *mode == "first" {
		log.Printf("代理服务器: %s\n", *proxyAddr)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("启动代理服务器失败: %v", err)
	}
}

// NewProxyServer 创建新的代理服务器
func NewProxyServer(localAddrStr, proxyAddrStr, targetAddrStr string, timeout time.Duration) (*ProxyServer, error) {
	localAddr, err := net.ResolveUDPAddr("udp", localAddrStr)
	if err != nil {
		return nil, fmt.Errorf("解析本地地址失败: %v", err)
	}

	var proxyAddr *net.UDPAddr
	if proxyAddrStr != "" {
		proxyAddr, err = net.ResolveUDPAddr("udp", proxyAddrStr)
		if err != nil {
			return nil, fmt.Errorf("解析代理地址失败: %v", err)
		}
	}

	var targetAddr *net.UDPAddr
	if targetAddrStr != "" {
		targetAddr, err = net.ResolveUDPAddr("udp", targetAddrStr)
		if err != nil {
			return nil, fmt.Errorf("解析代理地址失败: %v", err)
		}
	}
	hlog.Info("=>>>>>> target addr", targetAddrStr, "target", targetAddr)
	mode := ServerMode
	if targetAddrStr != "" {
		mode = ClientMode
	}

	return &ProxyServer{
		mode:         mode,
		localAddr:    localAddr,
		proxyAddr:    proxyAddr,
		targetAddr:   targetAddr,
		timeout:      timeout,
		shutdownChan: make(chan struct{}),
	}, nil
}

func (s *ProxyServer) GetStoreConnKey(port int) string {
	return "udp#" + strconv.Itoa(port)
}

// Start 启动代理服务器
func (s *ProxyServer) Start() error {
	var err error
	s.conn, err = net.ListenUDP("udp", s.localAddr)
	if err != nil {
		return fmt.Errorf("创建本地监听失败: %v", err)
	}

	s.wg.Add(1)
	go s.cleanupExpiredClients()

	buffer := make([]byte, 65507) // UDP最大包大小
	for {
		select {
		case <-s.shutdownChan:
			return nil
		default:
			s.conn.SetReadDeadline(time.Now().Add(time.Second))
			n, remoteAddr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("读取数据失败: %v", err)
				continue
			}

			if s.mode == ServerMode && n <= MinHeaderSize {
				log.Printf("收到的数据包太小，无法解析地址")
				continue
			}

			s.handlePacket(remoteAddr, buffer[:n])
		}
	}
}

// Stop 停止代理服务器
func (s *ProxyServer) Close() {
	close(s.shutdownChan)
	if s.conn != nil {
		s.conn.Close()
	}
	s.clients.Range(func(key, value interface{}) bool {
		client := value.(*Client)
		client.targetConn.Close()
		return true
	})
	s.wg.Wait()
}

// packAddress 将地址打包到数据前面
func (s *ProxyServer) packAddress(targetAddr *net.UDPAddr, data []byte) []byte {
	var version byte
	var ipBytes []byte

	if ip4 := targetAddr.IP.To4(); ip4 != nil {
		version = VersionIPv4
		ipBytes = ip4
	} else {
		version = VersionIPv6
		ipBytes = targetAddr.IP.To16()
	}

	headerSize := 2 + len(ipBytes) + 2
	header := make([]byte, headerSize)

	header[0] = version
	header[1] = byte(len(ipBytes))
	copy(header[2:], ipBytes)
	binary.BigEndian.PutUint16(header[headerSize-2:], uint16(targetAddr.Port))

	return append(header, data...)
}

// unpackAddress 从数据中解析出地址
func (s *ProxyServer) unpackAddress(data []byte) (*net.UDPAddr, []byte, error) {
	if len(data) < MinHeaderSize {
		return nil, nil, fmt.Errorf("数据包太小，最小需要 %d 字节", MinHeaderSize)
	}

	version := data[0]
	ipLen := int(data[1])

	fullHeaderSize := 2 + ipLen + 2
	if len(data) < fullHeaderSize {
		return nil, nil, fmt.Errorf("数据包长度不足，需要 %d 字节", fullHeaderSize)
	}

	ipBytes := data[2 : 2+ipLen]
	ip := net.IP(ipBytes)

	if version == VersionIPv4 && ipLen != 4 {
		return nil, nil, fmt.Errorf("IPv4 地址长度错误: %d", ipLen)
	}
	if version == VersionIPv6 && ipLen != 16 {
		return nil, nil, fmt.Errorf("IPv6 地址长度错误: %d", ipLen)
	}

	port := int(binary.BigEndian.Uint16(data[2+ipLen : fullHeaderSize]))

	return &net.UDPAddr{
		IP:   ip,
		Port: port,
	}, data[fullHeaderSize:], nil
}

// handlePacket 处理接收到的数据包
func (s *ProxyServer) handlePacket(clientAddr *net.UDPAddr, data []byte) {
	if s.mode == ClientMode {
		s.handleClientPacket(clientAddr, data)
	} else {
		s.handleServerPacket(clientAddr, data)
	}
}

// handleClientPacket 处理客户端模式的数据包
func (s *ProxyServer) handleClientPacket(clientAddr *net.UDPAddr, data []byte) {
	clientKey := clientAddr.String()
	clientValue, loaded := s.clients.Load(clientKey)
	var client *Client

	if !loaded {
		targetConn, err := net.DialUDP("udp", nil, s.proxyAddr)
		if err != nil {
			log.Printf("连接代理服务器失败: %v", err)
			return
		}

		clientWg := new(sync.WaitGroup)
		client = &Client{
			targetConn: targetConn,
			targetAddr: s.proxyAddr,
			lastActive: time.Now(),
			wg:         clientWg,
		}

		clientWg.Add(1)
		s.wg.Add(1)
		s.clients.Store(clientKey, client)
		go s.handleTargetReturn(clientAddr, client, clientKey)
	} else {
		client = clientValue.(*Client)
	}

	client.lastActive = time.Now()

	dataProxy := s.packAddress(s.targetAddr, data)

	// 转发数据到代理服务器
	_, err := client.targetConn.Write(dataProxy)
	if err != nil {
		log.Printf("发送数据到代理服务器失败: %v", err)
		s.closeClient(clientKey, client)
	}
}

// handleServerPacket 处理服务器模式的数据包
func (s *ProxyServer) handleServerPacket(clientAddr *net.UDPAddr, data []byte) {
	// 解析目标地址
	targetAddr, actualData, err := s.unpackAddress(data)
	if err != nil {
		log.Printf("解析目标地址失败: %v", err)
		return
	}

	log.Printf("handleServerPacket: 目标地址: %v", targetAddr)

	clientKey := clientAddr.String()
	clientValue, loaded := s.clients.Load(clientKey)
	var client *Client

	if !loaded {
		targetConn, err := net.DialUDP("udp", nil, targetAddr)
		if err != nil {
			log.Printf("连接目标服务器失败: %v", err)
			return
		}

		clientWg := new(sync.WaitGroup)
		client = &Client{
			targetConn: targetConn,
			targetAddr: targetAddr,
			lastActive: time.Now(),
			wg:         clientWg,
		}

		clientWg.Add(1)
		s.wg.Add(1)
		s.clients.Store(clientKey, client)
		go s.handleTargetReturn(clientAddr, client, clientKey)
	} else {
		client = clientValue.(*Client)
	}

	client.lastActive = time.Now()

	// 转发数据到目标服务器
	_, err = client.targetConn.Write(actualData)
	if err != nil {
		log.Printf("转发数据到目标服务器失败: %v", err)
		s.closeClient(clientKey, client)
	}
}

// handleTargetReturn 处理目标服务器的返回数据
func (s *ProxyServer) handleTargetReturn(clientAddr *net.UDPAddr, client *Client, clientKey string) {
	defer func() {
		client.wg.Done()
		s.wg.Done()
	}()

	buffer := make([]byte, 65507)
	for {
		select {
		case <-s.shutdownChan:
			return
		default:
			client.targetConn.SetReadDeadline(time.Now().Add(time.Second))
			n, err := client.targetConn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("从目标服务器读取数据失败: %v", err)
				s.closeClient(clientKey, client)
				return
			}

			// 发送数据回客户端
			_, err = s.conn.WriteToUDP(buffer[:n], clientAddr)
			if err != nil {
				log.Printf("发送数据回客户端失败: %v", err)
				s.closeClient(clientKey, client)
				return
			}

			client.lastActive = time.Now()
		}
	}
}

// closeClient 关闭并清理客户端连接
func (s *ProxyServer) closeClient(clientKey string, client *Client) {
	s.clients.Delete(clientKey)
	client.targetConn.Close()
}

// cleanupExpiredClients 清理过期的客户端连接
func (s *ProxyServer) cleanupExpiredClients() {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdownChan:
			return
		case <-ticker.C:
			now := time.Now()
			var expiredKeys []string

			s.clients.Range(func(key, value interface{}) bool {
				client := value.(*Client)
				if now.Sub(client.lastActive) > s.timeout {
					expiredKeys = append(expiredKeys, key.(string))
				}
				return true
			})

			for _, key := range expiredKeys {
				if clientValue, ok := s.clients.Load(key); ok {
					client := clientValue.(*Client)
					log.Printf("清理过期连接: %s", key)
					s.closeClient(key, client)
				}
			}
		}
	}
}
