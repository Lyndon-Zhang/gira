package platform

import (
	"context"
	"fmt"
	"strconv"
	"github.com/Lyndon-Zhang/gira/platform/kuaishou"
    "github.com/Lyndon-Zhang/gira/platform/a4399"
	"github.com/Lyndon-Zhang/gira/platform/weixin"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/codes"
	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/platform/douyin"
	"github.com/Lyndon-Zhang/gira/platform/ultra"
)

func NewConfigSdk(config gira.PlatformConfig) *PlatformSdk {
	self := &PlatformSdk{
		sdkDict: make(map[string]sdk_server),
	}
	if config.Test != nil {
		self.testSdk = NewConfigTestSdk(*config.Test)
		self.sdkDict["test"] = NewConfigTestSdk(*config.Test)
	}
	if config.Pwd != nil {
		self.pwdSdk = NewConfigGfSdk(*config.Pwd)
		self.sdkDict["pwd"] = NewConfigGfSdk(*config.Pwd)
	}
	if config.Ultra != nil {
		self.ultraSdk = NewConfigUltraSdk(*config.Ultra)
		self.sdkDict["ultra"] = NewConfigUltraSdk(*config.Ultra)
	}
	if config.Douyin != nil {
		self.douyinSdk = NewConfigDouyinSdk(*config.Douyin)
		self.sdkDict["douyin"] = NewConfigDouyinSdk(*config.Douyin)
	}
	if config.Weixin != nil {
		self.weixinSdk = NewConfigWeixinSdk(*config.Weixin)
		self.sdkDict["weixin"] = NewConfigWeixinSdk(*config.Weixin)
	}
	if config.Kuaishou != nil {
		self.kuaishouSdk = NewConfigKuaishouSdk(*config.Kuaishou)
		self.sdkDict["kuaishou"] = NewConfigKuaishouSdk(*config.Kuaishou)
	}
    if config.A4399 != nil {
		self.a4399Sdk = NewConfigA4399Sdk(*config.A4399)
		self.sdkDict["a4399"] = NewConfigA4399Sdk(*config.A4399)
	}
	return self
}

func NewConfigTestSdk(config gira.TestPlatformConfig) *TestSdk {
	self := &TestSdk{
		config: config,
	}
	return self
}

func NewConfigGfSdk(config gira.PwdPlatformConfig) *PwdSdk {
	self := &PwdSdk{
		config: config,
	}
	return self
}

func NewConfigUltraSdk(config gira.UltraPlatformConfig) *UltraSdk {
	self := &UltraSdk{
		config: config,
	}
	return self
}

func NewConfigDouyinSdk(config gira.DouyinPlatformConfig) *DouyinSdk {
	self := &DouyinSdk{
		config: config,
	}
	return self
}

func NewConfigWeixinSdk(config gira.WeixinPlatformConfig) *WeixinSdk {
	self := &WeixinSdk{
		config: config,
	}
	return self
}
func NewConfigKuaishouSdk(config gira.KuaishouPlatformConfig) *KuaishouSdk {
	self := &KuaishouSdk{
		config: config,
	}
	return self
}
func NewConfigA4399Sdk(config gira.A4399PlatformConfig) *A4399Sdk {
	self := &A4399Sdk{
		config: config,
	}
	return self
}

// 服务端sdk接口
type sdk_server interface {
	Login(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*gira.SdkAccount, error)
	PayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*gira.SdkPayOrder, error)
}

type PlatformSdk struct {
	testSdk     *TestSdk
	pwdSdk      *PwdSdk
	ultraSdk    *UltraSdk
	douyinSdk   *DouyinSdk
	weixinSdk   *WeixinSdk
	kuaishouSdk *KuaishouSdk
    a4399Sdk    *A4399Sdk
	sdkDict     map[string]sdk_server
}

func (self *PlatformSdk) Login(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*gira.SdkAccount, error) {
	if sdk, ok := self.sdkDict[accountPlat]; !ok {
		return nil, errors.ErrSdkComponentNotImplement
	} else {
		return sdk.Login(accountPlat, openId, token, authUrl, appId, appSecret)
	}
}

