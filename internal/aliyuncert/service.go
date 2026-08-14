package aliyuncert

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/certimate-go/certimate/internal/domain"
	"github.com/certimate-go/certimate/internal/domain/dtos"
	alicas "github.com/certimate-go/certimate/pkg/sdk3rd-trimmed/github.com/alibabacloud-go/cas-20200407/v4/client"
	xmaps "github.com/certimate-go/certimate/pkg/utils/maps"
)

const (
	// 免费证书资源包规格（仅中国大陆站点可用）
	productCodeFree3Months = "digicert-free-1-free" // DigiCert 单域名 DV，免费 3 个月
	productCodeFree1Year   = "symantec-free-1-free" // DigiCert 单域名 DV，免费 1 年
	productCodeDefault     = productCodeFree3Months
)

const (
	// 证书申请订单状态（CAS DescribeCertificateState 的返回值）
	applyStatusDomainVerify = "domain_verify" // 待完成域名验证
	applyStatusProcess      = "process"       // 审核中
	applyStatusCertificate  = "certificate"   // 已签发
	applyStatusVerifyFail   = "verify_fail"   // 审核失败
	applyStatusPayed        = "payed"         // 待申请
	applyStatusUnknown      = "unknow"        // 未知

	applyStatusDnsAdded = "dns_added" // 验证记录已添加（本地状态）
	applyStatusTimeout  = "timeout"   // 等待超时（本地状态）
)

const (
	applyWatchInterval   = 30 * time.Second
	applyWatchTimeout    = 2 * time.Hour
	applyTaskExpireAfter = 10 * time.Minute
)

type accessRepository interface {
	GetById(ctx context.Context, id string) (*domain.Access, error)
}

type applyTask struct {
	orderId    int64
	domain     string
	status     string
	message    string
	dnsInfo    *dnsRecordInfo
	dnsCleaned bool
	updatedAt  time.Time
}

type AliyunCertificateService struct {
	accessRepo   accessRepository
	applyTasks   map[int64]*applyTask
	applyTasksMu sync.Mutex
}

func NewAliyunCertificateService(accessRepo accessRepository) *AliyunCertificateService {
	return &AliyunCertificateService{
		accessRepo: accessRepo,
		applyTasks: make(map[int64]*applyTask),
	}
}

func (s *AliyunCertificateService) getAccessCredentials(ctx context.Context, accessId string) (*domain.AccessConfigForAliyun, error) {
	access, err := s.accessRepo.GetById(ctx, accessId)
	if err != nil {
		return nil, err
	}

	if access.Provider != string(domain.AccessProviderTypeAliyun) {
		return nil, fmt.Errorf("invalid access provider: expected '%s', got '%s'", domain.AccessProviderTypeAliyun, access.Provider)
	}

	credentials := domain.AccessConfigForAliyun{}
	if err := xmaps.Populate(access.Config, &credentials); err != nil {
		return nil, fmt.Errorf("failed to populate access config: %w", err)
	}

	if credentials.AccessKeyId == "" || credentials.AccessKeySecret == "" {
		return nil, fmt.Errorf("access config is incomplete: missing accessKeyId or accessKeySecret")
	}

	return &credentials, nil
}

func (s *AliyunCertificateService) ListCertificates(ctx context.Context, req *dtos.AliyunCertificateListReq) (*dtos.AliyunCertificateListResp, error) {
	credentials, err := s.getAccessCredentials(ctx, req.AccessId)
	if err != nil {
		return nil, err
	}

	client, err := createCasClient(credentials.AccessKeyId, credentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	var items []*dtos.AliyunCertificateListItem
	page := 1
	limit := 50
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		listReq := &alicas.ListUserCertificateOrderRequest{
			CurrentPage: tea.Int64(int64(page)),
			ShowSize:    tea.Int64(int64(limit)),
			OrderType:   tea.String("CERT"),
		}
		listResp, err := client.ListUserCertificateOrderWithContext(ctx, listReq, &dara.RuntimeOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to execute sdk request 'cas.ListUserCertificateOrder': %w", err)
		}

		if listResp.Body == nil {
			break
		}

		for _, cert := range listResp.Body.CertificateOrderList {
			items = append(items, &dtos.AliyunCertificateListItem{
				CertificateId: tea.Int64Value(cert.CertificateId),
				Name:          tea.StringValue(cert.Name),
				CommonName:    tea.StringValue(cert.CommonName),
				Sans:          tea.StringValue(cert.Sans),
				Issuer:        tea.StringValue(cert.Issuer),
				StartDate:     tea.StringValue(cert.StartDate),
				EndDate:       tea.StringValue(cert.EndDate),
				Expired:       tea.BoolValue(cert.Expired),
				Upload:        tea.BoolValue(cert.Upload),
			})
		}

		if len(listResp.Body.CertificateOrderList) < limit {
			break
		}

		page++
	}

	return &dtos.AliyunCertificateListResp{
		Total: int64(len(items)),
		Items: items,
	}, nil
}

