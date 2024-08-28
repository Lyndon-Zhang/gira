package a4399

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	//"net/url"
)

type JsCode2SessionRequest struct {
	AppId         string `json:"appid"`
	Secret        string `json:"secret"`
	Code          string `json:"code"`
	AnonymousCode string `json:"anonymous_code"`
}

type User struct {
	UID       int64  `json:"uid"`
	Username  string `json:"username"`
	NickName  string `json:"nickName"`
	AvatarUrl string `json:"avatarUrl"`
}

type JsCode2SessionResponse struct {
	Code    int  `json:"code"`
	Result  User `json:"result"`
	Message string `json:"message"`
}

// https://mgc-api.5054399.net/service/api/v1/check/${platform_id}/user
// 小程序登录
func JsCode2Session(appId string, secret string, code string, anonymousCode string) (*JsCode2SessionResponse, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	host := "https://mgc-api.5054399.net"
	//params := url.Values{}

	url := fmt.Sprintf("%s/service/api/v1/check/1025/user", host)
	log.Println(url)

	var httpReq *http.Request
	var result *http.Response
	var err error
	if err != nil {
		return nil, err
	}
	httpReq, err = http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	// httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Set("P-Token", code)
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
