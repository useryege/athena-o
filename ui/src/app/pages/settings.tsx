import {Card, Switch, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ChoiceGroup, ResourceTable, Section, StatusTag, useAsyncData} from '../components';
import {Context, useAuthorization} from '../shared/context';
import {Account, AccountAccess, AccountDataAccess} from '../shared/models';
import {services} from '../shared/services';
import {requestErrorDetails, requestErrorMessage} from '../shared/services/requests';

const dataAccessOptions = [
    {value: AccountDataAccess.None, label: 'No access'},
    {value: AccountDataAccess.Read, label: 'Read only'},
    {value: AccountDataAccess.ReadWrite, label: 'Read & write'}
];

const dataAccessLabel = (value: AccountDataAccess) => dataAccessOptions.find(option => option.value === value)?.label || 'No access';

const accountStatus = (account: Account) => (
    <StatusTag value={account.access.loginEnabled ? 'Enabled' : 'Disabled'} positive={account.access.loginEnabled} negative={!account.access.loginEnabled} />
);

export const SettingsPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const accounts = useAsyncData<Account[]>(
        () =>
            (authorization.isAdmin ? services.accounts.list() : services.accounts.get(authorization.user.username).then(account => [account])) as Promise<Account[]> & {
                abort?: () => void;
            },
        [authorization.isAdmin, authorization.user.username]
    );
    const [accountItems, setAccountItems] = React.useState<Account[]>([]);
    const [updatingAccounts, setUpdatingAccounts] = React.useState<Record<string, boolean>>({});

    React.useEffect(() => {
        if (accounts.data) {
            setAccountItems(accounts.data);
        }
    }, [accounts.data]);

    const updateAccountAccess = async (account: Account, nextAccess: AccountAccess) => {
        if (account.administrator || updatingAccounts[account.name]) {
            return;
        }
        setUpdatingAccounts(current => ({...current, [account.name]: true}));
        try {
            const updated = await services.accounts.updateAccess(account.name, nextAccess);
            setAccountItems(current => current.map(item => (item.name === account.name ? updated : item)));
            ctx.notifications.success(
                'Account access updated',
                `${account.name}: login ${updated.access.loginEnabled ? 'allowed' : 'blocked'}, data ${dataAccessLabel(updated.access.dataAccess).toLowerCase()}.`
            );
        } catch (err) {
            if (requestErrorDetails(err).status === 409) {
                accounts.reload();
                ctx.notifications.warning('Account access changed', 'A newer setting was saved elsewhere. The account list has been reloaded.');
            } else {
                ctx.notifications.error('Could not update account access', requestErrorMessage(err, 'The account was not changed.'));
            }
        } finally {
            setUpdatingAccounts(current => {
                const next = {...current};
                delete next[account.name];
                return next;
            });
        }
    };

    const requestAccessChange = (account: Account, nextAccess: AccountAccess) => {
        const loginDisabled = account.access.loginEnabled && !nextAccess.loginEnabled;
        const dataDowngraded = nextAccess.dataAccess < account.access.dataAccess;
        const writeGranted = account.access.dataAccess !== AccountDataAccess.ReadWrite && nextAccess.dataAccess === AccountDataAccess.ReadWrite;
        let confirmation: {title: string; content: string} | undefined;
        if (loginDisabled) {
            confirmation = {
                title: `Disable login for ${account.name}?`,
                content: 'The account will be unable to use the web console, existing sessions, or API keys until login is enabled again.'
            };
        } else if (dataDowngraded) {
            confirmation = {
                title: `Reduce data access for ${account.name}?`,
                content: `Access will change from ${dataAccessLabel(account.access.dataAccess)} to ${dataAccessLabel(nextAccess.dataAccess)} on the account's next request.`
            };
        } else if (writeGranted) {
            confirmation = {
                title: `Grant read and write access to ${account.name}?`,
                content: 'This permits data mutations and sensitive operations, including wallet secret access.'
            };
        }
        if (!confirmation) {
            void updateAccountAccess(account, nextAccess);
            return;
        }
        ctx.modal.confirm({
            ...confirmation,
            onOk: () => updateAccountAccess(account, nextAccess)
        });
    };

    const loginAccess = (account: Account) => {
        if (!authorization.isAdmin) {
            return accountStatus(account);
        }
        if (account.administrator) {
            return <Typography.Text type='secondary'>Always enabled</Typography.Text>;
        }
        const updating = Boolean(updatingAccounts[account.name]);
        return (
            <Switch
                aria-label={`Allow ${account.name} to log in`}
                checked={account.access.loginEnabled}
                loading={updating}
                disabled={updating}
                checkedChildren='Allowed'
                unCheckedChildren='Blocked'
                onChange={loginEnabled => requestAccessChange(account, {...account.access, loginEnabled})}
            />
        );
    };

    const dataAccess = (account: Account) => {
        if (!authorization.isAdmin || account.administrator) {
            return <Typography.Text type='secondary'>{account.administrator ? 'Full access' : dataAccessLabel(account.access.dataAccess)}</Typography.Text>;
        }
        return (
            <ChoiceGroup<AccountDataAccess>
                className='account-data-access-choice'
                size='small'
                ariaLabel={`Data access for ${account.name}`}
                value={account.access.dataAccess}
                options={dataAccessOptions}
                disabled={Boolean(updatingAccounts[account.name])}
                onChange={nextDataAccess => requestAccessChange(account, {...account.access, dataAccess: nextDataAccess})}
            />
        );
    };

    const columns: ColumnsType<Account> = [
        {title: 'Name', dataIndex: 'name'},
        {title: 'Login access', render: loginAccess},
        {title: 'Data access', render: dataAccess},
        {title: 'Capabilities', render: item => (item.capabilities || []).join(', ') || '-'}
    ];

    const compactAccount = (account: Account) => (
        <Card className={`account-compact-card${updatingAccounts[account.name] ? ' account-compact-card--updating' : ''}`} size='small' title={account.name}>
            <div className='account-compact-card__details'>
                <div>
                    <Typography.Text type='secondary'>Login access</Typography.Text>
                    <span>{loginAccess(account)}</span>
                </div>
                <div>
                    <Typography.Text type='secondary'>Data access</Typography.Text>
                    <span>{dataAccess(account)}</span>
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
            loading={accounts.loading}
            error={accounts.error}
            onRefresh={() => {
                void authorization.refresh();
                accounts.reload();
            }}>
            <Section title='Accounts'>
                <ResourceTable
                    rowKey='name'
                    label={authorization.isAdmin ? 'Account login and data access' : 'Your account access'}
                    items={accountItems}
                    columns={columns}
                    loading={accounts.loading}
                    rowClassName={account => (updatingAccounts[account.name] ? 'account-access-row--updating' : '')}
                    compactRender={compactAccount}
                    compactEmptyDescription='No account available'
                />
            </Section>
        </AppPage>
    );
};
