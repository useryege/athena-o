import {Empty} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import {AppPage, CardTitle, MetricRow, ResponsiveResourceList, Section, StatusTag, TruncatedText, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {TokenAPINodeStatus} from '../../shared/services/tokenapi-service';
import {fmtNumber} from './shared';
import {ChainBadge, chainLabel} from './token-shared';

const displayNumber = (value?: number) => (value === undefined || value === 0 ? '-' : fmtNumber(value));
const displayLatency = (value?: number) => (value === undefined ? '-' : `${fmtNumber(value)} ms`);

export const NodeStatusesPage = () => {
    const data = useAsyncData(() => services.tokenapi.listNodeStatuses(), []);
    const statuses = data.data || [];
    const chainIDs = Array.from(new Set(statuses.map(item => item.chainID).filter((value): value is number => value !== undefined)));
    const columns: ColumnsType<TokenAPINodeStatus> = [
        {title: 'Endpoint', render: item => <TruncatedText value={item.endpoint} copyable={true} />},
        {title: 'Status', render: item => <StatusTag value={item.available ? 'Available' : 'Unavailable'} positive={item.available} negative={!item.available} />},
        {title: 'Latency', render: item => displayLatency(item.latencyMS)},
        {title: 'Reported Chain ID', render: item => displayNumber(item.reportedChainID)},
        {title: 'Latest Block', render: item => displayNumber(item.latestBlockNumber)},
        {title: 'Checked', dataIndex: 'checkedAt'},
        {title: 'Error', render: item => <TruncatedText value={item.error} copyable={Boolean(item.error)} />}
    ];

    return (
        <AppPage
            title='Node Status'
            subtitle='Live checks from the Token API service. Results do not represent the node currently selected by a worker.'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
        >
            {chainIDs.length === 0 && !data.loading ? (
                <Empty description='No node endpoints configured' />
            ) : (
                chainIDs.map(chainID => {
                    const items = statuses.filter(item => item.chainID === chainID);
                    return (
                        <Section key={chainID} title={`${items[0]?.chainName || chainLabel(chainID)} Nodes`} extra={<ChainBadge chainID={chainID} />}>
                            <ResponsiveResourceList
                                rowKey={item => `${chainID}:${item.endpoint}`}
                                items={items}
                                columns={columns}
                                loading={data.loading}
                                card={item => (
                                    <>
                                        <CardTitle
                                            title={<TruncatedText value={item.endpoint} copyable={true} />}
                                            tags={
                                                <StatusTag
                                                    value={item.available ? 'Available' : 'Unavailable'}
                                                    positive={item.available}
                                                    negative={!item.available}
                                                />
                                            }
                                        />
                                        <MetricRow
                                            items={[
                                                {label: 'Latency', value: displayLatency(item.latencyMS), tone: item.available ? 'good' : 'bad'},
                                                {label: 'Chain ID', value: displayNumber(item.reportedChainID)},
                                                {label: 'Latest Block', value: displayNumber(item.latestBlockNumber)},
                                                {label: 'Checked', value: item.checkedAt || '-'}
                                            ]}
                                        />
                                        {item.error && <TruncatedText value={item.error} copyable={true} />}
                                    </>
                                )}
                            />
                        </Section>
                    );
                })
            )}
        </AppPage>
    );
};
