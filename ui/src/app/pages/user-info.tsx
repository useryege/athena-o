import {Button} from 'antd';
import * as React from 'react';
import {useNavigate} from 'react-router-dom';
import {AppPage, KeyValueGrid, Section, useAsyncData} from '../components';
import {Context} from '../shared/context';
import {UserInfo, VersionMessage} from '../shared/models';
import {services} from '../shared/services';
import {boolTag} from './shared';

export const UserInfoPage = (props: {onSessionEnded: () => void}) => {
    const ctx = React.useContext(Context);
    const navigate = useNavigate();
    const [loggingOut, setLoggingOut] = React.useState(false);
    const user = useAsyncData<UserInfo>(() => services.users.get() as any, []);
    const version = useAsyncData<VersionMessage & {version?: string}>(() => services.version.version() as any, []);
    const uiVersion = typeof SYSTEM_INFO === 'undefined' ? 'latest' : SYSTEM_INFO.version;

    React.useEffect(() => {
        if (user.data?.loggedIn === false) {
            props.onSessionEnded();
            navigate('/login', {replace: true});
        }
    }, [navigate, props.onSessionEnded, user.data?.loggedIn]);

    const logout = async () => {
        setLoggingOut(true);
        ctx.notifications.info('Logging out');
        try {
            await services.users.logout();
            props.onSessionEnded();
            navigate('/login', {replace: true});
        } catch (err: any) {
            setLoggingOut(false);
            ctx.notifications.error('Logout failed', err?.message || 'Could not log out');
        }
    };
    return (
        <AppPage
            title='User Info'
            subtitle='Session, version, and account context'
            loading={user.loading || version.loading}
            error={user.error || version.error}
            onRefresh={() => {
                user.reload();
                version.reload();
            }}>
            <Section title='Current Session'>
                <KeyValueGrid
                    items={[
                        {label: 'Username', value: user.data?.username},
                        {label: 'Logged In', value: boolTag(user.data?.loggedIn)},
                        {label: 'Issuer', value: user.data?.iss || 'athena'},
                        {label: 'UI Version', value: uiVersion || '-'},
                        {label: 'Version', value: version.data?.Version || version.data?.version || '-'}
                    ]}
                />
            </Section>
            <Section title='Session'>
                <Button danger={true} loading={loggingOut} onClick={logout}>
                    Log out
                </Button>
            </Section>
        </AppPage>
    );
};
