import {
  type AliyunCertificateApplyReq,
  type AliyunCertificateApplyResp,
  type AliyunCertificateApplyStatusResp,
  type AliyunCertificateListResp,
  type AliyunCertificateQuotaResp,
} from "@/domain/aliyunCertificate";

import { get as httpGet, post as httpPost } from "./_api";

export const list = (accessId: string) => {
  return httpGet<AliyunCertificateListResp>({
    url: `/api/aliyun-certificates?accessId=${encodeURIComponent(accessId)}`,
  });
};

export const getQuota = (accessId: string) => {
  return httpGet<AliyunCertificateQuotaResp>({
    url: `/api/aliyun-certificates/quota?accessId=${encodeURIComponent(accessId)}`,
  });
};

export const apply = (req: AliyunCertificateApplyReq) => {
  return httpPost<AliyunCertificateApplyResp>({
    url: "/api/aliyun-certificates/apply",
    body: req,
  });
};

export const getApplyStatus = (accessId: string, orderId: number) => {
  return httpGet<AliyunCertificateApplyStatusResp>({
    url: `/api/aliyun-certificates/apply-status?accessId=${encodeURIComponent(accessId)}&orderId=${orderId}`,
  });
};

export const revoke = (accessId: string, certificateId: number) => {
  return httpPost({
    url: "/api/aliyun-certificates/revoke",
    body: {
      accessId,
      certificateId,
    },
  });
};

export const remove = (accessId: string, certificateId: number, upload: boolean) => {
  return httpPost({
    url: "/api/aliyun-certificates/delete",
    body: {
      accessId,
      certificateId,
      upload,
    },
  });
};