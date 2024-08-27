package ultra

type BaseResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type BaseAuthInfo struct {
	CUid      string `json:"cUid"`      // 渠道的用户id
	CName     string `json:"cName"`     // 渠道的账号
	ChannelId int    `json:"channelId"` // 渠道ID
	Channel   string `json:"channel"`   // 渠道
	SwLogin   int    `json:"swLogin"`   // 切登录
	Age       int    `json:"age"`       // 年龄（经过实名的玩家才有）

	Ext interface{} `json:"ext,omitempty"` // 补充信息
}

// 登录 验证结果
type AuthInfo struct {
	BaseResp
	BaseAuthInfo

	Timestamp int64 `json:"timestamp"`
}

// 支付通知的业务数据（来自SDK）
type PayResult struct {
	Amount     float64 `json:"amount"`     // 必须	SDK	成交金额，单位元
	GameOrder  string  `json:"gameOrder"`  // 必须	游戏	游戏订单号
	OrderNo    string  `json:"orderNo"`    // 必须	SDK	整合sdk订单号
	Status     int     `json:"status"`     // 必须	SDK	交易状态，0表示成功
	PayTime    string  `json:"payTime"`    // 必须	SDK	交易时间yyyy-MM-dd HHmmss
	GoodsId    string  `json:"goodsId"`    // 必须	SDK	玩家购买的商品ID
	GoodsName  string  `json:"goodsName"`  // 可选	SDK	后台配置的商品名称
	ChannelUid string  `json:"channelUid"` // 必须	SDK	渠道的用户id
	ChannelId  int     `json:"channelId"`  // 可选	SDK 渠道ID
	Channel    string  `json:"channel"`    // 必须	SDK	渠道类型
	SelfDefine string  `json:"selfDefine"` // 可选	游戏	透传参数

	DealAmount     string `json:"dealAmount"`
	Sandbox        string `json:"sandbox"`
	Paytype        string `json:"paytype"`
	Iap_sub        string `json:"iap_sub"`
	Iap_sub_expire string `json:"iap_sub_expire"`

	Yx_is_in_intro_offer_period string `json:"yx_is_in_intro_offer_period"`
	Yx_sub_type                 string `json:"yx_sub_type"`
	Yx_is_trial_period          string `json:"yx_is_trial_period"`
}
