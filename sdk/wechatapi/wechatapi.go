package wechatapi

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/log"

	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	terrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111" // 引入sms

	// 微信支付
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/signers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/validators"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

type JsCode2SessionResponse struct {
	ErrCode    int32  `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
	SessionKey string `json:"session_key"`
	UnionId    string `json:"unionid"`
	OpenId     string `json:"openid"`
}

const (
	JsCode2SessionGrantType_AuthorizationCode = "authorization_code"
)

// https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/code2Session.html
// 小程序登录
func JsCode2Session(appId string, secret string, jsCode string, grantType string) (*JsCode2SessionResponse, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	host := "https://api.weixin.qq.com"
	params := url.Values{}
	params.Set("appid", appId)
	params.Set("secret", secret)
	params.Set("js_code", jsCode)
	params.Set("grant_type", grantType)
	url := fmt.Sprintf("%s/sns/jscode2session?%s", host, params.Encode())
	log.Println("fff", url)
	var result *http.Response
	var err error
	result, err = client.Get(url)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	var body []byte
	body, err = io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	resp := &JsCode2SessionResponse{}
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type GetPhoneNumberRequest struct {
	Code string `json:"code"`
}

type GetPhoneNumberResponse struct {
	ErrCode   int32  `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	PhoneInfo struct {
		PhoneNumber     string `json:"phoneNumber"`
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode     string `json:"countryCode"`
	} `json:"phone_info"`
}

