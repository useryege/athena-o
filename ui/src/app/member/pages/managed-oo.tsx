import {LinkOutlined, SearchOutlined} from '@ant-design/icons';
import {Button, Drawer, InputNumber, Space, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {Link} from 'react-router-dom';
import {AppPage, KeyValueGrid, ResourceTable, Section, TruncatedText, useAsyncData} from '../../components';
import {Context, useAuthorization} from '../../shared/context';
import {AccountDataModule} from '../../shared/access-modules';
import {formatBeijingDateTime, formatBeijingUnixSeconds, formatBlockNumber} from '../../shared/format';
import {memberServices as services} from '../services';
import {ListManagedOOItemsResult, ManagedOODisputeItem, ManagedOOProposalItem} from '../../shared/services/managed-oo-service';
import {fmt, usePagedParams} from '../../shared/pages/shared';

type ManagedOOItem = ManagedOOProposalItem | ManagedOODisputeItem;
type ManagedOOKind = 'proposal' | 'dispute';

const explorerURL = (txHash: string) => `https://polygonscan.com/tx/${encodeURIComponent(txHash)}`;

const formatTimestamp = (value: number) => formatBeijingUnixSeconds(value) || 'Unknown';

const externalLink = (href: string, label: React.ReactNode) =>
    href ? (
        <a href={href} target='_blank' rel='noreferrer' onClick={event => event.stopPropagation()}>
            {label} <LinkOutlined />
        </a>
    ) : (
        '-'
    );

const ManagedOODetailDrawer = (props: {item?: ManagedOOItem; kind: ManagedOOKind; onClose: () => void}) => {
    const item = props.item;
    const roleItems = item
        ? [
              {label: 'Requester', value: <TruncatedText value={item.requester} copyable={true} />},
              {label: 'Proposer', value: <TruncatedText value={item.proposer} copyable={true} />},
              ...('disputer' in item ? [{label: 'Disputer', value: <TruncatedText value={item.disputer} copyable={true} />}] : [])
          ]
        : [];
    return (
        <Drawer
            className='managed-oo-drawer'
            footer={<Button onClick={props.onClose}>Close evidence</Button>}
            title={props.kind === 'proposal' ? 'Proposal details' : 'Dispute details'}
            open={Boolean(item)}
            onClose={props.onClose}
            placement='right'
            size={720}>
            {item && (
                <Space direction='vertical' size='large' style={{width: '100%'}}>
                    <Typography.Title level={5}>{item.question || item.marketId || 'Managed OO event'}</Typography.Title>
                    <Space wrap={true}>
                        {externalLink(explorerURL(item.txHash), 'Polygonscan')}
                        {externalLink(item.polymarketUrl, 'Polymarket')}
                    </Space>
                    <KeyValueGrid
                        columns={2}
                        items={[
                            {label: 'Block', value: formatBlockNumber(item.blockNumber)},
                            {label: 'Log index', value: fmt(item.logIndex)},
                            {label: 'Transaction', value: <TruncatedText value={item.txHash} copyable={true} />},
                            {label: 'Transaction index', value: fmt(item.txIndex)},
                            {label: 'Block hash', value: <TruncatedText value={item.blockHash} copyable={true} />},
                            {label: 'Contract', value: <TruncatedText value={item.contractAddress} copyable={true} />},
                            {label: 'Topic', value: <TruncatedText value={item.topic} copyable={true} />},
                            ...roleItems,
                            {label: 'Identifier', value: <TruncatedText value={item.identifier} copyable={true} />},
                            {label: 'Request time', value: formatTimestamp(item.requestTimestamp)},
                            {label: 'Request timestamp', value: fmt(item.requestTimestamp)},
                            {label: 'Market ID', value: <TruncatedText value={item.marketId} copyable={true} />},
                            {label: 'Condition ID', value: <TruncatedText value={item.conditionId} copyable={true} />},
                            {label: 'Event slug', value: <TruncatedText value={item.eventSlug} copyable={true} />},
                            {label: 'Market slug', value: <TruncatedText value={item.marketSlug} copyable={true} />},
                            {label: 'Proposed price', value: <span className='managed-oo-price'>{item.proposedPrice || 'Unknown'}</span>},
                            ...('expirationTimestamp' in item
                                ? [
                                      {label: 'Expiration', value: formatTimestamp(item.expirationTimestamp)},
                                      {label: 'Expiration timestamp', value: fmt(item.expirationTimestamp)},
                                      {label: 'Currency', value: <TruncatedText value={item.currency} copyable={true} />}
                                  ]
                                : []),
                            {label: 'Fetched at', value: formatBeijingDateTime(item.fetchedAt) || '-'},
                            {label: 'Ancillary data', value: <TruncatedText value={item.ancillaryDataText} copyable={true} />}
                        ]}
                    />
                    <details className='radar-facts'>
                        <summary>Raw evidence</summary>
                        <KeyValueGrid
                            columns={1}
                            items={[
                                {label: 'Ancillary data hex', value: <TruncatedText value={item.ancillaryDataHex} copyable={true} />},
                                {label: 'Raw topics', value: <TruncatedText value={item.rawTopics} copyable={true} />},
                                {label: 'Raw data', value: <TruncatedText value={item.rawData} copyable={true} />}
                            ]}
                        />
                    </details>
                </Space>
            )}
        </Drawer>
    );
};

const ManagedOOPage = (props: {kind: ManagedOOKind}) => {
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.ManagedOO);
    const canWriteRef = React.useRef(canWrite);
    canWriteRef.current = canWrite;
    const activeRequest = React.useRef(0);
    React.useEffect(
        () => () => {
            activeRequest.current += 1;
            canWriteRef.current = false;
        },
        []
    );
    const ctx = React.useContext(Context);
    const {params, setParams, page, pageSize, setPage} = usePagedParams();
    const blockParam = params.get('block') || params.get('block_number') || '';
    const parsedBlock = Number(blockParam);
    const blockNumber = Number.isSafeInteger(parsedBlock) && parsedBlock > 0 ? parsedBlock : undefined;
    const [scanBlock, setScanBlock] = React.useState<number | null>(blockNumber || null);
    const [scanning, setScanning] = React.useState(false);
    const [selected, setSelected] = React.useState<ManagedOOItem>();
    const data = useAsyncData<ListManagedOOItemsResult<ManagedOOItem>>(
        () =>
            (props.kind === 'proposal' ? services.managedOO.listProposals(page, pageSize, blockNumber) : services.managedOO.listDisputes(page, pageSize, blockNumber)) as Promise<
                ListManagedOOItemsResult<ManagedOOItem>
            > & {abort?: () => void},
        [props.kind, page, pageSize, blockNumber]
    );

    React.useEffect(() => setScanBlock(blockNumber || null), [blockNumber]);
    React.useEffect(() => {
        if (!canWrite) {
            activeRequest.current += 1;
            setScanning(false);
            setScanBlock(blockNumber || null);
        }
    }, [blockNumber, canWrite]);

    const scan = async () => {
        if (!canWrite) {
            return;
        }
        if (!scanBlock || !Number.isSafeInteger(scanBlock) || scanBlock <= 0) {
            ctx.notifications.error('Invalid block number', 'Enter a positive Polygon block number within JavaScript’s safe integer range.');
            return;
        }
        const request = ++activeRequest.current;
        const requestedBlock = scanBlock;
        setScanning(true);
        try {
            const result = await services.managedOO.scanBlock(requestedBlock);
            if (request !== activeRequest.current || !canWriteRef.current) return;
            const next = new URLSearchParams(params);
            next.set('block', String(requestedBlock));
            next.delete('block_number');
            next.set('page', '1');
            setParams(next);
            ctx.notifications.success(`Block ${result.blockNumber} parsed`, `Proposals: ${result.proposalCount}; Disputes: ${result.disputeCount}`);
            data.reload();
        } catch (err: any) {
            if (request === activeRequest.current && canWriteRef.current) {
                ctx.notifications.error('Block parse failed', err?.message || 'Could not parse this Polygon block.');
            }
        } finally {
            if (request === activeRequest.current) setScanning(false);
        }
    };

    const clearBlockFilter = () => {
        const next = new URLSearchParams(params);
        next.delete('block');
        next.delete('block_number');
        next.set('page', '1');
        setParams(next);
        setScanBlock(null);
    };

    const question = (item: ManagedOOItem) => (
        <div className='managed-oo-record'>
            <h3>{item.question || `Market ${item.marketId || 'Unknown'}`}</h3>
            {!item.question && <p className='radar-warmup'>Question not available</p>}
            <span className='radar-description'>
                Market {item.marketId || 'Unknown'} · {item.polymarketUrl ? externalLink(item.polymarketUrl, 'Polymarket') : 'Market link unavailable'}
            </span>
        </div>
    );
    const address = (item: ManagedOOItem) => (
        <div className='managed-oo-address'>
            <TruncatedText value={(props.kind === 'proposal' ? item.proposer : 'disputer' in item ? item.disputer : '') || 'Unknown'} copyable />
            {item.txHash ? externalLink(explorerURL(item.txHash), 'Transaction') : 'Transaction unavailable'}
            <Button type='text' onClick={() => setSelected(item)}>
                View evidence
            </Button>
        </div>
    );
    const role = props.kind === 'proposal' ? 'Proposer' : 'Disputer';
    const columns: ColumnsType<ManagedOOItem> = [
        {title: 'Question / Market', width: '30%', render: (_value: unknown, item: ManagedOOItem) => question(item)},
        {
            title: 'Block',
            render: (_value: unknown, item: ManagedOOItem) => (
                <div className='athena-number'>
                    {formatBlockNumber(item.blockNumber)}
                    <p className='radar-description'>Log {item.logIndex}</p>
                </div>
            )
        },
        {
            title: 'Proposed price',
            dataIndex: 'proposedPrice',
            render: (value: string) => (
                <div className='managed-oo-price'>
                    <span>{value || 'Unknown'}</span>
                    <p className='radar-description'>Raw oracle value</p>
                </div>
            )
        },
        {title: 'Request time · UTC+8', dataIndex: 'requestTimestamp', render: formatTimestamp},
        {title: role, width: '20%', render: (_value: unknown, item: ManagedOOItem) => address(item)}
    ];
    const compact = (item: ManagedOOItem) => (
        <div className='managed-oo-record'>
            {question(item)}
            <dl className='market-fact-grid'>
                <div>
                    <dt>Block</dt>
                    <dd className='athena-number'>
                        {formatBlockNumber(item.blockNumber)} · Log {item.logIndex}
                    </dd>
                </div>
                <div>
                    <dt>Request time · UTC+8</dt>
                    <dd>{formatTimestamp(item.requestTimestamp)}</dd>
                </div>
                <div>
                    <dt>Proposed price</dt>
                    <dd className='managed-oo-price'>
                        <span>{item.proposedPrice || 'Unknown'}</span>
                        <p className='radar-description'>Raw oracle value</p>
                    </dd>
                </div>
                <div>
                    <dt>{role}</dt>
                    <dd>{address(item)}</dd>
                </div>
            </dl>
        </div>
    );

    const title = props.kind === 'proposal' ? 'Managed OO Proposals' : 'Managed OO Disputes';
    return (
        <div className='market-intelligence-page managed-oo-page'>
            <AppPage
                title={title}
                subtitle={
                    blockNumber
                        ? `Showing Polygon block ${blockNumber}`
                        : props.kind === 'proposal'
                          ? 'Proposed oracle values and their Polygon transaction evidence.'
                          : 'Disputed oracle requests and the accounts involved.'
                }
                loading={data.loading}
                error={data.error}
                onRefresh={data.reload}>
                <nav className='managed-oo-tabs' aria-label='Oracle event type'>
                    <Link to='/managed-oo/proposals' aria-current={props.kind === 'proposal' ? 'page' : undefined}>
                        Proposals
                    </Link>
                    <Link to='/managed-oo/disputes' aria-current={props.kind === 'dispute' ? 'page' : undefined}>
                        Disputes
                    </Link>
                </nav>
                {canWrite && (
                    <Section title='Parse one Polygon block'>
                        <div className='managed-oo-parser'>
                            <label>
                                Block number
                                <InputNumber
                                    aria-label='Polygon block number'
                                    value={scanBlock}
                                    min={1}
                                    max={Number.MAX_SAFE_INTEGER}
                                    precision={0}
                                    controls={false}
                                    placeholder='Block number'
                                    disabled={scanning}
                                    onChange={value => setScanBlock(typeof value === 'number' ? value : null)}
                                    onPressEnter={scan}
                                />
                            </label>
                            <Button aria-label='Parse block' type='primary' icon={<SearchOutlined />} loading={scanning} disabled={scanning || !scanBlock} onClick={scan}>
                                Parse block
                            </Button>
                            {blockNumber && (
                                <Button disabled={scanning} onClick={clearBlockFilter}>
                                    Clear block filter
                                </Button>
                            )}
                        </div>
                        <p className='radar-description'>Read events from one block, then show its proposals or disputes.</p>
                    </Section>
                )}
                {data.error && data.data && <p className='radar-warmup'>Stale saved events. Refresh to retry.</p>}
                {data.data && (
                    <ResourceTable<ManagedOOItem>
                        rowKey={item => `${item.txHash}:${item.logIndex}`}
                        items={data.data?.items || []}
                        columns={columns}
                        loading={data.loading}
                        total={data.data?.total || 0}
                        page={page}
                        pageSize={pageSize}
                        onPageChange={setPage}
                        compactRender={compact}
                        label='Saved oracle events'
                    />
                )}
                <ManagedOODetailDrawer item={selected} kind={props.kind} onClose={() => setSelected(undefined)} />
            </AppPage>
        </div>
    );
};

export const ManagedOOProposalsPage = () => <ManagedOOPage kind='proposal' />;
export const ManagedOODisputesPage = () => <ManagedOOPage kind='dispute' />;
