import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  IconBadgeFilled,
  IconCertificate,
  IconCirclePlus,
  IconCloudLock,
  IconDots,
  IconShieldCancel,
  IconTrash,
} from "@tabler/icons-react";
import { useInterval, useRequest } from "ahooks";
import { App, Button, Card, Col, Dropdown, Form, Input, Modal, Row, Select, Skeleton, Statistic, Table, type TableProps, Tag, Typography, theme } from "antd";
import dayjs from "dayjs";

import * as api from "@/api/aliyunCertificates";
import AccessSelect from "@/components/access/AccessSelect";
import Empty from "@/components/Empty";
import Show from "@/components/Show";
import { type AccessModel } from "@/domain/access";
import {
  ALIYUN_CERTIFICATE_APPLY_STATUS,
  ALIYUN_CERTIFICATE_PRODUCT_CODES,
  type AliyunCertificateApplyResp,
  type AliyunCertificateListItem,
  type AliyunCertificateQuotaItem,
} from "@/domain/aliyunCertificate";
import { useAccessesStore } from "@/stores/access";
import { unwrapErrMsg } from "@/utils/error";

const APPLY_FINAL_STATUSES = new Set<string>([
  ALIYUN_CERTIFICATE_APPLY_STATUS.CERTIFICATE,
  ALIYUN_CERTIFICATE_APPLY_STATUS.VERIFY_FAIL,
  ALIYUN_CERTIFICATE_APPLY_STATUS.TIMEOUT,
]);

const AliyunCertificateList = () => {
  const { t } = useTranslation();

  const { message } = App.useApp();

  const accesses = useAccessesStore((state) => state.accesses) ?? [];

  const [access, setAccess] = useState<AccessModel>();

  const [applyModalOpen, setApplyModalOpen] = useState(false);
  const [applyResult, setApplyResult] = useState<AliyunCertificateApplyResp>();

  useEffect(() => {
    useAccessesStore.getState().fetchAccesses(false);
  }, []);

  const handleAccessChange = (accessId?: string) => {
    const nextAccess = accesses.find((item) => item.id === accessId);
    setAccess(nextAccess);
  };

  const { data: listResp, loading: listLoading, refresh: refreshList } = useRequest(
    () => (access ? api.list(access.id) : Promise.resolve(undefined)),
    {
      ready: !!access,
      refreshDeps: [access?.id],
      onError: (err) => {
        message.error(unwrapErrMsg(err));
      },
    }
  );

  const { data: quotaResp, loading: quotaLoading } = useRequest(
    () => (access ? api.getQuota(access.id) : Promise.resolve(undefined)),
    {
      ready: !!access,
      refreshDeps: [access?.id],
      onError: (err) => {
        console.error(err);
      },
    }
  );

  return (
    <>
      <div className="mt-4 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <AccessSelect
            className="w-64"
            placeholder={t("aliyunCert.access.placeholder")}
            value={access?.id}
            onChange={handleAccessChange}
          />
          <Button icon={<IconCirclePlus size="1.25em" stroke="1.25" />} type="primary" disabled={!access} onClick={() => setApplyModalOpen(true)}>
            {t("aliyunCert.apply.button")}
          </Button>
        </div>
      </div>

      <Show when={!!access}>
        {access && (
          <div className="my-4">
            <QuotaCards items={quotaResp?.data?.items} loading={quotaLoading} />

            <div className="mt-6">
              <CertificateTable accessId={access.id} data={listResp?.data?.items} loading={listLoading} onChanged={refreshList} />
            </div>
          </div>
        )}
      </Show>

      <Show when={!access}>
        <div className="mt-12 text-center text-gray-400">{t("aliyunCert.empty.select_access")}</div>
      </Show>

      <ApplyModal
        accessId={access?.id}
        open={applyModalOpen}
        onClose={(result) => {
          setApplyModalOpen(false);
          if (result) {
            setApplyResult(result);
          }
        }}
      />

      <ApplyStatusPanel
        accessId={access?.id}
        applyResult={applyResult}
        onDone={() => {
          refreshList();
        }}
        onDismiss={() => setApplyResult(void 0)}
      />
    </>
  );
};

