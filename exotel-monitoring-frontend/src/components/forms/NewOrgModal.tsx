import React from 'react';
import {
  Modal, Form, Input, Select, Button, Space, Divider, Typography, Alert,
} from 'antd';
import { PlusOutlined, MinusCircleOutlined } from '@ant-design/icons';
import { useCreateAccount } from '@/hooks/useCreateAccount';
import type { CreateAccountPayload } from '@/api/accounts';

const { Text } = Typography;

interface Props {
  open: boolean;
  onClose: () => void;
}

const CLUSTERS = ['in1', 'in2', 'us1', 'ap1'];
const API_REGIONS = [
  { value: 'api',  label: 'India — api.exotel.com' },
  { value: 'sg1',  label: 'Singapore — sg1.exotel.com' },
  { value: 'us1',  label: 'US — us1.exotel.com' },
];
const PRIORITIES = ['P0', 'P1', 'P2', 'P3'];

const NewOrgModal: React.FC<Props> = ({ open, onClose }) => {
  const [form] = Form.useForm();
  const { mutate: create, isPending, isError, error } = useCreateAccount();

  const handleFinish = (values: CreateAccountPayload) => {
    create(values, {
      onSuccess: () => {
        form.resetFields();
        onClose();
      },
    });
  };

  return (
    <Modal
      title="Add New Organisation"
      open={open}
      onCancel={() => { form.resetFields(); onClose(); }}
      footer={null}
      width={620}
      destroyOnClose
    >
      {isError && (
        <Alert
          type="error"
          showIcon
          message="Creation failed"
          description={(error as Error)?.message}
          style={{ marginBottom: 16 }}
        />
      )}

      <Form
        form={form}
        layout="vertical"
        onFinish={handleFinish}
        initialValues={{ cluster: 'in1', subdomain: 'api', exophones: [{ priority: 'P1' }] }}
      >
        <Divider orientation="left" plain><Text type="secondary">Organisation Details</Text></Divider>

        <Form.Item name="account_name" label="Company Name" rules={[{ required: true, message: 'Enter company name' }]}>
          <Input placeholder="Cashify India" />
        </Form.Item>

        <Form.Item name="sid" label="Exotel Account SID" rules={[{ required: true, message: 'Enter the Exotel SID' }]}>
          <Input placeholder="CASHIFY_SID_001" />
        </Form.Item>

        <Space style={{ width: '100%' }} size={12}>
          <Form.Item
            name="subdomain"
            label="API Region"
            style={{ flex: 1 }}
            rules={[{ required: true, message: 'Select API region' }]}
            tooltip="The Exotel API cluster for your account. Indian accounts use 'India (api.exotel.com)'."
          >
            <Select options={API_REGIONS} />
          </Form.Item>
          <Form.Item name="cluster" label="Cluster" style={{ width: 120 }}>
            <Select options={CLUSTERS.map((c) => ({ value: c, label: c }))} />
          </Form.Item>
        </Space>

        <Divider orientation="left" plain><Text type="secondary">API Credentials</Text></Divider>

        <Form.Item name="api_key" label="API Key" rules={[{ required: true, message: 'Enter API key' }]}>
          <Input.Password placeholder="Exotel API key" />
        </Form.Item>
        <Form.Item name="api_token" label="API Token" rules={[{ required: true, message: 'Enter API token' }]}>
          <Input.Password placeholder="Exotel API token" />
        </Form.Item>

        <Divider orientation="left" plain><Text type="secondary">Exophones</Text></Divider>

        <Form.List
          name="exophones"
          rules={[{
            validator: async (_, phones) => {
              if (!phones || phones.length === 0) throw new Error('Add at least one exophone');
            },
          }]}
        >
          {(fields, { add, remove }, { errors }) => (
            <>
              {fields.map(({ key, name, ...rest }) => (
                <Space key={key} align="baseline" style={{ display: 'flex', marginBottom: 8 }}>
                  <Form.Item
                    {...rest}
                    name={[name, 'number']}
                    rules={[{ required: true, message: 'Enter phone number' }]}
                    style={{ flex: 1, marginBottom: 0 }}
                  >
                    <Input placeholder="+911140000001" style={{ width: 220 }} />
                  </Form.Item>
                  <Form.Item
                    {...rest}
                    name={[name, 'priority']}
                    style={{ marginBottom: 0 }}
                  >
                    <Select style={{ width: 90 }} options={PRIORITIES.map((p) => ({ value: p, label: p }))} />
                  </Form.Item>
                  {fields.length > 1 && (
                    <MinusCircleOutlined onClick={() => remove(name)} style={{ color: '#ff4d4f' }} />
                  )}
                </Space>
              ))}
              <Form.Item>
                <Button type="dashed" onClick={() => add({ priority: 'P1' })} icon={<PlusOutlined />} block>
                  Add Exophone
                </Button>
                <Form.ErrorList errors={errors} />
              </Form.Item>
            </>
          )}
        </Form.List>

        <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
          <Space>
            <Button onClick={() => { form.resetFields(); onClose(); }}>Cancel</Button>
            <Button type="primary" htmlType="submit" loading={isPending}>
              Create Organisation
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default NewOrgModal;
