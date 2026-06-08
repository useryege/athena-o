import {Tag} from 'antd';
import {AppPage, CardTitle, ResponsiveResourceList, Section, useAsyncData} from '../components';
import {Account, UserInfo} from '../../shared/models';
import {services} from '../../shared/services';
import {boolTag} from './shared';
import {visibleAccountsForUser} from './settings-shared';

export const SettingsPage = () => {
    const user = useAsyncData<UserInfo>(() => services.users.get() as any, []);
    const accounts = useAsyncData<Account[]>(() => services.accounts.list() as any, []);
    const visibleAccounts = visibleAccountsForUser(accounts.data || [], user.data);
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
                <ResponsiveResourceList
                    rowKey='name'
                    items={visibleAccounts}
                    columns={[
                        {title: 'Name', dataIndex: 'name'},
                        {title: 'Enabled', render: item => boolTag(item.enabled)},
                        {title: 'Capabilities', render: item => (item.capabilities || []).join(', ')}
                    ]}
                    card={item => (
                        <CardTitle title={item.name} subtitle={(item.capabilities || []).join(', ')} tags={item.enabled ? <Tag color='green'>Enabled</Tag> : <Tag>Disabled</Tag>} />
                    )}
                />
            </Section>
        </AppPage>
    );
};