const QuotaCards = ({ items, loading }: { items?: AliyunCertificateQuotaItem[]; loading?: boolean }) => {
  const { t } = useTranslation();

  const { token: themeToken } = theme.useToken();

  if (loading && !items) {
    return <Skeleton active paragraph={{ rows: 2 }} />;
  }

  if (!items || items.length === 0) {
    return null;
  }

  const renderLabel = (productCode: string) => {
    const is3Months = productCode === ALIYUN_CERTIFICATE_PRODUCT_CODES.DIGICERT_FREE_3MONTHS;
    if (is3Months) {
      return t("aliyunCert.quota.free_3months");
    }
    const is1Year = productCode === ALIYUN_CERTIFICATE_PRODUCT_CODES.SYMANTEC_FREE_1YEAR;
    if (is1Year) {
      return t("aliyunCert.quota.free_1year");
    }
    return productCode;
  };

  return (
    <Row gutter={[16, 16]}>
      {items.map((item) => {
        const left = item.totalCount - item.usedCount;
        return (
          <Col key={item.productCode} span={8} md={8} xs={24}>
            <Card variant="borderless" style={{ border: `1px solid ${themeToken.colorBorderSecondary}` }}>
              <Statistic
                title={renderLabel(item.productCode)}
                value={left}
                prefix={<IconBadgeFilled size="1.25em" color={themeToken.colorSuccess} />}
                suffix={t("aliyunCert.quota.left")}
              />
              <Typography.Text type="secondary">{t("aliyunCert.quota.used", { used: item.usedCount, total: item.totalCount })}</Typography.Text>
            </Card>
          </Col>
        );
      })}
    </Row>
  );
};

const CertificateTable = ({
  accessId,
  data,
  loading,
  onChanged,
}: {
  accessId: string;
  data?: AliyunCertificateListItem[];
  loading?: boolean;
  onChanged: () => void;
}) => {
  const { t } = useTranslation();

  const { message, modal } = App.useApp();

  const { runAsync: revokeRun } = useRequest((certificateId: number) => api.revoke(accessId, certificateId), {
    manual: true,
    onSuccess: () => {
      message.success(t("common.text.operation_succeeded"));
      onChanged();
    },
    onError: (err) => {
      message.error(unwrapErrMsg(err));
    },
  });

  const { runAsync: removeRun } = useRequest((certificateId: number, upload: boolean) => api.remove(accessId, certificateId, upload), {
    manual: true,
    onSuccess: () => {
      message.success(t("common.text.operation_succeeded"));
      onChanged();
    },
    onError: (err) => {
      message.error(unwrapErrMsg(err));
    },
  });

  const tableColumns: TableProps<AliyunCertificateListItem>["columns"] = [
    {
      key: "commonName",
      title: t("aliyunCert.props.common_name"),
      render: (_, record) => <Typography.Text ellipsis>{record.commonName || record.name || `#${record.certificateId}`}</Typography.Text>,
    },
    {
      key: "sans",
      title: t("aliyunCert.props.sans"),
      responsive: ["lg"],
      render: (_, record) => <Typography.Text ellipsis>{record.sans || "-"}</Typography.Text>,
    },
    {
      key: "validity",
      title: t("aliyunCert.props.validity"),
      render: (_, record) => {
        const start = dayjs(record.startDate, "YYYY-MM-DD", true);
        const end = dayjs(record.endDate, "YYYY-MM-DD", true);

        if (!start.isValid() || !end.isValid()) {
          return <Typography.Text type="secondary">-</Typography.Text>;
        }

        return (
          <div className="flex max-w-full flex-col gap-1 truncate">
            <Typography.Text ellipsis>
              {start.format("YYYY-MM-DD")} ~ <span className="font-medium">{end.format("YYYY-MM-DD")}</span>
            </Typography.Text>
            <Typography.Text ellipsis type={record.expired ? "danger" : "secondary"}>
              {record.expired ? t("aliyunCert.props.expired") : t("aliyunCert.props.valid")}
            </Typography.Text>
          </div>
        );
      },
    },
    {
      key: "issuer",
      title: t("aliyunCert.props.issuer"),
      responsive: ["lg"],
      render: (_, record) => <Typography.Text ellipsis>{record.issuer || "-"}</Typography.Text>,
    },
    {
      key: "source",
      title: t("aliyunCert.props.source"),
      render: (_, record) =>
        record.upload ? (
          <Tag color="default">{t("aliyunCert.props.uploaded")}</Tag>
        ) : (
          <Tag color="blue">{t("aliyunCert.props.issued_by_aliyun")}</Tag>
        ),
    },
    {
      key: "$action",
      align: "end",
      width: 64,
      render: (_, record) => (
        <Dropdown
          trigger={["click"]}
          menu={{
            items: [
              {
                key: "revoke",
                label: t("aliyunCert.action.revoke"),
                icon: <IconShieldCancel size="1em" />,
                disabled: record.upload,
                onClick: () => {
                  modal.confirm({
                    title: t("aliyunCert.confirm.revoke_title"),
                    icon: <IconShieldCancel />,
                    okButtonProps: { danger: true },
                    onOk: () => revokeRun(record.certificateId),
                  });
                },
              },
              {
                key: "delete",
                label: t("aliyunCert.action.delete"),
                icon: <IconTrash size="1em" />,
                danger: true,
                onClick: () => {
                  modal.confirm({
                    title: t("aliyunCert.confirm.delete_title"),
                    icon: <IconTrash />,
                    okButtonProps: { danger: true },
                    onOk: () => removeRun(record.certificateId, record.upload),
                  });
                },
              },
            ],
          }}
        >
          <Button icon={<IconDots />} shape="circle" type="text" />
        </Dropdown>
      ),
    },
  ];

  return (
    <Table
      rowKey={(record) => `${record.certificateId}`}
      columns={tableColumns}
      dataSource={data}
      loading={loading}
      locale={{
        emptyText: <Empty description={t("aliyunCert.empty.no_certificates")} icon={<IconCertificate size="3.5em" stroke="1" />} />,
      }}
      pagination={false}
      scroll={{ x: "max-content" }}
    />
  );
};

