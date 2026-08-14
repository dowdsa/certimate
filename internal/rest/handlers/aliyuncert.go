package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/certimate-go/certimate/internal/domain/dtos"
	"github.com/certimate-go/certimate/internal/rest/resp"
)

type aliyunCertificateService interface {
	ListCertificates(ctx context.Context, req *dtos.AliyunCertificateListReq) (*dtos.AliyunCertificateListResp, error)
	GetQuota(ctx context.Context, req *dtos.AliyunCertificateQuotaReq) (*dtos.AliyunCertificateQuotaResp, error)
	ApplyCertificate(ctx context.Context, req *dtos.AliyunCertificateApplyReq) (*dtos.AliyunCertificateApplyResp, error)
	GetApplyStatus(ctx context.Context, req *dtos.AliyunCertificateApplyStatusReq) (*dtos.AliyunCertificateApplyStatusResp, error)
	RevokeCertificate(ctx context.Context, req *dtos.AliyunCertificateRevokeReq) (*dtos.AliyunCertificateRevokeResp, error)
	DeleteCertificate(ctx context.Context, req *dtos.AliyunCertificateDeleteReq) (*dtos.AliyunCertificateDeleteResp, error)
}

type AliyunCertificatesHandler struct {
	service aliyunCertificateService
}

func NewAliyunCertificatesHandler(router *router.RouterGroup[*core.RequestEvent], service aliyunCertificateService) {
	handler := &AliyunCertificatesHandler{
		service: service,
	}

	group := router.Group("/aliyun-certificates")
	group.GET("", handler.listCertificates)
	group.GET("/quota", handler.getQuota)
	group.POST("/apply", handler.applyCertificate)
	group.GET("/apply-status", handler.getApplyStatus)
	group.POST("/revoke", handler.revokeCertificate)
	group.POST("/delete", handler.deleteCertificate)
}

func (handler *AliyunCertificatesHandler) listCertificates(e *core.RequestEvent) error {
	req := &dtos.AliyunCertificateListReq{
		AccessId: e.Request.URL.Query().Get("accessId"),
	}
	if req.AccessId == "" {
		return resp.Err(e, fmt.Errorf("invalid params: accessId is required"))
	}

	res, err := handler.service.ListCertificates(e.Request.Context(), req)
	if err != nil {
		return resp.Err(e, err)
	}

	return resp.Ok(e, res)
}

func (handler *AliyunCertificatesHandler) getQuota(e *core.RequestEvent) error {
	req := &dtos.AliyunCertificateQuotaReq{
		AccessId: e.Request.URL.Query().Get("accessId"),
	}
	if req.AccessId == "" {
		return resp.Err(e, fmt.Errorf("invalid params: accessId is required"))
	}

	res, err := handler.service.GetQuota(e.Request.Context(), req)
	if err != nil {
		return resp.Err(e, err)
	}

	return resp.Ok(e, res)
}

func (handler *AliyunCertificatesHandler) applyCertificate(e *core.RequestEvent) error {
	req := &dtos.AliyunCertificateApplyReq{}
	if err := e.BindBody(req); err != nil {
		return resp.Err(e, err)
	}

	res, err := handler.service.ApplyCertificate(e.Request.Context(), req)
	if err != nil {
		return resp.Err(e, err)
	}

	return resp.Ok(e, res)
}

func (handler *AliyunCertificatesHandler) getApplyStatus(e *core.RequestEvent) error {
	req := &dtos.AliyunCertificateApplyStatusReq{
		AccessId: e.Request.URL.Query().Get("accessId"),
	}
	if req.AccessId == "" || e.Request.URL.Query().Get("orderId") == "" {
		return resp.Err(e, fmt.Errorf("invalid params: accessId and orderId are required"))
	}

	orderId, err := strconv.ParseInt(e.Request.URL.Query().Get("orderId"), 10, 64)
	if err != nil {
		return resp.Err(e, fmt.Errorf("invalid params: orderId must be an integer"))
	}
	req.OrderId = orderId

	res, err := handler.service.GetApplyStatus(e.Request.Context(), req)
	if err != nil {
		return resp.Err(e, err)
	}

	return resp.Ok(e, res)
}

func (handler *AliyunCertificatesHandler) revokeCertificate(e *core.RequestEvent) error {
	req := &dtos.AliyunCertificateRevokeReq{}
	if err := e.BindBody(req); err != nil {
		return resp.Err(e, err)
	}

	res, err := handler.service.RevokeCertificate(e.Request.Context(), req)
	if err != nil {
		return resp.Err(e, err)
	}

	return resp.Ok(e, res)
}

func (handler *AliyunCertificatesHandler) deleteCertificate(e *core.RequestEvent) error {
	req := &dtos.AliyunCertificateDeleteReq{}
	if err := e.BindBody(req); err != nil {
		return resp.Err(e, err)
	}

	res, err := handler.service.DeleteCertificate(e.Request.Context(), req)
	if err != nil {
		return resp.Err(e, err)
	}

	return resp.Ok(e, res)
}