// https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-info/phone-number/getPhoneNumber.html
// 获取手机号码
func GetPhoneNumber(accessToken string, code string) (*GetPhoneNumberResponse, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	host := "https://api.weixin.qq.com"
	params := url.Values{}
	params.Set("access_token", accessToken)
	url := fmt.Sprintf("%s/wxa/business/getuserphonenumber?%s", host, params.Encode())
	req := &GetPhoneNumberRequest{
		Code: code,
	}
	var httpReq *http.Request
	var result *http.Response
	var data []byte
	var err error
	data, err = json.Marshal(req)
	if err != nil {
		return nil, err
	}
	log.Println(string(data))
	httpReq, err = http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	client := &http.Client{Transport: tr}
	// Send the HTTP request
	result, err = client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	var body []byte
	body, err = io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	resp := &GetPhoneNumberResponse{}
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type GetAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

const (
	GetAccessTokenGrantType_AuthorizationCode = "client_credential"
)

// https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/mp-access-token/getAccessToken.html
// 小程序登录
func GetAccessToken(appId string, secret string, grantType string) (*GetAccessTokenResponse, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	host := "https://api.weixin.qq.com"
	params := url.Values{}
	params.Set("appid", appId)
	params.Set("secret", secret)
	params.Set("grant_type", grantType)
	url := fmt.Sprintf("%s/cgi-bin/token?%s", host, params.Encode())
	var result *http.Response
	var err error
	result, err = client.Get(url)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	var body []byte
	body, err = io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	resp := &GetAccessTokenResponse{}
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type GetWxaCodeUnLimitResponse struct {
	ErrCode     int32  `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	Buffer      []byte
	ContentType string
}

type GetWxaCodeUnLimitRequest struct {
	Scene      string `json:"scene"`
	Page       string `json:"page"`
	EnvVersion string `json:"env_version"`
	CheckPath  bool   `json:"check_path"`
}

// https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/qrcode-link/qr-code/getUnlimitedQRCode.html
// 小程序登录
func GetWxaCodeUnLimit(accessToken string, scene string, page string, envVersion string) (*GetWxaCodeUnLimitResponse, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	host := "https://api.weixin.qq.com"
	params := url.Values{}
	params.Set("access_token", accessToken)
	url := fmt.Sprintf("%s/wxa/getwxacodeunlimit?%s", host, params.Encode())
	req := &GetWxaCodeUnLimitRequest{
		Scene:      scene,
		Page:       page,
		EnvVersion: envVersion,
		CheckPath:  false,
	}
	var httpReq *http.Request
	var result *http.Response
	var data []byte
	var err error
	data, err = json.Marshal(req)
	if err != nil {
		return nil, err
	}
	log.Println(string(data))
	httpReq, err = http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	client := &http.Client{Transport: tr}
	// Send the HTTP request
	result, err = client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	var body []byte
	body, err = io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	resp := &GetWxaCodeUnLimitResponse{
		ContentType: result.Header.Get("Content-Type"),
	}
	log.Println(result.Header)
	log.Println(resp.ContentType)
	if strings.HasPrefix(resp.ContentType, "image/jpeg") {
		resp.Buffer = body
		return resp, nil
	} else if strings.HasPrefix(resp.ContentType, "application/json") {
		// log.Println(string(body))
		if err = json.Unmarshal(body, resp); err != nil {
			return nil, err
		}
		return resp, nil
	} else {
		return nil, errors.ErrTODO
	}
}

type StsGetCredentialResponse struct {
	ErrCode      int32  `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
	TmpSecretID  string `json:"TmpSecretId"`
	TmpSecretKey string `json:"TmpSecretKey"`
	SessionToken string `json:"Token"`
}

// https://cloud.tencent.com/document/product/436/14048
// 获取临时密钥
// region ap-guangzhou
func StsGetCredential(appId string, secretId string, secretKey string, bucket string, region string) (*StsGetCredentialResponse, error) {
	response := &StsGetCredentialResponse{}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	c := sts.NewClient(
		secretId,  // 用户的 SecretId，建议使用子账号密钥，授权遵循最小权限指引，降低使用风险。子账号密钥获取可参考https://cloud.tencent.com/document/product/598/37140
		secretKey, // 用户的 SecretKey，建议使用子账号密钥，授权遵循最小权限指引，降低使用风险。子账号密钥获取可参考https://cloud.tencent.com/document/product/598/37140
		client,
		// sts.Host("sts.internal.tencentcloudapi.com"), // 设置域名, 默认域名sts.tencentcloudapi.com
		// sts.Scheme("http"),      // 设置协议, 默认为https，公有云sts获取临时密钥不允许走http，特殊场景才需要设置http
	)
	// 策略概述 https://cloud.tencent.com/document/product/436/18023
	opt := &sts.CredentialOptions{
		DurationSeconds: int64(1 * time.Hour.Seconds()),
		Region:          region,
		Policy: &sts.CredentialPolicy{
			Statement: []sts.CredentialPolicyStatement{
				{
					// 密钥的权限列表。简单上传和分片需要以下的权限，其他权限列表请看 https://cloud.tencent.com/document/product/436/31923
					Action: []string{
						// 简单上传
						"name/cos:PostObject",
						"name/cos:PutObject",
						// 分片上传
						"name/cos:InitiateMultipartUpload",
						"name/cos:ListMultipartUploads",
						"name/cos:ListParts",
						"name/cos:UploadPart",
						"name/cos:CompleteMultipartUpload",
					},
					Effect: "allow",
					Resource: []string{
						"*",
						// 这里改成允许的路径前缀，可以根据自己网站的用户登录态判断允许上传的具体路径，例子： a.jpg 或者 a/* 或者 * (使用通配符*存在重大安全风险, 请谨慎评估使用)
						// 存储桶的命名格式为 BucketName-APPID，此处填写的 bucket 必须为此格式
						//	"qcs::cos:ap-guangzhou:uid/" + appId + ":" + bucket + "/exampleobject",
					},
					// 开始构建生效条件 condition
					// 关于 condition 的详细设置规则和COS支持的condition类型可以参考https://cloud.tencent.com/document/product/436/71306
					// Condition: map[string]map[string]interface{}{
					// 	"ip_equal": map[string]interface{}{
					// 		"qcs:ip": []string{
					// 			"10.217.182.3/24",
					// 			"111.21.33.72/24",
					// 		},
					// 	},
					// },
				},
			},
		},
	}
	res, err := c.GetCredential(opt)
	if err != nil {
		return nil, err
	}
	response.TmpSecretID = res.Credentials.TmpSecretID
	response.SessionToken = res.Credentials.SessionToken
	response.TmpSecretKey = res.Credentials.TmpSecretKey
	return response, nil
}

type SmsSendResponse struct {
	ErrCode int32  `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func SmsSend(appId string, secretId string, secretKey string, region string, signName string, templateId string, templateParamSet []string, phoneNumberSet []string) (*SmsSendResponse, error) {
	/* 必要步骤：
	 * 实例化一个认证对象，入参需要传入腾讯云账户密钥对secretId，secretKey。
	 * 这里采用的是从环境变量读取的方式，需要在环境变量中先设置这两个值。
	 * 你也可以直接在代码中写死密钥对，但是小心不要将代码复制、上传或者分享给他人，
	 * 以免泄露密钥对危及你的财产安全。
	 * SecretId、SecretKey 查询: https://console.cloud.tencent.com/cam/capi */
	credential := common.NewCredential(
		secretId,
		secretKey,
	)
	/* 非必要步骤:
	 * 实例化一个客户端配置对象，可以指定超时时间等配置 */
	cpf := profile.NewClientProfile()

	/* SDK默认使用POST方法。
	 * 如果你一定要使用GET方法，可以在这里设置。GET方法无法处理一些较大的请求 */
	cpf.HttpProfile.ReqMethod = "POST"

	/* SDK有默认的超时时间，非必要请不要进行调整
	 * 如有需要请在代码中查阅以获取最新的默认值 */
	// cpf.HttpProfile.ReqTimeout = 5

	/* 指定接入地域域名，默认就近地域接入域名为 sms.tencentcloudapi.com ，也支持指定地域域名访问，例如广州地域的域名为 sms.ap-guangzhou.tencentcloudapi.com */
	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"

	/* SDK默认用TC3-HMAC-SHA256进行签名，非必要请不要修改这个字段 */
	cpf.SignMethod = "HmacSHA1"

	/* 实例化要请求产品(以sms为例)的client对象
	 * 第二个参数是地域信息，可以直接填写字符串ap-guangzhou，支持的地域列表参考 https://cloud.tencent.com/document/api/382/52071#.E5.9C.B0.E5.9F.9F.E5.88.97.E8.A1.A8 */
	client, _ := sms.NewClient(credential, region, cpf)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.WithHttpTransport(tr)

	/* 实例化一个请求对象，根据调用的接口和实际情况，可以进一步设置请求参数
	 * 你可以直接查询SDK源码确定接口有哪些属性可以设置
	 * 属性可能是基本类型，也可能引用了另一个数据结构
	 * 推荐使用IDE进行开发，可以方便的跳转查阅各个接口和数据结构的文档说明 */
	request := sms.NewSendSmsRequest()

	/* 基本类型的设置:
	 * SDK采用的是指针风格指定参数，即使对于基本类型你也需要用指针来对参数赋值。
	 * SDK提供对基本类型的指针引用封装函数
	 * 帮助链接：
	 * 短信控制台: https://console.cloud.tencent.com/smsv2
	 * 腾讯云短信小助手: https://cloud.tencent.com/document/product/382/3773#.E6.8A.80.E6.9C.AF.E4.BA.A4.E6.B5.81 */

	/* 短信应用ID: 短信SdkAppId在 [短信控制台] 添加应用后生成的实际SdkAppId，示例如1400006666 */
	// 应用 ID 可前往 [短信控制台](https://console.cloud.tencent.com/smsv2/app-manage) 查看
	request.SmsSdkAppId = common.StringPtr(appId)

	/* 短信签名内容: 使用 UTF-8 编码，必须填写已审核通过的签名 */
	// 签名信息可前往 [国内短信](https://console.cloud.tencent.com/smsv2/csms-sign) 或 [国际/港澳台短信](https://console.cloud.tencent.com/smsv2/isms-sign) 的签名管理查看
	request.SignName = common.StringPtr(signName)

	/* 模板 ID: 必须填写已审核通过的模板 ID */
	// 模板 ID 可前往 [国内短信](https://console.cloud.tencent.com/smsv2/csms-template) 或 [国际/港澳台短信](https://console.cloud.tencent.com/smsv2/isms-template) 的正文模板管理查看
	request.TemplateId = common.StringPtr(templateId)

	/* 模板参数: 模板参数的个数需要与 TemplateId 对应模板的变量个数保持一致，若无模板参数，则设置为空*/
	request.TemplateParamSet = common.StringPtrs(templateParamSet)

	/* 下发手机号码，采用 E.164 标准，+[国家或地区码][手机号]
	 * 示例如：+8613711112222， 其中前面有一个+号 ，86为国家码，13711112222为手机号，最多不要超过200个手机号*/
	request.PhoneNumberSet = common.StringPtrs(phoneNumberSet)

	/* 用户的 session 内容（无需要可忽略）: 可以携带用户侧 ID 等上下文信息，server 会原样返回 */
	// request.SessionContext = common.StringPtr("")

	/* 短信码号扩展号（无需要可忽略）: 默认未开通，如需开通请联系 [腾讯云短信小助手] */
	// request.ExtendCode = common.StringPtr("")

	/* 国内短信无需填写该项；国际/港澳台短信已申请独立 SenderId 需要填写该字段，默认使用公共 SenderId，无需填写该字段。注：月度使用量达到指定量级可申请独立 SenderId 使用，详情请联系 [腾讯云短信小助手](https://cloud.tencent.com/document/product/382/3773#.E6.8A.80.E6.9C.AF.E4.BA.A4.E6.B5.81)。 */
	// request.SenderId = common.StringPtr("")

	// 通过client对象调用想要访问的接口，需要传入请求对象
	resp, err := client.SendSms(request)
	// 处理异常
	if _, ok := err.(*terrors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return nil, err
	}
	// 非SDK异常，直接失败。实际代码中可以加入其他的处理。
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(resp.Response)
	log.Println(templateId)
	log.Println(signName)
	log.Println(appId)
	log.Println(templateParamSet)
	// 打印返回的json字符串
	fmt.Printf("%s", b)
	response := &SmsSendResponse{}

	/* 当出现以下错误码时，快速解决方案参考
	 * [FailedOperation.SignatureIncorrectOrUnapproved](https://cloud.tencent.com/document/product/382/9558#.E7.9F.AD.E4.BF.A1.E5.8F.91.E9.80.81.E6.8F.90.E7.A4.BA.EF.BC.9Afailedoperation.signatureincorrectorunapproved-.E5.A6.82.E4.BD.95.E5.A4.84.E7.90.86.EF.BC.9F)
	 * [FailedOperation.TemplateIncorrectOrUnapproved](https://cloud.tencent.com/document/product/382/9558#.E7.9F.AD.E4.BF.A1.E5.8F.91.E9.80.81.E6.8F.90.E7.A4.BA.EF.BC.9Afailedoperation.templateincorrectorunapproved-.E5.A6.82.E4.BD.95.E5.A4.84.E7.90.86.EF.BC.9F)
	 * [UnauthorizedOperation.SmsSdkAppIdVerifyFail](https://cloud.tencent.com/document/product/382/9558#.E7.9F.AD.E4.BF.A1.E5.8F.91.E9.80.81.E6.8F.90.E7.A4.BA.EF.BC.9Aunauthorizedoperation.smssdkappidverifyfail-.E5.A6.82.E4.BD.95.E5.A4.84.E7.90.86.EF.BC.9F)
	 * [UnsupportedOperation.ContainDomesticAndInternationalPhoneNumber](https://cloud.tencent.com/document/product/382/9558#.E7.9F.AD.E4.BF.A1.E5.8F.91.E9.80.81.E6.8F.90.E7.A4.BA.EF.BC.9Aunsupportedoperation.containdomesticandinternationalphonenumber-.E5.A6.82.E4.BD.95.E5.A4.84.E7.90.86.EF.BC.9F)
	 * 更多错误，可咨询[腾讯云助手](https://tccc.qcloud.com/web/im/index.html#/chat?webAppId=8fa15978f85cb41f7e2ea36920cb3ae1&title=Sms)
	 */
	return response, nil
}

type PayTransactionsJSAPIResponse struct {
	ErrCode   int32
	ErrMsg    string
	PrepayId  *string
	Appid     *string
	TimeStamp *string
	NonceStr  *string
	Package   *string
	SignType  *string
	PaySign   *string
}

func WithWechatPayAutoAuthCipher(
	mchID string, certificateSerialNo string, privateKey *rsa.PrivateKey, mchAPIv3Key string,
) core.ClientOption {
	mgr := downloader.MgrInstance()

	if !mgr.HasDownloader(context.Background(), mchID) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		c := &http.Client{Transport: tr}
		settings := core.DialSettings{
			HTTPClient: c,
			Signer: &signers.SHA256WithRSASigner{
				MchID:               mchID,
				PrivateKey:          privateKey,
				CertificateSerialNo: certificateSerialNo,
			},
			Validator: &validators.NullValidator{},
		}
		client, err := core.NewClientWithDialSettings(context.Background(), &settings)
		if err != nil {
			return core.ErrorOption{Error: fmt.Errorf("create downloader failed, create client err:%v", err)}
		}
		err = mgr.RegisterDownloaderWithClient(context.Background(), client, mchID, mchAPIv3Key)
		if err != nil {
			return core.ErrorOption{Error: err}
		}
	}
	return option.WithWechatPayAutoAuthCipherUsingDownloaderMgr(mchID, certificateSerialNo, privateKey, mgr)
}

// https://github.com/wechatpay-apiv3/wechatpay-go
// https://pay.weixin.qq.com/wiki/doc/apiv3/apis/chapter3_5_1.shtml
// 小程序jsapi下单
func PayTransactionsJSAPI(ctx context.Context, appId string, mchId string, mchAPIv3Key string, certificateSerialNumber string, apiclientKeyPath string, openId string, description string, cpOrderId string, notifyUrl string, amount int64) (*PayTransactionsJSAPIResponse, error) {
	// 使用 utils 提供的函数从本地文件中加载商户私钥，商户私钥会用来生成请求的签名
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath(apiclientKeyPath)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c := &http.Client{Transport: tr}
	// 使用商户私钥等初始化 client，并使它具有自动定时获取微信支付平台证书的能力
	opts := []core.ClientOption{
		WithWechatPayAutoAuthCipher(mchId, certificateSerialNumber, mchPrivateKey, mchAPIv3Key),
		option.WithHTTPClient(c),
	}
	for _, v := range opts {
		log.Println(v)
	}
	client, err := core.NewClient(ctx, opts...)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	svc := jsapi.JsapiApiService{Client: client}
	// 得到prepay_id，以及调起支付所需的参数和签名
	result, _, err := svc.PrepayWithRequestPayment(ctx,
		jsapi.PrepayRequest{
			Appid:       core.String(appId),
			Mchid:       core.String(mchId),
			Description: core.String(description),
			OutTradeNo:  core.String(cpOrderId),
			NotifyUrl:   core.String(notifyUrl),
			Amount: &jsapi.Amount{
				Total: core.Int64(amount),
			},
			Payer: &jsapi.Payer{
				Openid: core.String(openId),
			},
		},
	)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	resp := &PayTransactionsJSAPIResponse{
		PrepayId:  result.PrepayId,
		Appid:     result.Appid,
		TimeStamp: result.TimeStamp,
		NonceStr:  result.NonceStr,
		Package:   result.Package,
		SignType:  result.SignType,
		PaySign:   result.PaySign,
	}
	return resp, nil
}

type PayTransactionsJSAPINotifyResponse struct {
	ErrCode         int32
	ErrMsg          string
	Amount          *payments.TransactionAmount
	Appid           *string
	Attach          *string
	BankType        *string
	Mchid           *string
	OutTradeNo      *string
	Payer           *payments.TransactionPayer
	PromotionDetail []payments.PromotionDetail
	SuccessTime     *string
	TradeState      *string
	TradeStateDesc  *string
	TradeType       *string
	TransactionId   *string
}

func PayTransactionsJSAPINotify(ctx context.Context, appId string, mchId string, mchAPIv3Key string, certificateSerialNumber string, apiclientKeyPath string, request *http.Request) (*PayTransactionsJSAPINotifyResponse, error) {
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath(apiclientKeyPath)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c := &http.Client{Transport: tr}
	settings := core.DialSettings{
		HTTPClient: c,
		Signer: &signers.SHA256WithRSASigner{
			MchID:               mchId,
			PrivateKey:          mchPrivateKey,
			CertificateSerialNo: certificateSerialNumber,
		},
		Validator: &validators.NullValidator{},
	}
	client, err := core.NewClientWithDialSettings(ctx, &settings)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	err = downloader.MgrInstance().RegisterDownloaderWithClient(ctx, client, mchId, mchAPIv3Key)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	// 2. 获取商户号对应的微信支付平台证书访问器
	certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(mchId)
	// 3. 使用证书访问器初始化 `notify.Handler`
	handler := notify.NewNotifyHandler(mchAPIv3Key, verifiers.NewSHA256WithRSAVerifier(certificateVisitor))

	transaction := new(payments.Transaction)
	notifyReq, err := handler.ParseNotifyRequest(context.Background(), request, transaction)
	// 如果验签未通过，或者解密失败
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	// 处理通知内容
	log.Println(notifyReq.Summary)
	resp := &PayTransactionsJSAPINotifyResponse{
		Amount:          transaction.Amount,
		Appid:           transaction.Appid,
		Attach:          transaction.Attach,
		BankType:        transaction.BankType,
		Mchid:           transaction.Mchid,
		OutTradeNo:      transaction.OutTradeNo,
		Payer:           transaction.Payer,
		PromotionDetail: transaction.PromotionDetail,
		SuccessTime:     transaction.SuccessTime,
		TradeState:      transaction.TradeState,
		TradeStateDesc:  transaction.TradeStateDesc,
		TradeType:       transaction.TradeType,
		TransactionId:   transaction.TransactionId,
	}
	return resp, nil
}
