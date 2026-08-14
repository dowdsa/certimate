package aliyuncert

import (
	"context"
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"

	alidns "github.com/certimate-go/certimate/pkg/sdk3rd-trimmed/github.com/alibabacloud-go/alidns-20150109/v4/client"
)

type dnsRecordInfo struct {
	Zone     string
	RR       string
	RecordId string
}

// resolveDnsZone 在账号托管的云解析域名中查找与证书域名最匹配（最长后缀）的 zone，
// 并返回证书域名在 zone 之外的子域部分。
func resolveDnsZone(ctx context.Context, client *alidns.Client, certDomain string) (zone string, sub string, err error) {
	domain := strings.ToLower(strings.TrimPrefix(certDomain, "*."))

	// 拉取账号下全部云解析域名
	var zones []string
	page := int64(1)
	pageSize := int64(100)
	for {
		describeDomainsReq := &alidns.DescribeDomainsRequest{
			PageNumber: tea.Int64(page),
			PageSize:   tea.Int64(pageSize),
		}
		describeDomainsResp, err := client.DescribeDomainsWithContext(ctx, describeDomainsReq, &dara.RuntimeOptions{})
		if err != nil {
			return "", "", fmt.Errorf("failed to execute sdk request 'alidns.DescribeDomains': %w", err)
		}

		if describeDomainsResp.Body == nil || describeDomainsResp.Body.Domains == nil {
			break
		}

		for _, d := range describeDomainsResp.Body.Domains.Domain {
			zones = append(zones, strings.ToLower(tea.StringValue(d.DomainName)))
		}

		if int64(len(describeDomainsResp.Body.Domains.Domain)) < pageSize {
			break
		}

		page++
	}

	// 匹配最长后缀的 zone
	bestZone := ""
	for _, z := range zones {
		if z != "" && (domain == z || strings.HasSuffix(domain, "."+z)) {
			if len(z) > len(bestZone) {
				bestZone = z
			}
		}
	}

	if bestZone == "" {
		return "", "", fmt.Errorf("domain '%s' is not hosted in Alibaba Cloud DNS (alidns)", certDomain)
	}

	sub = strings.TrimSuffix(strings.TrimSuffix(domain, bestZone), ".")
	return bestZone, sub, nil
}

// addVerificationRecord 在云解析中添加证书验证记录；若已存在相同记录则复用，若存在但值不同则更新。
func addVerificationRecord(ctx context.Context, client *alidns.Client, certDomain, recordDomain, recordType, recordValue string) (*dnsRecordInfo, error) {
	zone, sub, err := resolveDnsZone(ctx, client, certDomain)
	if err != nil {
		return nil, err
	}

	rr := recordDomain
	if sub != "" {
		rr = recordDomain + "." + sub
	}

	// 查询同 zone 下是否已存在同名记录
	describeDomainRecordsReq := &alidns.DescribeDomainRecordsRequest{
		DomainName: tea.String(zone),
		RRKeyWord:  tea.String(rr),
		SearchMode: tea.String("EXACT"),
		PageSize:   tea.Int64(100),
	}
	describeDomainRecordsResp, err := client.DescribeDomainRecordsWithContext(ctx, describeDomainRecordsReq, &dara.RuntimeOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to execute sdk request 'alidns.DescribeDomainRecords': %w", err)
	}

	if describeDomainRecordsResp.Body != nil && describeDomainRecordsResp.Body.DomainRecords != nil {
		for _, rec := range describeDomainRecordsResp.Body.DomainRecords.Record {
			if !strings.EqualFold(tea.StringValue(rec.RR), rr) {
				continue
			}

			if tea.StringValue(rec.Value) == recordValue {
				return &dnsRecordInfo{
					Zone:     zone,
					RR:       rr,
					RecordId: tea.StringValue(rec.RecordId),
				}, nil
			}

			// 已存在相同主机记录但值不同，更新之
			updateDomainRecordReq := &alidns.UpdateDomainRecordRequest{
				RecordId: rec.RecordId,
				RR:       tea.String(rr),
				Type:     tea.String(recordType),
				Value:    tea.String(recordValue),
				TTL:      tea.Int64(600),
			}
			_, err := client.UpdateDomainRecordWithContext(ctx, updateDomainRecordReq, &dara.RuntimeOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to execute sdk request 'alidns.UpdateDomainRecord': %w", err)
			}

			return &dnsRecordInfo{
				Zone:     zone,
				RR:       rr,
				RecordId: tea.StringValue(rec.RecordId),
			}, nil
		}
	}

	// 新增验证记录
	addDomainRecordReq := &alidns.AddDomainRecordRequest{
		DomainName: tea.String(zone),
		RR:         tea.String(rr),
		Type:       tea.String(recordType),
		Value:      tea.String(recordValue),
		TTL:        tea.Int64(600),
	}
	addDomainRecordResp, err := client.AddDomainRecordWithContext(ctx, addDomainRecordReq, &dara.RuntimeOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to execute sdk request 'alidns.AddDomainRecord': %w", err)
	}

	return &dnsRecordInfo{
		Zone:     zone,
		RR:       rr,
		RecordId: tea.StringValue(addDomainRecordResp.Body.RecordId),
	}, nil
}

func deleteVerificationRecord(ctx context.Context, client *alidns.Client, info *dnsRecordInfo) error {
	if info == nil || info.RecordId == "" {
		return nil
	}

	deleteDomainRecordReq := &alidns.DeleteDomainRecordRequest{
		RecordId: tea.String(info.RecordId),
	}
	_, err := client.DeleteDomainRecordWithContext(ctx, deleteDomainRecordReq, &dara.RuntimeOptions{})
	if err != nil {
		return fmt.Errorf("failed to execute sdk request 'alidns.DeleteDomainRecord': %w", err)
	}

	return nil
}
