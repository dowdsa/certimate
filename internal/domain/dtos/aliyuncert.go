package dtos

type AliyunCertificateListReq struct {
	AccessId string `json:"accessId"`
}

type AliyunCertificateListResp struct {
	Total int64                        `json:"total"`
	Items []*AliyunCertificateListItem `json:"items"`
}

type AliyunCertificateListItem struct {
	CertificateId int64  `json:"certificateId"`
	Name          string `json:"name"`
	CommonName    string `json:"commonName"`
	Sans          string `json:"sans"`
	Issuer        string `json:"issuer"`
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	Expired       bool   `json:"expired"`
	Upload        bool   `json:"upload"`
}

type AliyunCertificateQuotaReq struct {
	AccessId string `json:"accessId"`
}

type AliyunCertificateQuotaResp struct {
	Items []*AliyunCertificateQuotaItem `json:"items"`
}

type AliyunCertificateQuotaItem struct {
	ProductCode string `json:"productCode"`
	TotalCount  int64  `json:"totalCount"`
	UsedCount   int64  `json:"usedCount"`
	IssuedCount int64  `json:"issuedCount"`
}

type AliyunCertificateApplyReq struct {
	AccessId    string `json:"accessId"`
	Domain      string `json:"domain"`
	Username    string `json:"username"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	ProductCode string `json:"productCode"`
}

type AliyunCertificateApplyResp struct {
	OrderId      int64  `json:"orderId"`
	Domain       string `json:"domain"`
	Status       string `json:"status"`
	RecordType   string `json:"recordType"`
	RecordDomain string `json:"recordDomain"`
	RecordValue  string `json:"recordValue"`
	DnsZone      string `json:"dnsZone"`
}

type AliyunCertificateApplyStatusReq struct {
	AccessId string `json:"accessId"`
	OrderId  int64  `json:"orderId"`
}

type AliyunCertificateApplyStatusResp struct {
	OrderId int64  `json:"orderId"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type AliyunCertificateRevokeReq struct {
	AccessId      string `json:"accessId"`
	CertificateId int64  `json:"certificateId"`
}

type AliyunCertificateRevokeResp struct{}

type AliyunCertificateDeleteReq struct {
	AccessId      string `json:"accessId"`
	CertificateId int64  `json:"certificateId"`
	Upload        bool   `json:"upload"`
}

type AliyunCertificateDeleteResp struct{}
