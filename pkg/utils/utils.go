package utils

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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

func IsPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false // 无效的IP
	}

	// 判断是否为局域网IP
	privateCIDRs := []string{
		"10.0.0.0/8",     // 10.0.0.0 - 10.255.255.255
		"172.16.0.0/12",  // 172.16.0.0 - 172.31.255.255
		"192.168.0.0/16", // 192.168.0.0 - 192.168.255.255
	}

	for _, cidr := range privateCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}

func GetOutBoundIPv4V2() string {
	// 创建一个自定义的 HTTP 客户端，强制使用 IPv4
	transport := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			// 使用 IPv4 地址进行连接
			return net.DialTimeout("tcp4", addr, 30*time.Second)
		},
	}

	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get("https://ip.me")
	if err != nil {
		return ""
	}
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	re := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	ip := re.FindString(string(bs))
	return ip
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

func MustStructToReader(v interface{}) io.Reader {
	// 序列化为 JSON 字节流
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
		return nil
	}
	// 包装为 Reader
	return bytes.NewReader(data)
}

func DownloadFile(url, dir, filename string) error {
	// 1. 确保目录存在（自动创建）
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 2. 拼接完整文件路径
	filePath := filepath.Join(dir, filename)

	// 3. 创建目标文件
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer outFile.Close()

	// 4. 发起HTTP GET请求
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 5. 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("错误状态码: %d", resp.StatusCode)
	}

	// 6. 写入文件内容
	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}