func (s *AliyunCertificateService) GetQuota(ctx context.Context, req *dtos.AliyunCertificateQuotaReq) (*dtos.AliyunCertificateQuotaResp, error) {
	credentials, err := s.getAccessCredentials(ctx, req.AccessId)
	if err != nil {
		return nil, err
	}

	client, err := createCasClient(credentials.AccessKeyId, credentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	var items []*dtos.AliyunCertificateQuotaItem
	var lastErr error
	for _, productCode := range []string{productCodeFree3Months, productCodeFree1Year} {
		quotaReq := &alicas.DescribePackageStateRequest{
			ProductCode: tea.String(productCode),
		}
		quotaResp, err := client.DescribePackageStateWithContext(ctx, quotaReq, &dara.RuntimeOptions{})
		if err != nil {
			// 未购买该规格或当前账号不可用时跳过，避免阻塞页面
			lastErr = fmt.Errorf("failed to execute sdk request 'cas.DescribePackageState(productCode=%s)': %w", productCode, err)
			slog.Warn("query certificate package state failed", slog.String("productCode", productCode), slog.Any("error", err))
			continue
		}

		if quotaResp.Body == nil {
			lastErr = fmt.Errorf("unexpected nil response body for productCode '%s'", productCode)
			continue
		}

		items = append(items, &dtos.AliyunCertificateQuotaItem{
			ProductCode: tea.StringValue(quotaResp.Body.ProductCode),
			TotalCount:  tea.Int64Value(quotaResp.Body.TotalCount),
			UsedCount:   tea.Int64Value(quotaResp.Body.UsedCount),
			IssuedCount: tea.Int64Value(quotaResp.Body.IssuedCount),
		})
	}

	if len(items) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return &dtos.AliyunCertificateQuotaResp{Items: items}, nil
}

func (s *AliyunCertificateService) ApplyCertificate(ctx context.Context, req *dtos.AliyunCertificateApplyReq) (*dtos.AliyunCertificateApplyResp, error) {
	if req.Domain == "" || req.Username == "" || req.Phone == "" || req.Email == "" {
		return nil, fmt.Errorf("invalid params: domain, username, phone and email are required")
	}

	productCode := req.ProductCode
	if productCode == "" {
		productCode = productCodeDefault
	}

	credentials, err := s.getAccessCredentials(ctx, req.AccessId)
	if err != nil {
		return nil, err
	}

	client, err := createCasClient(credentials.AccessKeyId, credentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	// 创建证书申请（消耗免费证书资源包名额）
	createReq := &alicas.CreateCertificateRequestRequest{
		Domain:       tea.String(req.Domain),
		Username:     tea.String(req.Username),
		Phone:        tea.String(req.Phone),
		Email:        tea.String(req.Email),
		ValidateType: tea.String("DNS"),
		ProductCode:  tea.String(productCode),
	}
	createResp, err := client.CreateCertificateRequestWithContext(ctx, createReq, &dara.RuntimeOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to execute sdk request 'cas.CreateCertificateRequest': %w", err)
	}

	if createResp.Body == nil || createResp.Body.OrderId == nil {
		return nil, fmt.Errorf("unexpected nil response body from sdk request 'cas.CreateCertificateRequest'")
	}

	orderId := tea.Int64Value(createResp.Body.OrderId)
	result := &dtos.AliyunCertificateApplyResp{
		OrderId: orderId,
		Domain:  req.Domain,
		Status:  applyStatusUnknown,
	}

	// 获取域名验证信息与订单状态
	stateReq := &alicas.DescribeCertificateStateRequest{
		OrderId: tea.Int64(orderId),
	}
	stateResp, err := client.DescribeCertificateStateWithContext(ctx, stateReq, &dara.RuntimeOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to execute sdk request 'cas.DescribeCertificateState': %w", err)
	}

	if stateResp.Body == nil {
		return nil, fmt.Errorf("unexpected nil response body from sdk request 'cas.DescribeCertificateState'")
	}

	stateType := tea.StringValue(stateResp.Body.Type)
	result.Status = stateType

	// 需要 DNS 验证时，自动在云解析中添加验证记录，并启动后台轮询等待签发
	if stateType == applyStatusDomainVerify && strings.EqualFold(tea.StringValue(stateResp.Body.ValidateType), "DNS") {
		recordType := tea.StringValue(stateResp.Body.RecordType)
		recordDomain := tea.StringValue(stateResp.Body.RecordDomain)
		recordValue := tea.StringValue(stateResp.Body.RecordValue)

		dnsClient, err := createAlidnsClient(credentials.AccessKeyId, credentials.AccessKeySecret)
		if err != nil {
			return nil, err
		}

		dnsInfo, err := addVerificationRecord(ctx, dnsClient, req.Domain, recordDomain, recordType, recordValue)
		if err != nil {
			// 域名不在云解析或添加失败时，删除申请单以免继续消耗配额
			deleteReq := &alicas.DeleteCertificateRequestRequest{OrderId: tea.Int64(orderId)}
			if _, deleteErr := client.DeleteCertificateRequestWithContext(context.Background(), deleteReq, &dara.RuntimeOptions{}); deleteErr != nil {
				slog.Warn("failed to delete certificate request after dns failure", slog.Int64("orderId", orderId), slog.Any("error", deleteErr))
			}
			return nil, err
		}

		result.Status = applyStatusDnsAdded
		result.RecordType = recordType
		result.RecordDomain = recordDomain
		result.RecordValue = recordValue
		result.DnsZone = dnsInfo.Zone

		s.startApplyWatcher(credentials, req.Domain, orderId, dnsInfo)
	}

	return result, nil
}

func (s *AliyunCertificateService) GetApplyStatus(ctx context.Context, req *dtos.AliyunCertificateApplyStatusReq) (*dtos.AliyunCertificateApplyStatusResp, error) {
	// 优先返回本地后台任务状态
	s.applyTasksMu.Lock()
	task, ok := s.applyTasks[req.OrderId]
	if ok {
		if isFinalApplyStatus(task.status) && time.Since(task.updatedAt) > applyTaskExpireAfter {
			delete(s.applyTasks, req.OrderId)
			ok = false
		}
	}
	var resp *dtos.AliyunCertificateApplyStatusResp
	if ok {
		resp = &dtos.AliyunCertificateApplyStatusResp{
			OrderId: task.orderId,
			Status:  task.status,
			Message: task.message,
		}
	}
	s.applyTasksMu.Unlock()

	if resp != nil {
		return resp, nil
	}

	// 无本地任务（如服务重启后）时，实时查询云上状态
	credentials, err := s.getAccessCredentials(ctx, req.AccessId)
	if err != nil {
		return nil, err
	}

	client, err := createCasClient(credentials.AccessKeyId, credentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	stateReq := &alicas.DescribeCertificateStateRequest{
		OrderId: tea.Int64(req.OrderId),
	}
	stateResp, err := client.DescribeCertificateStateWithContext(ctx, stateReq, &dara.RuntimeOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to execute sdk request 'cas.DescribeCertificateState': %w", err)
	}

	if stateResp.Body == nil {
		return nil, fmt.Errorf("unexpected nil response body from sdk request 'cas.DescribeCertificateState'")
	}

	return &dtos.AliyunCertificateApplyStatusResp{
		OrderId: req.OrderId,
		Status:  tea.StringValue(stateResp.Body.Type),
	}, nil
}

func (s *AliyunCertificateService) RevokeCertificate(ctx context.Context, req *dtos.AliyunCertificateRevokeReq) (*dtos.AliyunCertificateRevokeResp, error) {
	credentials, err := s.getAccessCredentials(ctx, req.AccessId)
	if err != nil {
		return nil, err
	}

	client, err := createCasClient(credentials.AccessKeyId, credentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	// 吊销需要实例 ID（仅阿里云托管的证书实例支持吊销）
	detailReq := &alicas.GetUserCertificateDetailRequest{
		CertId:     tea.Int64(req.CertificateId),
		CertFilter: tea.Bool(false),
	}
	detailResp, err := client.GetUserCertificateDetailWithContext(ctx, detailReq, &dara.RuntimeOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to execute sdk request 'cas.GetUserCertificateDetail': %w", err)
	}

	if detailResp.Body == nil {
		return nil, fmt.Errorf("unexpected nil response body from sdk request 'cas.GetUserCertificateDetail'")
	}

	instanceId := tea.StringValue(detailResp.Body.InstanceId)
	if instanceId == "" {
		return nil, fmt.Errorf("certificate has no instance id, cannot be revoked")
	}

	revokeReq := &alicas.RevokeCertificateRequest{
		CertificateId: tea.Int64(req.CertificateId),
		InstanceId:    tea.String(instanceId),
	}
	_, err = client.RevokeCertificateWithContext(ctx, revokeReq, &dara.RuntimeOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to execute sdk request 'cas.RevokeCertificate': %w", err)
	}

	return &dtos.AliyunCertificateRevokeResp{}, nil
}

func (s *AliyunCertificateService) DeleteCertificate(ctx context.Context, req *dtos.AliyunCertificateDeleteReq) (*dtos.AliyunCertificateDeleteResp, error) {
	credentials, err := s.getAccessCredentials(ctx, req.AccessId)
	if err != nil {
		return nil, err
	}

	client, err := createCasClient(credentials.AccessKeyId, credentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	if req.Upload {
		// 上传的证书：从证书列表删除
		deleteReq := &alicas.DeleteUserCertificateRequest{
			CertId: tea.Int64(req.CertificateId),
		}
		_, err := client.DeleteUserCertificateWithContext(ctx, deleteReq, &dara.RuntimeOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to execute sdk request 'cas.DeleteUserCertificate': %w", err)
		}
	} else {
		// 阿里云签发的证书：删除申请单（已失败的申请单删除后不消耗配额）
		deleteReq := &alicas.DeleteCertificateRequestRequest{
			OrderId: tea.Int64(req.CertificateId),
		}
		_, err := client.DeleteCertificateRequestWithContext(ctx, deleteReq, &dara.RuntimeOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to execute sdk request 'cas.DeleteCertificateRequest': %w", err)
		}
	}

	return &dtos.AliyunCertificateDeleteResp{}, nil
}

// startApplyWatcher 启动后台协程轮询证书申请状态，在验证阶段结束后自动清理 DNS 验证记录。
func (s *AliyunCertificateService) startApplyWatcher(credentials *domain.AccessConfigForAliyun, domain string, orderId int64, dnsInfo *dnsRecordInfo) {
	task := &applyTask{
		orderId:   orderId,
		domain:    domain,
		status:    applyStatusDnsAdded,
		dnsInfo:   dnsInfo,
		updatedAt: time.Now(),
	}

	s.applyTasksMu.Lock()
	s.applyTasks[orderId] = task
	s.applyTasksMu.Unlock()

	go func() {
		deadline := time.Now().Add(applyWatchTimeout)
		ticker := time.NewTicker(applyWatchInterval)
		defer ticker.Stop()

		for {
			<-ticker.C

			stateType, err := s.queryApplyState(credentials, orderId)

			s.applyTasksMu.Lock()
			task.updatedAt = time.Now()
			if err != nil {
				task.message = err.Error()
				s.applyTasksMu.Unlock()
				continue
			}

			task.message = ""
			task.status = stateType
			s.applyTasksMu.Unlock()

			switch stateType {
			case applyStatusProcess, applyStatusCertificate, applyStatusVerifyFail:
				// 验证阶段结束，清理 DNS 验证记录
				if !task.dnsCleaned {
					dnsClient, clientErr := createAlidnsClient(credentials.AccessKeyId, credentials.AccessKeySecret)
					if clientErr == nil {
						if deleteErr := deleteVerificationRecord(context.Background(), dnsClient, task.dnsInfo); deleteErr != nil {
							slog.Warn("failed to delete dns verification record", slog.Int64("orderId", orderId), slog.Any("error", deleteErr))
						}
					}
					s.applyTasksMu.Lock()
					task.dnsCleaned = true
					s.applyTasksMu.Unlock()
				}
			}

			if isFinalApplyStatus(stateType) {
				return
			}

			if time.Now().After(deadline) {
				// 等待超时，清理 DNS 验证记录
				s.applyTasksMu.Lock()
				task.status = applyStatusTimeout
				task.message = "waiting for certificate issuance timed out"
				s.applyTasksMu.Unlock()

				if !task.dnsCleaned {
					dnsClient, clientErr := createAlidnsClient(credentials.AccessKeyId, credentials.AccessKeySecret)
					if clientErr == nil {
						if deleteErr := deleteVerificationRecord(context.Background(), dnsClient, task.dnsInfo); deleteErr != nil {
							slog.Warn("failed to delete dns verification record", slog.Int64("orderId", orderId), slog.Any("error", deleteErr))
						}
					}
					s.applyTasksMu.Lock()
					task.dnsCleaned = true
					s.applyTasksMu.Unlock()
				}
				return
			}
		}
	}()
}

func (s *AliyunCertificateService) queryApplyState(credentials *domain.AccessConfigForAliyun, orderId int64) (string, error) {
	client, err := createCasClient(credentials.AccessKeyId, credentials.AccessKeySecret)
	if err != nil {
		return "", err
	}

	stateReq := &alicas.DescribeCertificateStateRequest{
		OrderId: tea.Int64(orderId),
	}
	stateResp, err := client.DescribeCertificateStateWithContext(context.Background(), stateReq, &dara.RuntimeOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to execute sdk request 'cas.DescribeCertificateState': %w", err)
	}

	if stateResp.Body == nil {
		return "", fmt.Errorf("unexpected nil response body from sdk request 'cas.DescribeCertificateState'")
	}

	return tea.StringValue(stateResp.Body.Type), nil
}

func isFinalApplyStatus(status string) bool {
	switch status {
	case applyStatusCertificate, applyStatusVerifyFail, applyStatusTimeout:
		return true
	default:
		return false
	}
}
