package douyinapi

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Lyndon-Zhang/gira/errors"
)

type JsCode2SessionRequest struct {
	AppId         string `json:"appid"`
	Secret        string `json:"secret"`
	Code          string `json:"code"`
	AnonymousCode string `json:"anonymous_code"`
}

type JsCode2SessionResponse struct {
	ErrCode int32  `json:"err_code"`
	ErrMsg  string `json:"err_tips"`
	Data    struct {
		OpenId     string `json:"openid"`
		UnionId    string `json:"unionid"`
		SessionKey string `json:"session_key"`
	} `json:"data"`
}

// https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/log-in/code-2-session
// 小程序登录
func JsCode2Session(appId string, secret string, code string, anonymousCode string) (*JsCode2SessionResponse, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	host := "https://developer.toutiao.com"
	url := fmt.Sprintf("%s/api/apps/v2/jscode2session", host)
	req := &JsCode2SessionRequest{
		AppId:         appId,
		Secret:        secret,
		Code:          code,
		AnonymousCode: anonymousCode,
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
	httpReq.Header.Add("Content-Type", "application/json")
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
	resp := &JsCode2SessionResponse{}
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type GetAccessTokenRequest struct {
	AppId     string `json:"appid"`
	Secret    string `json:"secret"`
	GrantType string `json:"grant_type"`
}

type GetAccessTokenResponse struct {
	ErrCode int32  `json:"err_code"`
	ErrTips string `json:"err_tips"`
	Data    struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	} `json:"data"`
}

const (
	GetAccessTokenGrantType_AuthorizationCode = "client_credential"
)

// https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/interface-request-credential/get-access-token
// getAccessToken
func GetAccessToken(appId string, secret string, grantType string) (*GetAccessTokenResponse, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	host := "https://developer.toutiao.com"
	url := fmt.Sprintf("%s/api/apps/v2/token", host)
	req := &GetAccessTokenRequest{
		AppId:     appId,
		Secret:    secret,
		GrantType: grantType,
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
	httpReq.Header.Add("Content-Type", "application/json")
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
	resp := &GetAccessTokenResponse{}
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CreateQRCodeResponse struct {
	ErrCode     int32  `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	Buffer      []byte
	ContentType string
}

type CreateQRCodeRequest struct {
	AccessToken  string `json:"access_token"`
	AppName      string `json:"appname"`
	Path         string `json:"path"`
	SetIcon      bool   `json:"set_icon"`
	IsCircleCode bool   `json:"is_circle_code"`
}

// https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/qr-code/create-qr-code
// 二维码
func CreateQRCode(accessToken string, appName string, path string, setIcon bool, isCircleCode bool) (*CreateQRCodeResponse, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	host := "https://developer.toutiao.com"
	params := url.Values{}
	url := fmt.Sprintf("%s/api/apps/qrcode?%s", host, params.Encode())
	req := &CreateQRCodeRequest{
		AccessToken:  accessToken,
		AppName:      appName,
		Path:         path,
		SetIcon:      setIcon,
		IsCircleCode: isCircleCode,
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
	httpReq.Header.Add("Content-Type", "application/json")
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
	resp := &CreateQRCodeResponse{
		ContentType: result.Header.Get("Content-Type"),
	}
	log.Println(result.Header)
	log.Println(resp.ContentType)
	if strings.HasPrefix(resp.ContentType, "image/png") {
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

type GetPhoneNumberResponse struct {
	PhoneNumber     string `json:"phoneNumber"`
	PurePhoneNumber string `json:"purePhoneNumber"`
	CountryCode     string `json:"countryCode"`
	AvatarUrl string `json:"avatarUrl"`
	NickName string `json:"nickName"`
}

// https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/guide/open-capabilities/sensitive-data-process/
// 获取手机号码
func GetPhoneNumber(sessionKey string, encryptedData string, iv string) (*GetPhoneNumberResponse, error) {
	src, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}
	_key, err := base64.StdEncoding.DecodeString(sessionKey)
	if err != nil {
		return nil, err
	}
	_iv, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(_key)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, _iv)
	dst := make([]byte, len(src))
	mode.CryptBlocks(dst, src)
	// 去除填充  --- 数据尾端有'/x0e'占位符,去除它
	l := len(dst)
	dst = dst[:l-int(dst[l-1])]
	fmt.Println("1ff", string(dst))
	resp := &GetPhoneNumberResponse{}
	if err := json.Unmarshal(dst, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
