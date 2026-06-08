import {LoginOutlined} from '@ant-design/icons';
import {Alert, Button, Card, Form, Input, Typography} from 'antd';
import * as React from 'react';
import {useNavigate} from 'react-router-dom';
import {BrandMark} from '../components';
import {services} from '../../shared/services';

const postLoginPath = '/settings';

export const LoginPage = () => {
    const navigate = useNavigate();
    const [form] = Form.useForm();
    const [loading, setLoading] = React.useState(false);
    const [error, setError] = React.useState('');

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

    const submit = async (values: {username: string; password: string}) => {
        setLoading(true);
        setError('');
        try {
            await services.users.login(values.username, values.password);
            navigate(postLoginPath, {replace: true});
        } catch (err: any) {
            setError(err?.message || 'Login failed');
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
                    <Button block={true} type='primary' htmlType='submit' loading={loading} icon={<LoginOutlined />}>
                        Log in
                    </Button>
                </Form>
            </Card>
        </div>
    );
};
