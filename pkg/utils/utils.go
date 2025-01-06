package utils

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func GetOutBoundIPv4() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return net.IPv4(127, 0, 0, 1).String()
	}
	defer func() {
		if err = conn.Close(); err != nil {
			hlog.CtxErrorf(context.Background(), "GetOutboundIPv4 close connection failed.", "error", err)
		}
	}()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func GetOutBoundIPv6() string {
	conn, err := net.Dial("udp", "[2001:4860:4860::8888]:53")
	if err != nil {
		return net.IPv6loopback.String()
	}
	defer func() {
		if err = conn.Close(); err != nil {
			hlog.CtxErrorf(context.Background(), "GetOutboundIPv6 close connection failed.", "error", err)
		}
	}()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func GenIpAndPort(ipAddr string, port int32) string {
	if strings.Contains(ipAddr, ":") {
		return fmt.Sprintf("[%s]:%d", ipAddr, port)
	}
	return fmt.Sprintf("%s:%d", ipAddr, port)
}

func Md5Hash(s string) string {
	hash := md5.New()
	hash.Write([]byte(s))
	return hex.EncodeToString(hash.Sum(nil))
}

func HandleTcpping(destination string) (int, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", destination, time.Second*10)
	if err == nil {
		conn.Write([]byte("ping\n"))
		conn.Close()
		return int(time.Since(start).Microseconds()) / 1000, nil
	} else {
		return 0, err
	}
}
