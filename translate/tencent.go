package translate

import (
	"simpleTranslate/config"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tmt "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tmt/v20180321"
)

func newClient() (*tmt.Client, error) {
	configPath := config.GetConfigPath()
	config := config.GetConfig(configPath)
	cred := common.NewCredential(config.Tencent.SecretId, config.Tencent.SecretKey)
	clientProfile := profile.NewClientProfile()
	clientProfile.HttpProfile.Endpoint = "tmt.tencentcloudapi.com"
	if config.Tencent.Region == "" {
		config.Tencent.Region = "ap-beijing"
	}
	return tmt.NewClient(cred, config.Tencent.Region, clientProfile)
}

// 语种识别
func DetectLanguage(text string) (string, error) {
	client, err := newClient()
	if err != nil {
		return "", err
	}

	req := tmt.NewLanguageDetectRequest()
	req.Text = common.StringPtr(text)
	req.ProjectId = common.Int64Ptr(0)

	resp, err := client.LanguageDetect(req)

	if err != nil {
		return "", err
	}

	return *resp.Response.Lang, nil
}

// 翻译
func Translate(text, source, target string) (string, error) {
	client, err := newClient()
	if err != nil {
		return "", err
	}

	req := tmt.NewTextTranslateRequest()
	req.SourceText = common.StringPtr(text)
	req.Source = common.StringPtr(source)
	req.Target = common.StringPtr(target)
	req.ProjectId = common.Int64Ptr(0)

	resp, err := client.TextTranslate(req)
	if err != nil {
		return "", err
	}

	return *resp.Response.TargetText, nil
}
