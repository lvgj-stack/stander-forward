package typ

type RequestForServiceRequest struct {
	Addr       string                       `json:"addr"`
	Admission  string                       `json:"admission"`
	Admissions []string                     `json:"admissions"`
	Bypass     string                       `json:"bypass"`
	Bypasses   []string                     `json:"bypasses"`
	Climiter   string                       `json:"climiter"`
	Forwarder  *ForwarderForServiceRequest  `json:"forwarder"`
	Handler    HandlerForServiceRequest     `json:"handler"`
	Hosts      string                       `json:"hosts"`
	Interface  string                       `json:"interface"`
	Limiter    string                       `json:"limiter"`
	Listener   ListenerForServiceRequest    `json:"listener"`
	Logger     string                       `json:"logger"`
	Loggers    []string                     `json:"loggers"`
	Metadata   map[string]string            `json:"metadata"`
	Name       string                       `json:"name"`
	Observer   string                       `json:"observer"`
	Recorders  []RecordersForServiceRequest `json:"recorders"`
	Resolver   string                       `json:"resolver"`
	Rlimiter   string                       `json:"rlimiter"`
	Sockopts   *SockoptsForServiceRequest   `json:"sockopts"`
	Status     *StatusForServiceRequest     `json:"status"`
}
type AuthForServiceRequest struct {
	Password string `json:"password"`
	Username string `json:"username"`
}
type FilterForServiceRequest struct {
	Host     string `json:"host"`
	Path     string `json:"path"`
	Protocol string `json:"protocol"`
}
type HeaderForServiceRequest map[string]string
type RequestHeaderForServiceRequest map[string]string
type ResponseHeaderForServiceRequest map[string]string
type RewriteForServiceRequest struct {
	Match       string `json:"Match"`
	Replacement string `json:"Replacement"`
}
type RewriteBodyForServiceRequest struct {
	Match       string `json:"Match"`
	Replacement string `json:"Replacement"`
	Type        string `json:"Type"`
}
type RewriteURLForServiceRequest struct {
	Match       string `json:"Match"`
	Replacement string `json:"Replacement"`
}
type HTTPForServiceRequest struct {
	Auth           AuthForServiceRequest           `json:"auth"`
	Header         HeaderForServiceRequest         `json:"header"`
	Host           string                          `json:"host"`
	RequestHeader  RequestHeaderForServiceRequest  `json:"requestHeader"`
	ResponseHeader ResponseHeaderForServiceRequest `json:"responseHeader"`
	Rewrite        []RewriteForServiceRequest      `json:"rewrite"`
	RewriteBody    []RewriteBodyForServiceRequest  `json:"rewriteBody"`
	RewriteURL     []RewriteURLForServiceRequest   `json:"rewriteURL"`
}
type MatcherForServiceRequest struct {
	Priority int    `json:"priority"`
	Rule     string `json:"rule"`
}
type MetadataForServiceRequest map[string]any

type OptionsForServiceRequest struct {
	Alpn         []string `json:"alpn"`
	CipherSuites []string `json:"cipherSuites"`
	MaxVersion   string   `json:"maxVersion"`
	MinVersion   string   `json:"minVersion"`
}
type TLSForServiceRequest2 struct {
	Options    OptionsForServiceRequest `json:"options"`
	Secure     bool                     `json:"secure"`
	ServerName string                   `json:"serverName"`
}
type TLSForServiceRequest struct {
	CaFile       string                   `json:"caFile"`
	CertFile     string                   `json:"certFile"`
	CommonName   string                   `json:"commonName"`
	KeyFile      string                   `json:"keyFile"`
	Options      OptionsForServiceRequest `json:"options"`
	Organization string                   `json:"organization"`
	Secure       bool                     `json:"secure"`
	ServerName   string                   `json:"serverName"`
	Validity     int                      `json:"validity"`
}
type NodesForServiceRequest struct {
	Addr     string                    `json:"addr"`
	Auth     AuthForServiceRequest     `json:"auth"`
	Bypass   string                    `json:"bypass"`
	Bypasses []string                  `json:"bypasses"`
	Filter   FilterForServiceRequest   `json:"filter"`
	Host     string                    `json:"host"`
	HTTP     HTTPForServiceRequest     `json:"http"`
	Matcher  MatcherForServiceRequest  `json:"matcher"`
	Metadata MetadataForServiceRequest `json:"metadata"`
	Name     string                    `json:"name"`
	Network  string                    `json:"network"`
	Path     string                    `json:"path"`
	Protocol string                    `json:"protocol"`
	TLS      TLSForServiceRequest2     `json:"tls"`
}
type SelectorForServiceRequest struct {
	FailTimeout int    `json:"failTimeout"`
	MaxFails    int    `json:"maxFails"`
	Strategy    string `json:"strategy"`
}
type ForwarderForServiceRequest struct {
	Hop      string                     `json:"hop"`
	Name     string                     `json:"name"`
	Nodes    []NodesForServiceRequest   `json:"nodes"`
	Selector *SelectorForServiceRequest `json:"selector"`
}
type ChainGroupForServiceRequest struct {
	Chains   []string                  `json:"chains"`
	Selector SelectorForServiceRequest `json:"selector"`
}