func (self *PlatformSdk) PayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*gira.SdkPayOrder, error) {
	if sdk, ok := self.sdkDict[accountPlat]; !ok {
		return nil, errors.ErrSdkComponentNotImplement
	} else {
		return sdk.PayOrderCheck(accountPlat, args, paySecret)
	}
}

type TestSdk struct {
	config gira.TestPlatformConfig
}

func (self *TestSdk) Login(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*gira.SdkAccount, error) {
	if token != appSecret {
		return nil, errors.ErrInvalidSdkToken
	}
	result := &gira.SdkAccount{
		OpenId:   openId,
		NickName: openId,
	}
	return result, nil
}

func (self *TestSdk) PayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*gira.SdkPayOrder, error) {
	if v, ok := args["cporder_id"]; !ok {
		return nil, errors.New("invalid pay order, cporder_id is nil")
	} else if cporderId, ok := v.(string); !ok {
		return nil, errors.New("invalid pay order, cporder_id is not string type")
	} else if v, ok := args["amount"]; !ok {
		return nil, errors.New("invalid pay order, amount is nil")
	} else if amount, ok := v.(int64); !ok {
		return nil, errors.New("invalid pay order, cporder_id is not int64 type")
	} else {
		result := &gira.SdkPayOrder{
			CporderId: cporderId,
			OrderId:   cporderId,
			// PayTime: payInfo.PayTime,
			Amount:   amount,
			Response: "SUCCESS",
		}
		return result, nil
	}
}

type PwdSdk struct {
	config gira.PwdPlatformConfig
}

func (self *PwdSdk) Login(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*gira.SdkAccount, error) {
	result := &gira.SdkAccount{
		OpenId:      openId,
		NickName:    openId,
		AccessToken: token, // 登录密码
	}
	return result, nil
}

func (self *PwdSdk) PayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*gira.SdkPayOrder, error) {
	return nil, errors.ErrSdkPayOrderCheckMethodNotImplement
}

type UltraSdk struct {
	config   gira.UltraPlatformConfig
	loginSdk *ultra.USDK
}

func (self *UltraSdk) Login(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*gira.SdkAccount, error) {
	log.Infow("ultra sdk login", "open_id", openId, "token", token, "app_id", appId, "app_secret", appSecret, "auth_url", authUrl)
	if self.loginSdk == nil {
		if sdk, err := ultra.NewUSDK(context.Background(), authUrl, appId, appSecret, "", ultra.AreaGlobal); err != nil {
			return nil, err
		} else {
			self.loginSdk = sdk
		}
	}
	if _, resp, err := self.loginSdk.Login(openId, "", token); err != nil {
		return nil, err
	} else if resp.Code != 0 {
		return nil, codes.New(int32(resp.Code), resp.Msg)
	} else {
		result := &gira.SdkAccount{
			OpenId:   fmt.Sprintf("%v_%v", resp.ChannelId, resp.CUid),
			NickName: resp.CName,
		}
		return result, nil
	}
}

func (self *UltraSdk) PayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*gira.SdkPayOrder, error) {
	if v, ok := args["data"]; !ok {
		return nil, errors.New("invalid pay order args, data is nil")
	} else if data, ok := v.(string); !ok {
		return nil, errors.New("invalid pay order args, data is not string type")
	} else if v, ok := args["sign"]; !ok {
		return nil, errors.New("invalid pay order args, sign is nil")
	} else if sign, ok := v.(string); !ok {
		return nil, errors.New("invalid pay order args, sign is not string type")
	} else if sdk, err := ultra.NewUSDK(context.Background(), "", "", "", paySecret, ultra.AreaGlobal); err != nil {
		return nil, err
	} else if response, _, payInfo, err := sdk.PayCallback(data, sign); err != nil {
		return nil, err
	} else {
		log.Printf("支付回调 gameOrder: %v\t游戏的订单号", payInfo.GameOrder)
		log.Printf("支付回调 selfDefine: %v\t游戏的自定义信息", payInfo.SelfDefine)
		log.Printf("支付回调 amount: %v\t付款金额", payInfo.Amount)
		log.Printf("支付回调 payTime: %v\t支付时间", payInfo.PayTime)
		log.Printf("支付回调 goodsId: %v\t支付的道具ID", payInfo.GoodsId)
		log.Printf("支付回调 channelUid: %v\t渠道用户ID", payInfo.ChannelUid)
		log.Printf("支付回调 channel: %v\t渠道码", payInfo.Channel)
		log.Printf("支付回调 channelId: %v\t渠道ID", payInfo.ChannelId)
		log.Printf("支付回调 orderNo: %v\tUSDK的订单号", payInfo.OrderNo)
		result := &gira.SdkPayOrder{
			CporderId: payInfo.GameOrder,
			OrderId:   payInfo.OrderNo,
			// PayTime: payInfo.PayTime,
			Amount:   int64(payInfo.Amount),
			Response: response,
		}
		return result, nil
	}
}

