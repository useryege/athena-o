import {Alert, Button, Card, Typography} from 'antd';
import * as React from 'react';
import {useLocation} from 'react-router-dom';
import googleMark from '../../assets/images/google-g.svg';
import {BrandMark} from '../components';
import {readAdminLoginReturnTo} from '../shared/login-navigation';
import {deploymentPath} from '../shared/runtime-base';

const alerts: Record<string, {type: 'error' | 'warning'; message: string}> = {
    google_cancelled: {type: 'warning', message: 'Google sign-in was cancelled. Try again when you are ready.'},
    google_not_allowed: {type: 'error', message: 'This Google identity is not allowed to administer Athena.'},
    google_state_invalid: {type: 'error', message: 'This Google sign-in request has expired. Please try again.'},
    google_unavailable: {type: 'error', message: 'Google sign-in is temporarily unavailable. Please try again later.'},
    maintenance: {type: 'warning', message: '系统维护中'}
};

export const AdminLoginPage = () => {
    const location = useLocation();
    const [loading, setLoading] = React.useState(false);
    const reason = new URLSearchParams(location.search).get('reason') || '';
    const alert = alerts[reason];

    const continueWithGoogle = () => {
        if (loading) {
            return;
        }
        setLoading(true);
        const returnTo = readAdminLoginReturnTo(location.search);
        const query = new URLSearchParams({returnTo});
        window.location.assign(`${deploymentPath('auth/google/login')}?${query.toString()}`);
    };

    return (
        <main className='login-screen admin-login-screen'>
            <Card className='login-panel'>
                <div className='login-panel__brand'>
                    <BrandMark size='large' />
                    <Typography.Title level={3}>Athena Admin</Typography.Title>
                    <Typography.Text type='secondary'>Administration Console</Typography.Text>
                </div>
                <div className='login-panel__alerts' aria-live='polite' aria-atomic='true'>
                    {alert && <Alert type={alert.type} title={alert.message} showIcon={true} />}
                </div>
                <div className='login-panel__actions' aria-busy={loading || undefined}>
                    <Button className='login-provider-button login-google-button' block={true} htmlType='button' loading={loading} disabled={loading} onClick={continueWithGoogle}>
                        {!loading && <img src={googleMark} alt='' aria-hidden='true' />}
                        <span>Continue with Google</span>
                    </Button>
                    <Typography.Paragraph className='login-panel__hint' type='secondary'>
                        Administrator access is verified by the Athena server. Member and wallet sign-in methods are not available in this console.
                    </Typography.Paragraph>
                </div>
            </Card>
        </main>
    );
};
