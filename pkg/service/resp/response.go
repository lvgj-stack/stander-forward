package resp

import (
	"time"

	"github.com/Mr-LvGJ/stander/pkg/model/entity"
)

type AddChainResp struct {
}

type DelChainResp struct {
	ChainId int64
}

type AddRuleResp struct {
	ListenPort string
	RemoteAddr string

	ChainId string
}

type TestRuleResp struct {
	InboundName string `json:"inboundName,omitempty"`
	InboundTo   string `json:"inboundTo,omitempty"`
	InboundPing int    `json:"inboundPing,omitempty"`

	OutboundName string `json:"outboundName,omitempty"`
	OutboundTo   string `json:"outboundTo,omitempty"`
	OutboundPing int    `json:"outboundPing,omitempty"`

	// for agent
	Ping int `json:"ping,omitempty"`
}

type AddNodeResp struct {
	Key string
}

type DelNodeResp struct {
	ID int64
}

type RawResponse[T any] struct {
	RequestId string
	Error     error
	Result    T
}

type RegisterNodeResp struct {
	Chains []*ChainVO
	Rules  []*RuleVO
}

type ChainVO struct {
	ChainType string
	Port      int32
}

type RuleVO struct {
	ListenPort int32
	ChainAddr  string
	RemoteAddr string
	ChainType  string
}

type DelRuleResp struct {
	RuleId int64 `json:"RuleId"`
}

type ListRuleResp struct {
	Rules      []*entity.Rule
	TotalCount int64
}

type ModifyRuleResp struct {
}

type ListChainResp struct {
	Chains     []*entity.Chain
	TotalCount int64
}

type ListNodeResp struct {
	Nodes      []*entity.Node
	TotalCount int64
}

type ReportNetworkTrafficResp struct {
}

type GetUserPlanInfoResp struct {
	Username         string
	ExpirationTime   time.Time
	ResetTrafficTime time.Time
	PlanTraffic      int32
	UsedTraffic      int32
	PlanName         string
	DailyTraffics    []DailyTraffic
}

type DailyTraffic struct {
	Date    time.Time
	Traffic int32
}

type EditChainResp struct {
}
type EditNodeResp struct {
}

type ListUsersResp struct {
	PageSize   int
	PageNumber int
	TotalCount int64
	Users      []*UserTo
}
type UserTo struct {
	*entity.User
	UsedTraffic int64
}

type EmptyResp struct {
}

type ListNodeChainRelationShipsResp struct {
	Chains []*entity.Chain
}

type AddChainGroupResp struct {
}

type ListChainGroupsResp struct {
	ChainGroups []*ChainGroupVO `json:"ChainGroups"`
}

type ChainEntityForChainGroup struct {
	ID       int64 `json:"ID"`
	Backup   bool  `json:"Backup"`
	MaxFails int32 `json:"MaxFails" default:"2"`
	Timeout  int32 `json:"Timeout" default:"10"`
	Weight   int32 `json:"Weight" default:"1"`

	ChainName string `json:"ChainName"`
}

type ChainGroupVO struct {
	ChainGroupID   string
	ChainGroupName string
	Chains         []*ChainEntityForChainGroup
}
