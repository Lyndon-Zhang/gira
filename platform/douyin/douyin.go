package douyin

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
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
	Error           int32  `json:"error"`
	ErrCode         int32  `json:"errcode"`
	ErrMsg          string `json:"errmsg"`
	Message         string `json:"message"`
	OpenId          string `json:"openid"`
	AnonymousOpenId string `json:"anonymous_openid"`
	SessionKey      string `json:"session_key"`
}

// https://developer.open-douyin.com/docs/resource/zh-CN/mini-game/develop/guide/game-engine/rd-to-SCgame/server/log-in/code-2-session
// 小程序登录
func JsCode2Session(appId string, secret string, code string, anonymousCode string) (*JsCode2SessionResponse, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	host := "https://minigame.zijieapi.com"
	params := url.Values{}
	params.Set("appid", appId)
	params.Set("secret", secret)
	params.Set("code", code)
	// params.Set("anonymous_code", anonymousCode)

	url := fmt.Sprintf("%s/mgplatform/api/apps/jscode2session?%s", host, params.Encode())
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
