import {Card, Switch, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ResourceTable, Section, StatusTag, useAsyncData} from '../components';
import {Context} from '../shared/context';
import {Account, UserInfo} from '../shared/models';
import {services} from '../shared/services';
import {requestErrorMessage} from '../shared/services/requests';
import {visibleAccountsForUser} from './settings-shared';

const accountStatus = (account: Account) => (
    <StatusTag value={account.enabled ? 'Enabled' : 'Disabled'} positive={account.enabled} negative={!account.enabled} />
);

export const SettingsPage = () => {
    const ctx = React.useContext(Context);
    const user = useAsyncData<UserInfo>(() => services.users.get() as any, []);
    const accounts = useAsyncData<Account[]>(() => services.accounts.list() as any, []);
    const [accountItems, setAccountItems] = React.useState<Account[]>([]);
    const [updatingAccounts, setUpdatingAccounts] = React.useState<Record<string, boolean>>({});

    React.useEffect(() => {
        if (accounts.data) {
            setAccountItems(accounts.data);
        }
    }, [accounts.data]);

    const isAdmin = user.data?.username === 'admin';
    const visibleAccounts = visibleAccountsForUser(accountItems, user.data);

    const updateLoginAccess = async (account: Account, enabled: boolean) => {
        if (account.name === 'admin' || updatingAccounts[account.name]) {
            return;
        }
        setUpdatingAccounts(current => ({...current, [account.name]: true}));
        try {
            const updated = await services.accounts.update(account.name, enabled);
            setAccountItems(current => current.map(item => (item.name === account.name ? updated : item)));
            ctx.notifications.success('Login access updated', `${account.name} is now ${updated.enabled ? 'enabled' : 'disabled'}.`);
        } catch (err) {
            ctx.notifications.error('Could not update login access', requestErrorMessage(err, 'The account was not changed.'));
        } finally {
            setUpdatingAccounts(current => {
                const next = {...current};
                delete next[account.name];
                return next;
            });
        }
    };

    const requestLoginAccessChange = (account: Account, enabled: boolean) => {
        if (enabled) {
            void updateLoginAccess(account, true);
            return;
        }
        ctx.modal.confirm({
            title: `Disable login for ${account.name}?`,
            content: 'The account will be unable to use the web console, existing sessions, or API keys until it is enabled again.',
            onOk: () => updateLoginAccess(account, false)
        });
    };

    const loginAccessControl = (account: Account) => {
        if (!isAdmin) {
            return null;
        }
        if (account.name === 'admin') {
            return <Typography.Text type='secondary'>Always enabled</Typography.Text>;
        }
        return (
            <Switch
                aria-label={`Allow ${account.name} to log in`}
                checked={account.enabled}
                loading={Boolean(updatingAccounts[account.name])}
                disabled={Boolean(updatingAccounts[account.name])}
                checkedChildren='Allowed'
                unCheckedChildren='Blocked'
                onChange={enabled => requestLoginAccessChange(account, enabled)}
            />
        );
    };

    const columns: ColumnsType<Account> = [
        {title: 'Name', dataIndex: 'name'},
        {title: 'Status', render: accountStatus},
        {title: 'Capabilities', render: item => (item.capabilities || []).join(', ') || '-'}
    ];
    if (isAdmin) {
        columns.push({title: 'Login access', render: loginAccessControl});
    }

    const compactAccount = (account: Account) => (
        <Card className='account-compact-card' size='small' title={account.name} extra={isAdmin ? loginAccessControl(account) : undefined}>
            <div className='account-compact-card__details'>
                <div>
                    <Typography.Text type='secondary'>Status</Typography.Text>
                    <span>{accountStatus(account)}</span>
                </div>
                <div>
                    <Typography.Text type='secondary'>Capabilities</Typography.Text>
                    <span>{(account.capabilities || []).join(', ') || '-'}</span>
                </div>
            </div>
        </Card>
    );

    return (
        <AppPage
            title='Settings'
            loading={user.loading || accounts.loading}
            error={user.error || accounts.error}
            onRefresh={() => {
                user.reload();
                accounts.reload();
            }}>
            <Section title='Accounts'>
                <ResourceTable
                    rowKey='name'
                    label={isAdmin ? 'Accounts and login access' : 'Accounts'}
                    items={visibleAccounts}
                    columns={columns}
                    loading={user.loading || accounts.loading}
                    compactRender={compactAccount}
                    compactEmptyDescription='No accounts available'
                />
            </Section>
        </AppPage>
    );
};
