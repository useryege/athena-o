import {CopyOutlined, LinkOutlined, SearchOutlined} from '@ant-design/icons';
import {Alert, Button, Input, Skeleton, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ResourceTable, Section, TruncatedText, useAsyncData} from '../../components';
import {requestErrorMessage} from '../../shared/services/requests';
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

const missingNameLabel = (status: string) => {
    if (status === 'pending' || !status) {
        return 'Metadata pending';
    }
    return status === 'error' ? 'Metadata read failed' : 'Name unavailable';
};

const issuanceLabel = (item: SolanaProject) => {
    if (item.sourceStatus === 'pending' || !item.sourceStatus) {
        return 'Source pending';
    }
    if (item.sourceStatus === 'error') {
        return 'Source read failed';
    }
    if (item.sourceStatus === 'identified') {
        switch (item.issuanceSource) {
            case 'pump_fun':
                return 'Pump.fun';
            case 'raydium_launchlab':
                return 'Raydium LaunchLab';
            case 'direct_token':
                return 'Direct Token initialization';
        }
    }
    return 'Unrecognized platform';
};

const IssuanceSource = ({item}: {item: SolanaProject}) => (
    <span className={item.sourceStatus === 'error' ? 'solana-source solana-source--error' : 'solana-source'}>{issuanceLabel(item)}</span>
);

const metadataSourceLabel = (source: string) => {
    if (source === 'token2022_on_mint') {
        return 'Token-2022 on-mint metadata';
    }
    return source === 'metaplex' ? 'Metaplex metadata' : 'Unavailable';
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
        <div className='foundation-address'>
            <TruncatedText value={props.value} />
            <Button aria-label={props.copyLabel} type='text' size='small' icon={<CopyOutlined />} onClick={copy} />
            <a href={solscanURL(props.kind, props.value)} target='_blank' rel='noreferrer' aria-label={`Open ${props.kind} in Solscan`}>
                <LinkOutlined />
            </a>
        </div>
    );
};

const Facts = (props: {items: Array<{label: string; value: React.ReactNode}>; className?: string}) => (
    <dl className={`foundation-facts ${props.className || ''}`}>
        {props.items.map(item => (
            <div key={item.label}>
                <dt>{item.label}</dt>
                <dd>{item.value}</dd>
            </div>
        ))}
    </dl>
);

const Evidence = ({item}: {item: SolanaProject}) => (
    <Facts
        items={[
            {label: 'Mint authority at initialization', value: <TruncatedText value={item.mintAuthority} copyable={true} />},
            {label: 'Freeze authority at initialization', value: <TruncatedText value={item.freezeAuthority} copyable={true} />},
            {label: 'Fee payer', value: <TruncatedText value={item.feePayer} copyable={true} />},
            {label: 'Slot', value: item.slot || 'Unknown'},
            {label: 'Token standard', value: programLabel(item.tokenProgram)},
            {label: 'Token program', value: <TruncatedText value={item.tokenProgram} copyable={true} />},
            {label: 'Issuance program', value: <TruncatedText value={item.issuanceProgram} copyable={true} />},
            {label: 'Metadata source', value: metadataSourceLabel(item.metadataSource)},
            {label: 'Metadata account', value: <TruncatedText value={item.metadataAccount} copyable={true} />},
            {label: 'Metadata observed slot', value: item.metadataObservedSlot && item.metadataObservedSlot !== '0' ? item.metadataObservedSlot : 'Unknown'},
            {label: 'Metadata observed at', value: formatTime(item.metadataUpdatedAt)}
        ]}
    />
);

const Status = (props: {status?: SolanaDiscoveryStatus; loading: boolean; error?: Error; retry: () => void}) => {
    const value = props.status;
    return (
        <Section
            title='Discovery status'
            extra={value && <Tag color={value.lastError ? 'warning' : value.status === 'current' ? 'success' : 'processing'}>{value.status || 'Unknown'}</Tag>}>
            {props.error && (
                <Alert
                    type='error'
                    showIcon={true}
                    title='Discovery status unavailable'
                    description={requestErrorMessage(props.error)}
                    action={
                        <Button aria-label='Retry discovery status' onClick={props.retry}>
                            Retry
                        </Button>
                    }
                />
            )}
            {props.loading && !value && <Skeleton active={true} paragraph={{rows: 2}} />}
            {value && (
                <>
                    {props.error && <p className='foundation-note'>Last known discovery status. This information may be stale.</p>}
                    {value.lastError && <Alert type='warning' showIcon={true} title='Scanner needs attention' description={value.lastError} />}
                    <Facts
                        className='solana-discovery-facts'
                        items={[
                            {label: 'Start slot', value: value.startSlot || 'Unknown'},
                            {label: 'Last processed slot', value: value.lastProcessedSlot || 'Unknown'},
                            {label: 'Latest finalized slot', value: value.latestFinalizedSlot || 'Unknown'},
                            {label: 'Last successful', value: formatTime(value.lastSuccessAt)},
                            {label: 'Discovered candidates', value: value.totalProjects || 'Unknown'}
                        ]}
                    />
                </>
            )}
        </Section>
    );
};