const ApplyModal = ({
  accessId,
  open,
  onClose,
}: {
  accessId?: string;
  open: boolean;
  onClose: (result?: AliyunCertificateApplyResp) => void;
}) => {
  const { t } = useTranslation();

  const { message } = App.useApp();

  const [form] = Form.useForm();

  const { runAsync: applyRun, loading: applyLoading } = useRequest(
    (values: Record<string, any>) =>
      api.apply({
        accessId: accessId!,
        domain: values.domain,
        username: values.username,
        phone: values.phone,
        email: values.email,
        productCode: values.productCode,
      }),
    {
      manual: true,
      onSuccess: (resp) => {
        message.success(t("aliyunCert.apply.success"));
        onClose(resp.data);
      },
      onError: (err) => {
        message.error(unwrapErrMsg(err));
      },
    }
  );

  useEffect(() => {
    if (open) {
      form.resetFields();
      form.setFieldsValue({ productCode: ALIYUN_CERTIFICATE_PRODUCT_CODES.DIGICERT_FREE_3MONTHS });
    }
  }, [open, form]);

  return (
    <Modal
      open={open}
      title={t("aliyunCert.apply.title")}
      confirmLoading={applyLoading}
      okText={t("aliyunCert.apply.submit")}
      onOk={form.submit}
      onCancel={() => onClose()}
      afterClose={() => form.resetFields()}
    >
      <Form form={form} layout="vertical" initialValues={{ productCode: ALIYUN_CERTIFICATE_PRODUCT_CODES.DIGICERT_FREE_3MONTHS }} onFinish={applyRun}>
        <Form.Item
          label={t("aliyunCert.apply.domain")}
          name="domain"
          rules={[{ required: true }, { pattern: /^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/ }]}
        >
          <Input placeholder="example.com" />
        </Form.Item>
        <Form.Item label={t("aliyunCert.apply.product_code")} name="productCode">
          <Select
            options={[
              { value: ALIYUN_CERTIFICATE_PRODUCT_CODES.DIGICERT_FREE_3MONTHS, label: t("aliyunCert.quota.free_3months") },
              { value: ALIYUN_CERTIFICATE_PRODUCT_CODES.SYMANTEC_FREE_1YEAR, label: t("aliyunCert.quota.free_1year") },
            ]}
          />
        </Form.Item>
        <Form.Item label={t("aliyunCert.apply.username")} name="username" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item label={t("aliyunCert.apply.phone")} name="phone" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item label={t("aliyunCert.apply.email")} name="email" rules={[{ required: true }, { type: "email" }]}>
          <Input />
        </Form.Item>
      </Form>
    </Modal>
  );
};

