import {LoginOutlined, ReloadOutlined} from '@ant-design/icons';
import {Alert, Button, Card, Form, Input, Typography} from 'antd';
import * as React from 'react';
import {useLocation, useNavigate} from 'react-router-dom';
import {BrandMark} from '../components';
import {services} from '../shared/services';
import {ACCOUNT_MAINTENANCE_MESSAGE, isAccountMaintenanceError, requestErrorMessage} from '../shared/services/requests';
import type {CaptchaChallenge} from '../shared/services/user-service';

const postLoginPath = '/settings';

export const LoginPage = (props: {onAuthenticated: () => Promise<boolean>}) => {
    const navigate = useNavigate();
    const location = useLocation();
    const [form] = Form.useForm();
    const maintenanceReason = new URLSearchParams(location.search).get('reason') === 'maintenance';
    const [loading, setLoading] = React.useState(false);
    const [captchaLoading, setCaptchaLoading] = React.useState(false);
    const [captcha, setCaptcha] = React.useState<CaptchaChallenge | null>(null);
    const [maintenance, setMaintenance] = React.useState(maintenanceReason);
    const [error, setError] = React.useState('');

    React.useEffect(() => {
        if (maintenanceReason) {
            setMaintenance(true);
        }
    }, [maintenanceReason]);

    const loadCaptcha = React.useCallback(async () => {
        setCaptchaLoading(true);
        try {
            const nextCaptcha = await services.users.getCaptcha();
            setCaptcha(nextCaptcha);
            form.setFieldsValue({captchaAnswer: ''});
        } catch (err: any) {
            setCaptcha(null);
            if (isAccountMaintenanceError(err)) {
                setMaintenance(true);
                setError('');
            } else {
                setError(requestErrorMessage(err, 'Failed to load captcha'));
            }
        } finally {
            setCaptchaLoading(false);
        }
    }, [form]);

    React.useEffect(() => {
        loadCaptcha();
    }, [loadCaptcha]);

    const submit = async (values: {username: string; password: string; captchaAnswer: string}) => {
        setLoading(true);
        setMaintenance(maintenanceReason);
        setError('');
        try {
            await services.users.login(values.username, values.password, captcha?.captchaId || '', values.captchaAnswer);
        } catch (err: any) {
            if (isAccountMaintenanceError(err)) {
                setMaintenance(true);
            } else {
                setError(requestErrorMessage(err, 'Login failed'));
            }
            await loadCaptcha();
            setLoading(false);
            return;
        }
        try {
            if (await props.onAuthenticated()) {
                navigate(postLoginPath, {replace: true});
            }
        } finally {
            setLoading(false);
        }
    };

    return (
        <main className='login-screen'>
            <Card className='login-panel'>
                <div className='login-panel__brand'>
                    <BrandMark size='large' />
                    <Typography.Title level={3}>Athena</Typography.Title>
                    <Typography.Text type='secondary'>Operations Console</Typography.Text>
                </div>
                {maintenance && <Alert type='warning' title={ACCOUNT_MAINTENANCE_MESSAGE} showIcon={true} />}
                {error && <Alert type='error' title={error} showIcon={true} />}
                <Form form={form} layout='vertical' aria-label='Athena login' onFinish={submit}>
                    <Form.Item name='username' label='Username' rules={[{required: true}]}>
                        <Input autoComplete='username' />
                    </Form.Item>
                    <Form.Item name='password' label='Password' rules={[{required: true}]}>
                        <Input.Password autoComplete='current-password' />
                    </Form.Item>
                    <Form.Item label='Captcha' required={true}>
                        <div className='login-captcha'>
                            <Form.Item name='captchaAnswer' noStyle={true} rules={[{required: true}]}>
                                <Input autoComplete='off' maxLength={5} />
                            </Form.Item>
                            <Button
                                className='login-captcha__image'
                                aria-label='Refresh captcha'
                                loading={captchaLoading}
                                onClick={loadCaptcha}
                                icon={!captcha ? <ReloadOutlined /> : undefined}>
                                {captcha && <img src={captcha.imageDataUrl} alt='Captcha' />}
                            </Button>
                        </div>
                    </Form.Item>
                    <Button block={true} type='primary' htmlType='submit' loading={loading} icon={<LoginOutlined />}>
                        Log in
                    </Button>
                </Form>
            </Card>
        </main>
    );
};
