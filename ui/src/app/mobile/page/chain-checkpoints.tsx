import {Select} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import {AppPage, CardTitle, MetricRow, ResponsiveResourceList, StatusTag, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {TokenAPIChainIngestCheckpoint} from '../../shared/services/tokenapi-service';
import {boolTag, fmt, fmtNumber, rbacActions, rbacResources, useCanI} from './shared';
import {ChainBadge} from './token-shared';

export const ChainCheckpointsPage = () => {
    const data = useAsyncData(() => services.tokenapi.listChainIngestCheckpoints(), []);
    const canUpdate = useCanI(rbacResources.tokenapi, rbacActions.update);
    const canModify = canUpdate.data === true;
    const updateStatus = async (item: TokenAPIChainIngestCheckpoint, status: string) => {
        if (!canModify || item.chainID === undefined) {
            return;
        }
        await services.tokenapi.updateChainIngestCheckpoint(item.chainID, status);
        data.reload();
    };
    const columns: ColumnsType<TokenAPIChainIngestCheckpoint> = [
        {title: 'Chain', render: item => <ChainBadge chainID={item.chainID} />},
        {title: 'Enabled', render: item => boolTag(item.enabled)},
        {title: 'Cursor', render: item => fmtNumber(item.cursorBlockNumber)},
        {title: 'Status', render: item => <StatusTag value={item.status} positive={item.status === 'running'} />},
        {title: 'Created', dataIndex: 'createdAt'},
        {
            title: 'Actions',
            render: item => (
                <Select
                    disabled={!canModify || item.chainID === undefined}
                    value={item.status}
                    style={{width: 130}}
                    onChange={value => void updateStatus(item, value)}
                    options={['running', 'stopped'].map(value => ({value, label: value}))}
                />
            )
        }
    ];
    return (
        <AppPage title='Chain Checkpoints' loading={data.loading} error={data.error} onRefresh={data.reload}>
            <ResponsiveResourceList
                rowKey={item => item.chainID ?? Math.random()}
                items={data.data || []}
                columns={columns}
                loading={data.loading}
                card={item => (
                    <>
                        <CardTitle title={item.chainName || `Chain ${item.chainID}`} subtitle={`Cursor ${fmtNumber(item.cursorBlockNumber)}`} tags={<ChainBadge chainID={item.chainID} />} />
                        <MetricRow
                            items={[
                                {label: 'Status', value: <StatusTag value={item.status} positive={item.status === 'running'} />},
                                {label: 'Enabled', value: fmt(item.enabled)},
                                {label: 'Created', value: item.createdAt}
                            ]}
                        />
                        <Select
                            disabled={!canModify || item.chainID === undefined}
                            value={item.status}
                            style={{width: '100%'}}
                            onChange={value => void updateStatus(item, value)}
                            options={['running', 'stopped'].map(value => ({value, label: value}))}
                        />
                    </>
                )}
            />
        </AppPage>
    );
};
