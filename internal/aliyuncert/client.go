package aliyuncert

import (
	"fmt"

	aliopen "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"

	alidns "github.com/certimate-go/certimate/pkg/sdk3rd-trimmed/github.com/alibabacloud-go/alidns-20150109/v4/client"
	alicas "github.com/certimate-go/certimate/pkg/sdk3rd-trimmed/github.com/alibabacloud-go/cas-20200407/v4/client"
)

func createCasClient(accessKeyId, accessKeySecret string) (*alicas.Client, error) {
	config := &aliopen.Config{
		Endpoint:        tea.String("cas.aliyuncs.com"),
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
	}

	client, err := alicas.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	return client, nil
}

func createAlidnsClient(accessKeyId, accessKeySecret string) (*alidns.Client, error) {
	config := &aliopen.Config{
		Endpoint:        tea.String("alidns.aliyuncs.com"),
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
	}

	client, err := alidns.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	return client, nil
}
