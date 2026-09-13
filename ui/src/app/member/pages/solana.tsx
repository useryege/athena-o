import {CopyOutlined, LinkOutlined, SearchOutlined} from '@ant-design/icons';
import {Alert, Button, Empty, Input, Space, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, KeyValueGrid, ResourceTable, Section, TruncatedText, useAsyncData} from '../../components';
import {formatBeijingUnixSeconds} from '../../shared/format';
import {memberServices as services} from '../services';
import type {SolanaDiscoveryStatus, SolanaProject} from '../../shared/services/solana-service';

const pageSize = 25;
const solscanURL = (kind: 'token' | 'tx', value: string) => `https://solscan.io/${kind}/${encodeURIComponent(value)}`;

const safeUnixSeconds = (value: string) => {
    if (!/^-?\d+$/.test(value)) {
        return undefined;
    }
    const parsed = Number(value);
    return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : undefined;
};

const formatTime = (value: string) => formatBeijingUnixSeconds(safeUnixSeconds(value)) || 'Unknown';

const programLabel = (program: string) => {
    if (program === 'TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA' || program === 'Token') {
        return 'Token Program';
    }
    if (program === 'TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb' || program === 'Token-2022') {
        return 'Token-2022';
    }
    return program || 'Unknown';
};

const AddressActions = (props: {value: string; kind: 'token' | 'tx'; copyLabel: string}) => {
    const copy = async () => {
        if (props.value) {
            await navigator.clipboard?.writeText(props.value);
        }
    };
    if (!props.value) {
        return <>-</>;
    }
    return (
        <Space size={4} wrap={false}>
            <TruncatedText value={props.value} singleLine={true} />
            <Button aria-label={props.copyLabel} type='text' size='small' icon={<CopyOutlined />} onClick={copy} />
            <a href={solscanURL(props.kind, props.value)} target='_blank' rel='noreferrer' aria-label={`Open ${props.kind} in Solscan`}>
                <LinkOutlined />
            </a>
        </Space>
    );
};

const Status = (props: {status?: SolanaDiscoveryStatus}) => {
    const value = props.status;
    if (!value) {
        return null;
    }
    return (
        <Section title='Discovery status'>
            <Space orientation='vertical' size={12} style={{width: '100%'}}>
                <Space wrap={true}>
                    <Typography.Text strong={true}>Scanner</Typography.Text>
                    <Tag color={value.lastError ? 'red' : value.status === 'current' ? 'green' : 'blue'}>{value.status || 'Unknown'}</Tag>
                </Space>
                {value.lastError && <Alert type='warning' showIcon={true} message='Scanner needs attention' description={value.lastError} />}
                <KeyValueGrid
                    items={[
                        {label: 'Start slot', value: value.startSlot || '-'},
                        {label: 'Last processed slot', value: value.lastProcessedSlot || '-'},
                        {label: 'Latest finalized slot', value: value.latestFinalizedSlot || '-'},
                        {label: 'Last successful', value: formatTime(value.lastSuccessAt)},
                        {label: 'Discovered candidates', value: value.totalProjects || '0'}
                    ]}
                />
            </Space>
        </Section>
    );
};

export const SolanaPage = () => {
    const [page, setPage] = React.useState(1);
    const [pendingQuery, setPendingQuery] = React.useState('');
    const [query, setQuery] = React.useState('');
    const projects = useAsyncData(() => services.solana.listProjects({page, pageSize, query}), [page, query]);
    const discoveryStatus = useAsyncData(() => services.solana.getDiscoveryStatus(), []);
    const reload = React.useCallback(() => {
        projects.reload();
        discoveryStatus.reload();
    }, [discoveryStatus.reload, projects.reload]);
    const search = React.useCallback(() => {
        setPage(1);
        setQuery(pendingQuery.trim());
    }, [pendingQuery]);
    const columns: ColumnsType<SolanaProject> = [
        {title: 'Mint', width: 280, render: item => <AddressActions value={item.mint} kind='token' copyLabel={`Copy mint ${item.mint}`} />},
        {title: 'Program', width: 160, render: item => programLabel(item.tokenProgram)},
        {title: 'Issued', width: 180, render: item => formatTime(item.blockTime)},
        {title: 'Discovered', width: 180, render: item => formatTime(item.discoveredAt)},
        {title: 'Decimals', width: 100, dataIndex: 'decimals'},
        {title: 'Transaction', width: 170, render: item => <AddressActions value={item.signature} kind='tx' copyLabel={`Copy transaction ${item.signature}`} />}
    ];
    return (
        <AppPage
            title='Solana'
            subtitle='Newly initialized token candidates from the original Token Program and Token-2022. Assets are not classified as projects, NFTs, LPs, or investments.'
            loading={projects.loading || discoveryStatus.loading}
            error={projects.error || discoveryStatus.error}
            onRefresh={reload}
            filters={
                <Input
                    aria-label='Mint address'
                    maxLength={128}
                    allowClear={true}
                    placeholder='Mint address'
                    prefix={<SearchOutlined />}
                    value={pendingQuery}
                    onChange={event => setPendingQuery(event.target.value)}
                    onPressEnter={search}
                    onClear={() => {
                        setPendingQuery('');
                        setQuery('');
                        setPage(1);
                    }}
                    suffix={
                        <Button type='text' size='small' onClick={search} aria-label='Search mint'>
                            Search
                        </Button>
                    }
                />
            }>
            <Status status={discoveryStatus.data} />
            {!projects.loading && !projects.error && (projects.data?.items.length || 0) === 0 ? (
                <Empty description={query ? 'No matching Mint' : 'No discovered candidates yet'} />
            ) : (
                <ResourceTable<SolanaProject>
                    label='Solana token candidates'
                    rowKey='mint'
                    items={projects.data?.items || []}
                    columns={columns}
                    loading={projects.loading}
                    total={projects.data?.totalSize || 0}
                    page={page}
                    pageSize={pageSize}
                    pageSizeOptions={[pageSize]}
                    scrollX={1050}
                    onPageChange={nextPage => setPage(nextPage)}
                    expandable={{
                        expandedRowRender: item => (
                            <KeyValueGrid
                                columns={3}
                                items={[
                                    {label: 'Mint authority at initialization', value: <TruncatedText value={item.mintAuthority} copyable={true} />},
                                    {label: 'Freeze authority at initialization', value: <TruncatedText value={item.freezeAuthority} copyable={true} />},
                                    {label: 'Fee payer', value: <TruncatedText value={item.feePayer} copyable={true} />},
                                    {label: 'Slot', value: item.slot || '-'},
                                    {label: 'Token program', value: <TruncatedText value={item.tokenProgram} copyable={true} />}
                                ]}
                            />
                        )
                    }}
                />
            )}
        </AppPage>
    );
};