type HandlerForServiceRequest struct {
	Auth       *AuthForServiceRequest       `json:"auth"`
	Auther     string                       `json:"auther"`
	Authers    []string                     `json:"authers"`
	Chain      string                       `json:"chain"`
	ChainGroup *ChainGroupForServiceRequest `json:"chainGroup"`
	Limiter    string                       `json:"limiter"`
	Metadata   *MetadataForServiceRequest   `json:"metadata"`
	Observer   string                       `json:"observer"`
	Retries    int                          `json:"retries"`
	TLS        *TLSForServiceRequest        `json:"tls"`
	Type       string                       `json:"type"`
}
type ListenerForServiceRequest struct {
	Auth       *AuthForServiceRequest       `json:"auth"`
	Auther     string                       `json:"auther"`
	Authers    []string                     `json:"authers"`
	Chain      string                       `json:"chain"`
	ChainGroup *ChainGroupForServiceRequest `json:"chainGroup"`
	Metadata   MetadataForServiceRequest    `json:"metadata"`
	TLS        *TLSForServiceRequest        `json:"tls"`
	Type       string                       `json:"type"`
}
type RecordersForServiceRequest struct {
	Metadata MetadataForServiceRequest `json:"metadata"`
	Name     string                    `json:"name"`
	Record   string                    `json:"record"`
}
type SockoptsForServiceRequest struct {
	Mark int `json:"mark"`
}
type EventsForServiceRequest struct {
	Msg  string `json:"msg"`
	Time int    `json:"time"`
}
type StatsForServiceRequest struct {
	CurrentConns int `json:"currentConns"`
	InputBytes   int `json:"inputBytes"`
	OutputBytes  int `json:"outputBytes"`
	TotalConns   int `json:"totalConns"`
	TotalErrs    int `json:"totalErrs"`
}
type StatusForServiceRequest struct {
	CreateTime int                       `json:"createTime"`
	Events     []EventsForServiceRequest `json:"events"`
	State      string                    `json:"state"`
	Stats      StatsForServiceRequest    `json:"stats"`
}

type RequestForObserverRequest struct {
	Name   string                   `json:"name"`
	Plugin PluginForObserverRequest `json:"plugin"`
}
type OptionsForObserverRequest struct {
	Alpn         []string `json:"alpn"`
	CipherSuites []string `json:"cipherSuites"`
	MaxVersion   string   `json:"maxVersion"`
	MinVersion   string   `json:"minVersion"`
}
type TLSForObserverRequest struct {
	CaFile       string                    `json:"caFile"`
	CertFile     string                    `json:"certFile"`
	CommonName   string                    `json:"commonName"`
	KeyFile      string                    `json:"keyFile"`
	Options      OptionsForObserverRequest `json:"options"`
	Organization string                    `json:"organization"`
	Secure       bool                      `json:"secure"`
	ServerName   string                    `json:"serverName"`
	Validity     int                       `json:"validity"`
}
type PluginForObserverRequest struct {
	Addr    string                `json:"addr"`
	Timeout int                   `json:"timeout"`
	TLS     TLSForObserverRequest `json:"tls"`
	Token   string                `json:"token"`
	Type    string                `json:"type"`
}

