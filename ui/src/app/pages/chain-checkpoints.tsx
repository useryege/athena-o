import {Select} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import {AppPage, ResourceTable, StatusTag, useAsyncData} from '../components';
import {services} from '../shared/services';
import {TokenAPIChainIngestCheckpoint} from '../shared/services/tokenapi-service';
import {boolTag, fmtNumber} from './shared';
import {ChainBadge} from './token-shared';

export const ChainCheckpointsPage = () => {
    const data = useAsyncData(() => services.tokenapi.listChainIngestCheckpoints(), []);
    const updateStatus = async (item: TokenAPIChainIngestCheckpoint, status: string) => {
        if (item.chainID === undefined) {
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
                    aria-label={`Set status for chain ${item.chainID ?? ''}`}
                    disabled={item.chainID === undefined}
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
            <ResourceTable rowKey={item => item.chainID ?? Math.random()} items={data.data || []} columns={columns} loading={data.loading} />
        </AppPage>
    );
};
