import {Alert, Button, Card, Typography} from 'antd';
import * as React from 'react';
import {useLocation} from 'react-router-dom';
import googleMark from '../../assets/images/google-g.svg';
import {BrandMark} from '../components';
import {readLoginReturnTo} from '../shared/login-navigation';
import requests, {ACCOUNT_MAINTENANCE_MESSAGE} from '../shared/services/requests';

const loginReasonAlerts: Record<string, {type: 'error' | 'warning'; message: string}> = {
    google_cancelled: {type: 'warning', message: 'Google sign-in was cancelled. Try again when you are ready.'},
    google_not_allowed: {type: 'error', message: 'This Google account is not approved for Athena.'},
    google_state_invalid: {type: 'error', message: 'This sign-in request has expired or is no longer valid. Please try again.'},
    google_unavailable: {type: 'error', message: 'Google sign-in is temporarily unavailable. Please try again later.'},
    maintenance: {type: 'warning', message: ACCOUNT_MAINTENANCE_MESSAGE}
};

export const LoginPage = () => {
    const location = useLocation();
    const [loading, setLoading] = React.useState(false);
    const reason = new URLSearchParams(location.search).get('reason') || '';
    const alert = loginReasonAlerts[reason];

    const continueWithGoogle = () => {
        if (loading) {
            return;
        }
        setLoading(true);
        const query = new URLSearchParams({returnTo: readLoginReturnTo(location.search)});
        window.location.assign(`${requests.toAbsURL('/auth/google/login')}?${query.toString()}`);
    };

    return (
        <main className='login-screen'>
            <Card className='login-panel'>
                <div className='login-panel__brand'>
                    <BrandMark size='large' />
                    <Typography.Title level={3}>Athena</Typography.Title>
                    <Typography.Text type='secondary'>Operations Console</Typography.Text>
                </div>
                <div className='login-panel__alerts' aria-live='polite' aria-atomic='true'>
                    {alert && <Alert type={alert.type} title={alert.message} showIcon={true} />}
                </div>
                <div className='login-panel__actions' aria-busy={loading || undefined}>
                    <Button className='login-google-button' block={true} htmlType='button' loading={loading} disabled={loading} onClick={continueWithGoogle}>
                        {!loading && <img src={googleMark} alt='' aria-hidden='true' />}
                        <span>Continue with Google</span>
                    </Button>
                    <Typography.Paragraph className='login-panel__hint' type='secondary'>
                        Use an approved Google account.
                    </Typography.Paragraph>
                </div>
            </Card>
        </main>
    );
};
