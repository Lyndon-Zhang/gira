package ultra

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type AREA int

const (
	AreaCn AREA = iota + 1
	AreaGlobal
)

// 地区
var AREAText = map[AREA]string{
	AreaCn:     "https://usdk.ultrasdk.com/hu/v1/login/checkUserInfo.lg",
	AreaGlobal: "https://uglobal.ultrasdk.com/hu/v1/login/checkUserInfo.lg",
}

type USDK struct {
	productId    string // 游戏在SDK平台的应用码
	productKey   string // 游戏的APP_KEY
	callbackKey  string // SDK服务端为游戏的同步的秘钥
	checkUserUrl string

	ctx    context.Context
	client *http.Client
}

func NewUSDK(ctx context.Context, authUrl string, productId string, productKey string, callbackKey string, area AREA) (*USDK, error) {
	if _, ok := AREAText[area]; !ok {
		return nil, fmt.Errorf("miss Area")
	} else {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		return &USDK{
			ctx:         ctx,
			productId:   productId,
			productKey:  productKey,
			callbackKey: callbackKey,
			// checkUserUrl: u,
			checkUserUrl: authUrl,
			client: &http.Client{
				Timeout:   time.Second * 10,
				Transport: tr,
			},
		}, nil
	}
}

// md5值
func (u *USDK) md5hex(content string) string {
	data := []byte(content)
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has)
	return md5str
}

// 加密data
func (u *USDK) encodeData(body interface{}) string {
	data, _ := json.Marshal(body)
	str := base64.StdEncoding.EncodeToString(data)
	if len(str) > 51 {
		ss := []byte(str)
		swap_index := []int{1, 33, 10, 42, 18, 50, 19, 51}
		for i, n := 1, len(swap_index); i < n; i += 2 {
			l, r := swap_index[i-1], swap_index[i]
			ss[l], ss[r] = ss[r], ss[l]
		}
		return string(ss)
	}
	return str
}

// 签名
func (u *USDK) sign(content map[string]string, key string) string {
	var keyLst = make([]string, 0, len(content))
	for k, _ := range content {
		if k != "sign" {
			keyLst = append(keyLst, k)
		}
	}
	sort.Strings(keyLst) //排序字符串
	var buf = make([]string, 0, 4*len(content)+2)
	for _, k := range keyLst {
		v := content[k]
		if k != "" && v != "" {
			buf = append(buf, k, "=", v, "&")
		}
	}
	buf = append(buf, key)
	return u.md5hex(strings.Join(buf, ""))
}

// 登录校验
//
// return:
//
//	原始内容,
//	反序列AuthInfo,
//	错误信息
func (u *USDK) Login(cUid, cName, accessToken string) (respBody []byte, authInfo *AuthInfo, err error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	data := u.encodeData(map[string]string{"cUid": cUid, "cName": cName, "accessToken": accessToken})
	sign := u.sign(map[string]string{"data": data, "pcode": u.productId, "timestamp": timestamp}, u.productKey)
	form := url.Values{"data": {data}, "pcode": {u.productId}, "timestamp": {timestamp}, "sign": {sign}}
	req, err := http.NewRequestWithContext(u.ctx, "POST", u.checkUserUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("code=%d", resp.StatusCode)
	}
	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if len(respBody) > 5 && respBody[0] == '{' && respBody[len(respBody)-1] == '}' {
		authInfo = &AuthInfo{}
		err = json.Unmarshal(respBody, authInfo)
		if err == nil {
			return respBody, authInfo, nil
		}
		return respBody, nil, err
	}
	return respBody, nil, fmt.Errorf("invalid json")
}

// 支付结果校验
//
// 返回:
//
//	状态
//	原始内容（解密后）
//	反序列化后的结果
//	错误消息
func (u *USDK) PayCallback(data string, signStr string) (ret string, payData []byte, payInfo *PayResult, err error) {
	sign := u.sign(map[string]string{"data": data}, u.callbackKey)
	if !strings.EqualFold(sign, signStr) {
		return "FAIL", nil, nil, fmt.Errorf("签名验证失败,[%s]!=[%s]", sign, signStr)
	}
	payData, err = base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "FAIL", nil, nil, fmt.Errorf("解密失败 %s", err.Error())
	}
	payInfo = &PayResult{}
	err = json.Unmarshal(payData, payInfo)
	if err != nil {
		return "FAIL", payData, nil, fmt.Errorf("反序列化失败 %s", err.Error())
	}
	return "SUCCESS", payData, payInfo, nil
}
