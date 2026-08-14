export const ALIYUN_CERTIFICATE_APPLY_STATUS = Object.freeze({
  DOMAIN_VERIFY: "domain_verify", // 待完成域名验证
  PROCESS: "process", // 审核中
  CERTIFICATE: "certificate", // 已签发
  VERIFY_FAIL: "verify_fail", // 审核失败
  PAYED: "payed", // 待申请
  UNKNOWN: "unknow", // 未知
  DNS_ADDED: "dns_added", // 验证记录已添加（本地状态）
  TIMEOUT: "timeout", // 等待超时（本地状态）
} as const);

export interface AliyunCertificateListItem {
  certificateId: number;
  name: string;
  commonName: string;
  sans: string;
  issuer: string;
  startDate: string;
  endDate: string;
  expired: boolean;
  upload: boolean;
}

export interface AliyunCertificateListResp {
  total: number;
  items: AliyunCertificateListItem[];
}

export interface AliyunCertificateQuotaItem {
  productCode: string;
  totalCount: number;
  usedCount: number;
  issuedCount: number;
}

export interface AliyunCertificateQuotaResp {
  items: AliyunCertificateQuotaItem[];
}

export interface AliyunCertificateApplyReq {
  accessId: string;
  domain: string;
  username: string;
  phone: string;
  email: string;
  productCode?: string;
}

export interface AliyunCertificateApplyResp {
  orderId: number;
  domain: string;
  status: string;
  recordType: string;
  recordDomain: string;
  recordValue: string;
  dnsZone: string;
}

export interface AliyunCertificateApplyStatusResp {
  orderId: number;
  status: string;
  message?: string;
}

export const ALIYUN_CERTIFICATE_PRODUCT_CODES = Object.freeze({
  DIGICERT_FREE_3MONTHS: "digicert-free-1-free",
  SYMANTEC_FREE_1YEAR: "symantec-free-1-free",
} as const);