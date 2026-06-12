import {LoginOutlined, ReloadOutlined} from '@ant-design/icons';
import {Alert, Button, Card, Form, Input, Typography} from 'antd';
import * as React from 'react';
import {useNavigate} from 'react-router-dom';
import {BrandMark} from '../components';
import {services} from '../../shared/services';
import type {CaptchaChallenge} from '../../shared/services/user-service';

const postLoginPath = '/settings';

export const LoginPage = () => {
    const navigate = useNavigate();
    const [form] = Form.useForm();
    const [loading, setLoading] = React.useState(false);
    const [captchaLoading, setCaptchaLoading] = React.useState(false);
    const [captcha, setCaptcha] = React.useState<CaptchaChallenge | null>(null);
    const [error, setError] = React.useState('');

    const loadCaptcha = React.useCallback(async () => {
        setCaptchaLoading(true);
        try {
            const nextCaptcha = await services.users.getCaptcha();
            setCaptcha(nextCaptcha);
            form.setFieldsValue({captchaAnswer: ''});
        } catch (err: any) {
            setCaptcha(null);
            setError(err?.message || 'Failed to load captcha');
        } finally {
            setCaptchaLoading(false);
        }
    }, [form]);

    React.useEffect(() => {
        let active = true;
        services.users
            .get()
            .then(user => {
                if (active && user.loggedIn) {
                    navigate(postLoginPath, {replace: true});
                }
            })
            .catch(() => undefined);
        return () => {
            active = false;
        };
    }, [navigate]);

    React.useEffect(() => {
        loadCaptcha();
    }, [loadCaptcha]);

    const submit = async (values: {username: string; password: string; captchaAnswer: string}) => {
        setLoading(true);
        setError('');
        try {
            await services.users.login(values.username, values.password, captcha?.captchaId || '', values.captchaAnswer);
            navigate(postLoginPath, {replace: true});
        } catch (err: any) {
            setError(err?.message || 'Login failed');
            await loadCaptcha();
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className='login-screen'>
            <Card className='login-panel'>
                <div className='login-panel__brand'>
                    <BrandMark size='large' />
                    <Typography.Title level={3}>Athena</Typography.Title>
                </div>
                {error && <Alert type='error' title={error} showIcon={true} />}
                <Form form={form} layout='vertical' onFinish={submit}>
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
                            <Button className='login-captcha__image' loading={captchaLoading} onClick={loadCaptcha} icon={!captcha ? <ReloadOutlined /> : undefined}>
                                {captcha && <img src={captcha.imageDataUrl} alt='Captcha' />}
                            </Button>
                        </div>
                    </Form.Item>
                    <Button block={true} type='primary' htmlType='submit' loading={loading} icon={<LoginOutlined />}>
                        Log in
                    </Button>
                </Form>
            </Card>
        </div>
    );
};
