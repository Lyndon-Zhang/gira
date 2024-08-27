package aliapi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
)

type StsGetCredentialResponse struct {
	ErrCode      int32  `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
	TmpSecretID  string `json:"TmpSecretId"`
	TmpSecretKey string `json:"TmpSecretKey"`
	SessionToken string `json:"Token"`
	Expiration   int64  `json:"expiration"`
}

type StsGetCredentialPolicyStatement struct {
	Action   []string
	Effect   string
	Resource []string
}

type StsGetCredentialPolicy struct {
	Version   string
	Statement []StsGetCredentialPolicyStatement
}

// 获取临时密钥
// 参考 sdk源码 https://github.com/aliyun/alibaba-cloud-sdk-go/blob/master/services/sts/client.go
// 参考 小程序 https://help.aliyun.com/document_detail/92883.html?spm=a2c4g.31923.0.0
// 参考 创建子用户，创建角色，角色添加权限 https://help.aliyun.com/document_detail/100624.html
func StsGetCredential(accessKeyId string, accessKeySecret string, ossRamRoleArn string, bucket string, region string) (*StsGetCredentialResponse, error) {
	response := &StsGetCredentialResponse{}
	client, err := sts.NewClientWithAccessKey(region, accessKeyId, accessKeySecret)
	if err != nil {
		return nil, err
	}
	policy := &StsGetCredentialPolicy{
		Version: "1",
		Statement: []StsGetCredentialPolicyStatement{
			{
				Action:   []string{"oss:PutObject"},
				Effect:   "Allow",
				Resource: []string{fmt.Sprintf("acs:oss:*:*:%s/*", bucket)},
			},
		},
	}
	policybyte, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}
	req := sts.CreateAssumeRoleRequest()
	req.RoleSessionName = bucket //"darenwo"
	req.RoleArn = ossRamRoleArn
	req.Policy = string(policybyte)
	req.DurationSeconds = requests.NewInteger(3600)
	req.SetScheme("https")
	req.SetHTTPSInsecure(true)
	r, err := client.AssumeRole(req)
	if err != nil {
		return nil, err
	}
	response.TmpSecretID = r.Credentials.AccessKeyId
	response.SessionToken = r.Credentials.SecurityToken
	response.TmpSecretKey = r.Credentials.AccessKeySecret
	if v, err := time.Parse(time.RFC3339, r.Credentials.Expiration); err != nil {
		response.Expiration = time.Now().Unix() + 3600
	} else {
		response.Expiration = v.Unix()
	}
	return response, nil
}
