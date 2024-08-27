package weixin

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type JsCode2SessionRequest struct {
	AppId         string `json:"appid"`
	Secret        string `json:"secret"`
	Code          string `json:"code"`
	AnonymousCode string `json:"anonymous_code"`
}

type JsCode2SessionResponse struct {
	ErrCode    int32  `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
	OpenId     string `json:"openid"`
	SessionKey string `json:"session_key"`
}

// https://developers.weixin.qq.com/minigame/dev/api-backend/open-api/login/auth.code2Session.html
// 小程序登录
func JsCode2Session(appId string, secret string, code string, anonymousCode string) (*JsCode2SessionResponse, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	host := "https://api.weixin.qq.com"
	params := url.Values{}
	params.Set("appid", appId)
	params.Set("secret", secret)
	params.Set("js_code", code)
	params.Set("grant_type", "authorization_code")
	// params.Set("anonymous_code", anonymousCode)

	url := fmt.Sprintf("%s/sns/jscode2session?%s", host, params.Encode())
	log.Println(url)
	var httpReq *http.Request
	var result *http.Response
	var err error
	if err != nil {
		return nil, err
	}
	httpReq, err = http.NewRequest("GET", url, nil)
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
