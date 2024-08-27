package gira

// sdk账号
type SdkAccount struct {
	UnionId         string
	OpenId          string
	AccessToken     string // access token
	RefreshToken    string // refresh token
	TokenExpireTime int64  // token过期时间
	NickName        string // 昵称
	Gender          int32  // 性别
	SmallPic        string // 小头像地址
	LargePic        string // 大头像地址
}

// sdk支付订单
type SdkPayOrder struct {
	CporderId  string
	OrderId    string
	PayTime    int64
	Amount     int64 // 单位 分
	UserDefine string
	Response   string
}

type PlatformSdk interface {
	// 登录
	Login(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*SdkAccount, error)
	// 检测支付订单是否有效
	PayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*SdkPayOrder, error)
}