const ApplyStatusPanel = ({
  accessId,
  applyResult,
  onDone,
  onDismiss,
}: {
  accessId?: string;
  applyResult?: AliyunCertificateApplyResp;
  onDone: () => void;
  onDismiss: () => void;
}) => {
  const { t } = useTranslation();

  const { message } = App.useApp();

  const [status, setStatus] = useState<string>();
  const [detail, setDetail] = useState<string>();
  const [finished, setFinished] = useState(false);

  const lastOrderIdRef = useRef<number>();

  useEffect(() => {
    if (!applyResult) {
      setStatus(void 0);
      setDetail(void 0);
      setFinished(false);
      return;
    }
    lastOrderIdRef.current = applyResult.orderId;
    setStatus(applyResult.status);
    setFinished(false);
  }, [applyResult]);

  useInterval(
    () => {
      if (!applyResult || !accessId || finished) {
        return;
      }

      api
        .getApplyStatus(accessId, applyResult.orderId)
        .then((resp) => {
          const next = resp.data;
          if (next.orderId !== lastOrderIdRef.current) {
            return;
          }
          setStatus(next.status);
          setDetail(next.message);
          if (APPLY_FINAL_STATUSES.has(next.status)) {
            setFinished(true);
            if (next.status === ALIYUN_CERTIFICATE_APPLY_STATUS.CERTIFICATE) {
              message.success(t("aliyunCert.apply.issued"));
              onDone();
            }
          }
        })
        .catch((err) => {
          console.error(err);
        });
    },
    applyResult && !finished ? 10000 : void 0,
    { immediate: true }
  );

  if (!applyResult) {
    return null;
  }

  const statusColor = (value?: string) => {
    switch (value) {
      case ALIYUN_CERTIFICATE_APPLY_STATUS.CERTIFICATE:
        return "success";
      case ALIYUN_CERTIFICATE_APPLY_STATUS.VERIFY_FAIL:
      case ALIYUN_CERTIFICATE_APPLY_STATUS.TIMEOUT:
        return "error";
      case ALIYUN_CERTIFICATE_APPLY_STATUS.DNS_ADDED:
      case ALIYUN_CERTIFICATE_APPLY_STATUS.DOMAIN_VERIFY:
      case ALIYUN_CERTIFICATE_APPLY_STATUS.PROCESS:
        return "processing";
      default:
        return "default";
    }
  };

  const statusLabel = (value?: string) => {
    switch (value) {
      case ALIYUN_CERTIFICATE_APPLY_STATUS.DNS_ADDED:
        return t("aliyunCert.status.dns_added");
      case ALIYUN_CERTIFICATE_APPLY_STATUS.DOMAIN_VERIFY:
        return t("aliyunCert.status.domain_verify");
      case ALIYUN_CERTIFICATE_APPLY_STATUS.PROCESS:
        return t("aliyunCert.status.process");
      case ALIYUN_CERTIFICATE_APPLY_STATUS.CERTIFICATE:
        return t("aliyunCert.status.certificate");
      case ALIYUN_CERTIFICATE_APPLY_STATUS.VERIFY_FAIL:
        return t("aliyunCert.status.verify_fail");
      case ALIYUN_CERTIFICATE_APPLY_STATUS.PAYED:
        return t("aliyunCert.status.payed");
      case ALIYUN_CERTIFICATE_APPLY_STATUS.TIMEOUT:
        return t("aliyunCert.status.timeout");
      case ALIYUN_CERTIFICATE_APPLY_STATUS.UNKNOWN:
      default:
        return value ?? "-";
    }
  };

  return (
    <div className="container mt-4">
      <Card
        title={
          <span className="flex items-center gap-2">
            <IconCloudLock size="1em" />
            {t("aliyunCert.apply.status_title")}
          </span>
        }
        extra={
          <Button type="link" onClick={onDismiss}>
            {t("common.button.close")}
          </Button>
        }
      >
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <Typography.Text>{t("aliyunCert.apply.order_id")}:</Typography.Text>
            <Typography.Text copyable strong>
              {applyResult.orderId}
            </Typography.Text>
            <Tag color={statusColor(status)}>{statusLabel(status)}</Tag>
          </div>

          <Show when={status === ALIYUN_CERTIFICATE_APPLY_STATUS.DNS_ADDED}>
            <Typography.Paragraph type="secondary" className="mb-1">
              {t("aliyunCert.apply.dns_hint")}
            </Typography.Paragraph>
            <div className="flex flex-col gap-1">
              <div>
                <Typography.Text type="secondary">{t("aliyunCert.props.record_zone")}: </Typography.Text>
                <Typography.Text code>{applyResult.dnsZone}</Typography.Text>
              </div>
              <div>
                <Typography.Text type="secondary">{t("aliyunCert.props.record_host")}: </Typography.Text>
                <Typography.Text code>
                  {applyResult.recordDomain}
                  {applyResult.recordDomain.includes(applyResult.dnsZone) ? "" : `.${applyResult.domain}`}
                </Typography.Text>
              </div>
              <div>
                <Typography.Text type="secondary">{t("aliyunCert.props.record_type")}: </Typography.Text>
                <Typography.Text code>{applyResult.recordType}</Typography.Text>
              </div>
              <div>
                <Typography.Text type="secondary">{t("aliyunCert.props.record_value")}: </Typography.Text>
                <Typography.Text code copyable>
                  {applyResult.recordValue}
                </Typography.Text>
              </div>
            </div>
          </Show>

          <Show when={!!detail}>
            <Typography.Paragraph type="warning" className="mb-0">
              {detail}
            </Typography.Paragraph>
          </Show>
        </div>
      </Card>
    </div>
  );
};

export default AliyunCertificateList;