package req

import (
	"time"
)

type AddChainGroupReq struct {
	Name   string
	Chains []ChainEntityForChainGroup
}

type ChainEntityForChainGroup struct {
	ID       int64 `json:"ID"`
	Backup   bool  `json:"Backup"`
	MaxFails int32 `json:"MaxFails" default:"2"`
	Timeout  int32 `json:"Timeout" default:"10"`
	Weight   int32 `json:"Weight" default:"1"`
}

type DelChainGroupReq struct {
	ChainGroupID string `json:"ChainGroupID,required"`
}

type AddChainReq struct {
	Port       int32  `json:"Port,required"`
	Name       string `json:"Name,required"`
	PreferIpv6 bool   `json:"PreferIpv6"`

	ChainType string `json:"ChainType" vd:"$=='TLS'||$=='TCP'||$==''"`
	NodeId    int64  `json:"NodeId,required"`
}

type DelChainReq struct {
	ID int64 `json:"ID,required"`

	Port int32
}
type AddRuleReq struct {
	ListenPort int32  `json:"ListenPort,required"`
	RemoteAddr string `json:"RemoteAddr,required"`
	ChainAddr  string `json:"ChainAddr"`
	RuleName   string `json:"RuleName,Required"`

	ChainType string `json:"ChainType" vd:"$=='TLS'||$=='TCP'||$==''"`
	NodeId    int64  `json:"NodeId,required"`
	ChainId   int64
}

type TestRuleReq struct {
	ID int64 `json:"ID,required"`

	// for agent
	Destination string
}

type AddNodeReq struct {
	NodeName string  `json:"NodeName,required"`
	NodeType string  `json:"NodeType,required" vd:"$=='inbound'||$=='outbound'"`
	Rate     float32 `json:"Rate"`

	DefaultIPv6 bool `json:"DefaultIPv6"`
}

type DelNodeReq struct {
	ID int64 `json:"ID,required"`
}

type RegisterNodeReq struct {
	Port int32  `json:"Port"`
	IP   string `json:"IP"`

	Ipv4       string `json:"Ipv4"`
	Ipv6       string `json:"Ipv6"`
	PreferIpv6 bool   `json:"PreferIpv6"`
	ManagerIp  string `json:"ManagerIp"`
}

type DelRuleReq struct {
	ID int64 `json:"ID,required"`

	Port int32
}

type ListRuleReq struct {
	PageSize int32 `json:"PageSize,omitempty"`
	PageNo   int32 `json:"PageNo,omitempty"`
	RuleName string
	OrderBy  string
	Asc      bool
}

type ModifyRuleReq struct {
	ID int64 `json:"ID,required"`

	RuleName   string `json:"RuleName"`
	ListenPort int32  `json:"ListenPort"`
	RemoteAddr string `json:"RemoteAddr"`
	ChainId    int64  `json:"ChainId"`
	ChainType  string `json:"ChainType" vd:"$=='TLS'||$=='TCP'||$==''"`
	// for agent
	OldListenPort int32  `json:"OldListenPort"`
	ChainAddr     string `json:"ChainAddr"`
}

type ModifyRulesReq struct {
	RuleIDs []int64 `json:"RuleIDs"`

	RemoteIp string `json:"RemoteIp"`
}

type ListChainReq struct {
	PageSize  int32  `json:"PageSize,omitempty"`
	PageNo    int32  `json:"PageNo,omitempty"`
	ChainName string `json:"ChainName,omitempty"`
	Protocol  string `json:"Protocol,omitempty"`
}

type ListChainGroupReq struct {
}

type ListNodeReq struct {
	PageSize int32  `json:"PageSize,omitempty"`
	PageNo   int32  `json:"PageNo,omitempty"`
	NodeName string `json:"NodeName,omitempty"`
	NodeType string `json:"nodeType,omitempty"`

	Scene ListNodeScene `json:"Scene"`
}

type ListNodeScene string

const (
	DefaultListNodeScene ListNodeScene = ""
	AddChainScene        ListNodeScene = "AddChainScene"
)

type ReportNetworkTrafficReq struct {
	Port    int32 `json:"Port,required"`
	Traffic int64 `json:"Traffic"`
}

type GetUserPlanInfoReq struct {
	UserId int32 `vd:"$!=0"`
}

type EditChainReq struct {
	ID        int64  `json:"ID,required"`
	ChainName string `json:"ChainName"`
}

type EditNodeReq struct {
	ID       int64   `json:"ID,omitempty"`
	NodeName string  `json:"NodeName,omitempty"`
	Rate     float32 `json:"Rate,omitempty"`
}

type ListUsersReq struct {
	PageSize int `default:"10"`
	PageNo   int `default:"1"`
	Username string
	OrderBy  string
	Asc      bool
}
type EditUserReq struct {
	ID             int32
	ExpirationTime *time.Time
}

type EmptyReq struct {
}

type ListNodeChainRelationShipsReq struct {
	NodeId int64
}

type ObserverNetworkTrafficReq struct {
	Events []struct {
		Kind    string `json:"kind"`
		Service string `json:"service"`
		Type    string `json:"type"`
		Stats   struct {
			TotalConns   int64 `json:"totalConns"`
			CurrentConns int64 `json:"currentConns"`
			InputBytes   int64 `json:"inputBytes"`
			OutputBytes  int64 `json:"outputBytes"`
			TotalErrs    int64 `json:"totalErrs"`
		} `json:"stats"`
	} `json:"events"`
}

type ListPlansReq struct {
}

type AssociatePlanReq struct {
	UserId int32 `json:"userId"`
	PlanId int64 `json:"planId"`
}
