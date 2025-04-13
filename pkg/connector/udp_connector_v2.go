package connector

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type UDPConnectorV2 struct {
	Src        *net.UDPAddr
	RAddr      string
	ln         *net.UDPConn
	listenPort int
	timeout    time.Duration

	connMap sync.Map
}

func (u *UDPConnectorV2) InitConfig(timeout time.Duration, src, raddr string) error {
	// 解析地址
	laddr, err := net.ResolveUDPAddr("udp", src)
	if err != nil {
		hlog.Errorf("解析本地地址失败: %v", err)
		return err
	}

	// 创建本地监听
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		hlog.Errorf("监听失败: %v", err)
		return err
	}
	hlog.Infof("监听中: %s -> 转发到 %s (超时: %v)", src, raddr, timeout)

	ss := strings.Split(src, ":")
	u.listenPort, _ = strconv.Atoi(ss[len(ss)-1])
	u.Src = laddr
	u.RAddr = raddr
	u.ln = conn
	u.listenPort = 0
	u.timeout = timeout
	return nil
}

func (u *UDPConnectorV2) Listen() error {
	buffer := lPool.Get().([]byte)
	defer lPool.Put(buffer)
	for {
		// 读取客户端数据
		n, clientAddr, err := u.ln.ReadFromUDP(buffer)
		if err != nil {
			hlog.Infof("读取错误: %v", err)
			continue
		}

		go func(data []byte, client *net.UDPAddr) {
			// 创建到目标服务器的连接
			remoteConn, err := net.Dial("udp", u.RAddr)
			if err != nil {
				hlog.Errorf("连接目标失败: %v", err)
				return
			}
			defer remoteConn.Close()

			// 设置写超时（UDP发送）
			err = remoteConn.SetWriteDeadline(time.Now().Add(u.timeout))
			if err != nil {
				return
			}
			if _, err := remoteConn.Write(data); err != nil {
				return
			}

			// 设置读超时（UDP接收）
			err = remoteConn.SetReadDeadline(time.Now().Add(u.timeout))
			if err != nil {
				return
			}
			resp := lPool.Get().([]byte)
			defer lPool.Put(resp)
			rn, err := remoteConn.Read(resp)
			if err != nil {
				return
			}

			// 设置回包写超时
			err = u.ln.SetWriteDeadline(time.Now().Add(u.timeout))
			if err != nil {
				return
			}
			if _, err := u.ln.WriteToUDP(resp[:rn], client); err != nil {
				return
			}
		}(buffer[:n], clientAddr)
	}
}

func (u *UDPConnectorV2) Serve() error {
	return nil
}

func (u *UDPConnectorV2) Accept() (conn net.Conn, err error) {
	return
}

func (u *UDPConnectorV2) Close() error {
	return u.ln.Close()
}

func (u *UDPConnectorV2) StoreConn(m *sync.Map) {
	for {
		if u.ln == nil {
			hlog.Warnf("udp listener is nil, port: %d", u.Src.Port)
			time.Sleep(3 * time.Second)
			continue
		} else {
			hlog.Infof("store src conn to map, port: %d", u.Src.Port)
			m.Store(u.GetStoreConnKey(u.Src.Port), u.ln)
			break
		}
	}
}

func (u *UDPConnectorV2) GetStoreConnKey(port int) string {
	return "udpV2#" + strconv.Itoa(port)
}