// chain
type RequestForChainRequest struct {
	Hops     []HopsForChainRequest   `json:"hops"`
	Metadata MetadataForChainRequest `json:"metadata"`
	Name     string                  `json:"name"`
}
type FileForChainRequest struct {
	Path string `json:"path"`
}
type HTTPForChainRequest struct {
	Timeout int    `json:"timeout"`
	URL     string `json:"url"`
}
type MetadataForChainRequest map[string]string
type AuthForChainRequest struct {
	Password string `json:"password"`
	Username string `json:"username"`
}
type OptionsForChainRequest struct {
	Alpn         []string `json:"alpn"`
	CipherSuites []string `json:"cipherSuites"`
	MaxVersion   string   `json:"maxVersion"`
	MinVersion   string   `json:"minVersion"`
}
type TLSForChainRequest struct {
	CaFile       string                  `json:"caFile"`
	CertFile     string                  `json:"certFile"`
	CommonName   string                  `json:"commonName"`
	KeyFile      string                  `json:"keyFile"`
	Options      *OptionsForChainRequest `json:"options"`
	Organization string                  `json:"organization"`
	Secure       bool                    `json:"secure"`
	ServerName   string                  `json:"serverName"`
	Validity     int                     `json:"validity"`
}
type ConnectorForChainRequest struct {
	Auth     *AuthForChainRequest    `json:"auth"`
	Metadata MetadataForChainRequest `json:"metadata"`
	TLS      *TLSForChainRequest     `json:"tls"`
	Type     string                  `json:"type"`
}
type DialerForChainRequest struct {
	Auth     *AuthForChainRequest    `json:"auth"`
	Metadata MetadataForChainRequest `json:"metadata"`
	TLS      *TLSForChainRequest     `json:"tls"`
	Type     string                  `json:"type"`
}
type FilterForChainRequest struct {
	Host     string `json:"host"`
	Path     string `json:"path"`
	Protocol string `json:"protocol"`
}
type HeaderForChainRequest map[string]string
type RequestHeaderForChainRequest map[string]string
type ResponseHeaderForChainRequest map[string]string
type RewriteForChainRequest struct {
	Match       string `json:"Match"`
	Replacement string `json:"Replacement"`
}
type RewriteBodyForChainRequest struct {
	Match       string `json:"Match"`
	Replacement string `json:"Replacement"`
	Type        string `json:"Type"`
}
type RewriteURLForChainRequest struct {
	Match       string `json:"Match"`
	Replacement string `json:"Replacement"`
}
type HTTPForChainRequest2 struct {
	Auth           AuthForChainRequest           `json:"auth"`
	Header         HeaderForChainRequest         `json:"header"`
	Host           string                        `json:"host"`
	RequestHeader  RequestHeaderForChainRequest  `json:"requestHeader"`
	ResponseHeader ResponseHeaderForChainRequest `json:"responseHeader"`
	Rewrite        []RewriteForChainRequest      `json:"rewrite"`
	RewriteBody    []RewriteBodyForChainRequest  `json:"rewriteBody"`
	RewriteURL     []RewriteURLForChainRequest   `json:"rewriteURL"`
}
type MatcherForChainRequest struct {
	Priority int    `json:"priority"`
	Rule     string `json:"rule"`
}
type SockoptsForChainRequest struct {
	Mark int `json:"mark"`
}
type NodesForChainRequest struct {
	Addr      string                   `json:"addr"`
	Bypass    string                   `json:"bypass"`
	Bypasses  []string                 `json:"bypasses"`
	Connector ConnectorForChainRequest `json:"connector"`
	Dialer    DialerForChainRequest    `json:"dialer"`
	Filter    *FilterForChainRequest   `json:"filter"`
	Hosts     string                   `json:"hosts"`
	HTTP      *HTTPForChainRequest2    `json:"http"`
	Interface string                   `json:"interface"`
	Matcher   *MatcherForChainRequest  `json:"matcher"`
	Metadata  MetadataForChainRequest  `json:"metadata"`
	Name      string                   `json:"name"`
	Netns     string                   `json:"netns"`
	Network   string                   `json:"network"`
	Resolver  string                   `json:"resolver"`
	Sockopts  *SockoptsForChainRequest `json:"sockopts"`
	TLS       *TLSForChainRequest      `json:"tls"`
}

type PluginForChainRequest struct {
	Addr    string             `json:"addr"`
	Timeout int                `json:"timeout"`
	TLS     TLSForChainRequest `json:"tls"`
	Token   string             `json:"token"`
	Type    string             `json:"type"`
}
type RedisForChainRequest struct {
	Addr     string `json:"addr"`
	Db       int    `json:"db"`
	Key      string `json:"key"`
	Password string `json:"password"`
	Type     string `json:"type"`
	Username string `json:"username"`
}
type SelectorForChainRequest struct {
	FailTimeout int    `json:"failTimeout"`
	MaxFails    int    `json:"maxFails"`
	Strategy    string `json:"strategy"`
}
type HopsForChainRequest struct {
	Bypass    string                   `json:"bypass"`
	Bypasses  []string                 `json:"bypasses"`
	File      *FileForChainRequest     `json:"file"`
	Hosts     string                   `json:"hosts"`
	HTTP      *HTTPForChainRequest2    `json:"http"`
	Interface string                   `json:"interface"`
	Metadata  *MetadataForChainRequest `json:"metadata"`
	Name      string                   `json:"name"`
	Nodes     []NodesForChainRequest   `json:"nodes"`
	Plugin    *PluginForChainRequest   `json:"plugin"`
	Redis     *RedisForChainRequest    `json:"redis"`
	Reload    int                      `json:"reload"`
	Resolver  string                   `json:"resolver"`
	Selector  *SelectorForChainRequest `json:"selector"`
	Sockopts  *SockoptsForChainRequest `json:"sockopts"`
}
