import {Button} from 'antd';
import * as React from 'react';
import {useNavigate} from 'react-router-dom';
import {AppPage, KeyValueGrid, Section, useAsyncData} from '../components';
import {Context, useAuthorization} from '../shared/context';
import {moduleAccessSummary} from '../shared/account-access';
import {VersionMessage} from '../shared/models';
import {services} from '../shared/services';
import {boolTag} from './shared';

export const UserInfoPage = (props: {onSessionEnded: () => void}) => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const navigate = useNavigate();
    const [loggingOut, setLoggingOut] = React.useState(false);
    const version = useAsyncData<VersionMessage & {version?: string}>(() => services.version.version() as any, []);
    const uiVersion = typeof SYSTEM_INFO === 'undefined' ? 'latest' : SYSTEM_INFO.version;

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
            loading={version.loading}
            error={version.error}
            onRefresh={() => {
                void authorization.refresh();
                version.reload();
            }}>
            <Section title='Current Session'>
                <KeyValueGrid
                    items={[
                        {label: 'Username', value: authorization.user.username},
                        {label: 'Logged In', value: boolTag(authorization.user.loggedIn)},
                        {label: 'Administrator', value: boolTag(authorization.isAdmin)},
                        {
                            label: 'Module Access',
                            value: moduleAccessSummary(authorization.user.access, authorization.isAdmin)
                        },
                        {label: 'Authorization Revision', value: authorization.revision},
                        {label: 'Issuer', value: authorization.user.iss || 'athena'},
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