const TokenIdentity = ({item}: {item: SolanaProject}) => (
    <div className='solana-token-identity'>
        <Typography.Text strong={Boolean(item.name)} type={item.name ? undefined : 'secondary'}>
            {item.name || missingNameLabel(item.metadataStatus)}
        </Typography.Text>
        <Typography.Text type='secondary'>{item.symbol || 'Symbol unavailable'}</Typography.Text>
        <AddressActions value={item.mint} kind='token' copyLabel={`Copy mint ${item.mint}`} />
    </div>
);

const CandidateCompact = ({item}: {item: SolanaProject}) => (
    <article className='solana-candidate'>
        <TokenIdentity item={item} />
        <Facts
            items={[
                {label: 'Issuance source', value: <IssuanceSource item={item} />},
                {label: 'Decimals', value: item.decimals},
                {label: 'Issued', value: formatTime(item.blockTime)},
                {label: 'Discovered', value: formatTime(item.discoveredAt)}
            ]}
        />
        <div className='solana-transaction'>
            <span>Transaction</span>
            <AddressActions value={item.signature} kind='tx' copyLabel={`Copy transaction ${item.signature}`} />
        </div>
        <details className='foundation-details'>
            <summary>Issuance evidence</summary>
            <Evidence item={item} />
        </details>
    </article>
);

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
        {title: 'Token / Mint', width: '32%', render: item => <TokenIdentity item={item} />},
        {title: 'Issuance source', width: '18%', render: item => <IssuanceSource item={item} />},
        {
            title: 'Issued / Discovered',
            width: '20%',
            render: item => (
                <Facts
                    items={[
                        {label: 'Issued', value: formatTime(item.blockTime)},
                        {label: 'Discovered', value: formatTime(item.discoveredAt)}
                    ]}
                />
            )
        },
        {title: 'Decimals', width: '10%', dataIndex: 'decimals'},
        {title: 'Transaction', width: '20%', render: item => <AddressActions value={item.signature} kind='tx' copyLabel={`Copy transaction ${item.signature}`} />}
    ];
    return (
        <div className='foundation-page'>
            <AppPage
                title='Solana'
                subtitle='Newly initialized token candidates from the original Token Program and Token-2022. Assets are not classified as projects, NFTs, LPs, or investments.'
                loading={projects.loading || discoveryStatus.loading}
                onRefresh={reload}>
                <Status status={discoveryStatus.data} loading={discoveryStatus.loading} error={discoveryStatus.error} retry={discoveryStatus.reload} />
                <section className='section-panel solana-candidates' aria-label='Issuance candidates'>
                    <div className='foundation-filters'>
                        <label htmlFor='solana-query'>Name, symbol or Mint</label>
                        <Input
                            id='solana-query'
                            aria-label='Name, symbol or Mint'
                            maxLength={128}
                            allowClear={true}
                            placeholder='Name, symbol or Mint'
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
                                <Button type='text' size='small' onClick={search} aria-label='Search candidates'>
                                    Search
                                </Button>
                            }
                        />
                    </div>
                    {projects.error && (
                        <Alert
                            type='error'
                            showIcon={true}
                            title='Candidates unavailable'
                            description={requestErrorMessage(projects.error)}
                            action={
                                <Button aria-label='Retry candidates' onClick={projects.reload}>
                                    Retry
                                </Button>
                            }
                        />
                    )}
                    {projects.error && projects.data && <p className='foundation-note'>Last known candidates. This list may be stale.</p>}
                    {projects.loading && !projects.data && <Skeleton active={true} paragraph={{rows: 4}} />}
                    {projects.data && (
                        <ResourceTable<SolanaProject>
                            label='Solana token candidates'
                            rowKey='mint'
                            items={projects.data.items}
                            columns={columns}
                            loading={projects.loading}
                            total={projects.data.totalSize}
                            page={page}
                            pageSize={pageSize}
                            pageSizeOptions={[pageSize]}
                            compactRender={item => <CandidateCompact item={item} />}
                            compactEmptyDescription={query ? 'No matching candidates' : 'No discovered candidates yet'}
                            onPageChange={nextPage => setPage(nextPage)}
                            expandable={{
                                expandedRowRender: item => <Evidence item={item} />
                            }}
                        />
                    )}
                    <p className='foundation-note'>Names and symbols are issuer-provided snapshots, not project verification.</p>
                </section>
            </AppPage>
        </div>
    );
};
