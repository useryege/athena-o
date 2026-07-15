import {Empty} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import {AppPage, ResourceTable, Section, StatusTag, TruncatedText, useAsyncData} from '../components';
import {services} from '../shared/services';
import {TokenNodeStatus} from '../shared/services/token-service';
import {fmtNumber} from './shared';
import {ChainBadge, chainLabel} from './token-shared';

const displayNumber = (value?: number) => (value === undefined || value === 0 ? '-' : fmtNumber(value));
const displayLag = (value?: number) => (value === undefined ? '-' : fmtNumber(value));
const displayLatency = (value?: number) => (value === undefined ? '-' : `${fmtNumber(value)} ms`);
const displaySyncing = (value?: boolean) => (value === undefined ? '-' : value ? 'Yes' : 'No');

export const NodeStatusesPage = () => {
    const data = useAsyncData(() => services.tokenapi.listNodeStatuses(), []);
    const statuses = data.data || [];
    const chainIDs = Array.from(new Set(statuses.map(item => item.chainID).filter((value): value is number => value !== undefined)));
    const columns: ColumnsType<TokenNodeStatus> = [
        {title: 'Endpoint', render: item => <TruncatedText value={item.endpoint} copyable={true} />},
        {title: 'Status', render: item => <StatusTag value={item.available ? 'Available' : 'Unavailable'} positive={item.available} negative={!item.available} />},
        {title: 'Latency', render: item => displayLatency(item.latencyMS)},
        {title: 'Reported Chain ID', render: item => displayNumber(item.reportedChainID)},
        {title: 'Latest Block', render: item => displayNumber(item.latestBlockNumber)},
        {title: 'Reference Block', render: item => displayNumber(item.referenceBlockNumber)},
        {title: 'Block Lag', render: item => displayLag(item.blockLag)},
        {title: 'Latest Block Time', dataIndex: 'latestBlockTime'},
        {
            title: 'Syncing',
            render: item => <StatusTag value={displaySyncing(item.syncing)} positive={item.syncing === false} negative={item.syncing === true} />
        },
        {title: 'Checked', dataIndex: 'checkedAt'},
        {title: 'Error', render: item => <TruncatedText value={item.error} copyable={Boolean(item.error)} />}
    ];

    return (
        <AppPage
            title='Node Status'
            subtitle='Live health checks using the same chain, sync, freshness, and block-lag rules as worker node selection.'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}>
            {chainIDs.length === 0 && !data.loading ? (
                <Empty description='No node endpoints configured' />
            ) : (
                chainIDs.map(chainID => {
                    const items = statuses.filter(item => item.chainID === chainID);
                    return (
                        <Section key={chainID} title={`${items[0]?.chainName || chainLabel(chainID)} Nodes`} extra={<ChainBadge chainID={chainID} />}>
                            <ResourceTable rowKey={item => `${chainID}:${item.endpoint}`} items={items} columns={columns} loading={data.loading} scrollX={1500} />
                        </Section>
                    );
                })
            )}
        </AppPage>
    );
};