type DouyinSdk struct {
	config gira.DouyinPlatformConfig
}

func (self *DouyinSdk) Login(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*gira.SdkAccount, error) {
	log.Infow("douyin sdk login", "open_id", openId, "token", token, "app_id", appId, "app_secret", appSecret, "auth_url", authUrl)
	if resp, err := douyin.JsCode2Session(appId, appSecret, token, ""); err != nil {
		return nil, err
	} else if resp.Error != 0 {
		return nil, errors.New(resp.ErrMsg)
	} else {
		result := &gira.SdkAccount{
			OpenId:   resp.OpenId,
			NickName: "",
		}
		return result, nil
	}
}

func (self *DouyinSdk) PayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*gira.SdkPayOrder, error) {
	return nil, errors.ErrTODO
}

type WeixinSdk struct {
	config gira.WeixinPlatformConfig
}

func (self *WeixinSdk) Login(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*gira.SdkAccount, error) {
	log.Infow("weixin sdk login", "open_id", openId, "token", token, "app_id", appId, "app_secret", appSecret, "auth_url", authUrl)
	if resp, err := weixin.JsCode2Session(appId, appSecret, token, ""); err != nil {
		return nil, err
	} else if resp.ErrCode != 0 {
		return nil, errors.New(resp.ErrMsg)
	} else {
		result := &gira.SdkAccount{
			OpenId:   resp.OpenId,
			NickName: "",
		}
		return result, nil
	}
}

func (self *WeixinSdk) PayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*gira.SdkPayOrder, error) {
	return nil, errors.ErrTODO
}

type KuaishouSdk struct {
	config gira.KuaishouPlatformConfig
}

func (self *KuaishouSdk) Login(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*gira.SdkAccount, error) {
	log.Infow("kuaishou sdk login", "open_id", openId, "token", token, "app_id", appId, "app_secret", appSecret, "auth_url", authUrl)
	if resp, err := kuaishou.JsCode2Session(appId, appSecret, token, ""); err != nil {
		return nil, err
	} else if resp.Result != 1 {
		return nil, errors.New(fmt.Sprintf("%d %s", resp.Result, resp.Message))
	} else {
		result := &gira.SdkAccount{
			OpenId:   resp.OpenId,
			NickName: "",
		}
		return result, nil
	}
}

func (self *KuaishouSdk) PayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*gira.SdkPayOrder, error) {
	return nil, errors.ErrTODO
}

type A4399Sdk struct {
	config gira.A4399PlatformConfig
}

func (self *A4399Sdk) Login(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*gira.SdkAccount, error) {
	log.Infow("4399 sdk login", "open_id", openId, "token", token, "app_id", appId, "app_secret", appSecret, "auth_url", authUrl)
	if resp, err := a4399.JsCode2Session(appId, appSecret, token, ""); err != nil {
		return nil, err
	} else if resp.Code != 100 {
		return nil, errors.New(fmt.Sprintf("%d %s", resp.Code, resp.Message))
	} else {
		result := &gira.SdkAccount{
			OpenId:strconv.FormatInt(resp.Result.UID, 10),
			// Username: resp.Result.Username,
			NickName: resp.Result.NickName,
			// AvatarUrl:resp.Result.AvatarUrl,
		}
		return result, nil
	}
}

func (self *A4399Sdk) PayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*gira.SdkPayOrder, error) {
	return nil, errors.ErrTODO
}


