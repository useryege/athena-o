import {LinkOutlined, SearchOutlined} from '@ant-design/icons';
import {Button, Drawer, InputNumber, Space, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, KeyValueGrid, ResourceTable, Section, TruncatedText, useAsyncData} from '../components';
import {Context} from '../shared/context';
import {formatBeijingDateTime, formatBeijingUnixSeconds, formatBlockNumber} from '../shared/format';
import {services} from '../shared/services';
import {ListPolymarketUMAResult, PolymarketUMADisputeItem, PolymarketUMAProposalItem} from '../shared/services/polymarket-service';
import {fmt, short, usePagedParams} from './shared';

type UMAItem = PolymarketUMAProposalItem | PolymarketUMADisputeItem;
type UMAKind = 'proposal' | 'dispute';

const explorerURL = (txHash: string) => `https://polygonscan.com/tx/${encodeURIComponent(txHash)}`;

const formatTimestamp = (value: number) => formatBeijingUnixSeconds(value) || '-';

const externalLink = (href: string, label: React.ReactNode) =>
    href ? (
        <a href={href} target='_blank' rel='noreferrer' onClick={event => event.stopPropagation()}>
            {label} <LinkOutlined />
        </a>
    ) : (
        '-'
    );

const UMADetailDrawer = (props: {item?: UMAItem; kind: UMAKind; onClose: () => void}) => {
    const item = props.item;
    const roleItems = item
        ? [
              {label: 'Requester', value: <TruncatedText value={item.requester} copyable={true} />},
              {label: 'Proposer', value: <TruncatedText value={item.proposer} copyable={true} />},
              ...('disputer' in item ? [{label: 'Disputer', value: <TruncatedText value={item.disputer} copyable={true} />}] : [])
          ]
        : [];
    return (
        <Drawer title={props.kind === 'proposal' ? 'UMA Proposed details' : 'UMA Disputed details'} open={Boolean(item)} onClose={props.onClose} placement='right' size={720}>
            {item && (
                <Space direction='vertical' size='large' style={{width: '100%'}}>
                    <Typography.Title level={5}>{item.question || item.marketId || 'UMA event'}</Typography.Title>
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
                            {label: 'Proposed price', value: item.proposedPrice},
                            ...('expirationTimestamp' in item
                                ? [
                                      {label: 'Expiration', value: formatTimestamp(item.expirationTimestamp)},
                                      {label: 'Expiration timestamp', value: fmt(item.expirationTimestamp)},
                                      {label: 'Currency', value: <TruncatedText value={item.currency} copyable={true} />}
                                  ]
                                : []),
                            {label: 'Fetched at', value: formatBeijingDateTime(item.fetchedAt) || '-'},
                            {label: 'Ancillary data', value: <TruncatedText value={item.ancillaryDataText} copyable={true} />},
                            {label: 'Ancillary data hex', value: <TruncatedText value={item.ancillaryDataHex} copyable={true} />},
                            {label: 'Raw topics', value: <TruncatedText value={item.rawTopics} copyable={true} />},
                            {label: 'Raw data', value: <TruncatedText value={item.rawData} copyable={true} />}
                        ]}
                    />
                </Space>
            )}
        </Drawer>
    );
};

const PolymarketUMAPage = (props: {kind: UMAKind; canScan: boolean}) => {
    const ctx = React.useContext(Context);
    const {params, setParams, page, pageSize, setPage} = usePagedParams();
    const blockParam = params.get('block') || params.get('block_number') || '';
    const parsedBlock = Number(blockParam);
    const blockNumber = Number.isSafeInteger(parsedBlock) && parsedBlock > 0 ? parsedBlock : undefined;
    const [scanBlock, setScanBlock] = React.useState<number | null>(blockNumber || null);
    const [scanning, setScanning] = React.useState(false);
    const [selected, setSelected] = React.useState<UMAItem>();
    const data = useAsyncData<ListPolymarketUMAResult<UMAItem>>(
        () =>
            (props.kind === 'proposal'
                ? services.polymarket.listUMAProposals(page, pageSize, blockNumber)
                : services.polymarket.listUMADisputes(page, pageSize, blockNumber)) as Promise<ListPolymarketUMAResult<UMAItem>> & {abort?: () => void},
        [props.kind, page, pageSize, blockNumber]
    );

    React.useEffect(() => setScanBlock(blockNumber || null), [blockNumber]);

    const scan = async () => {
        if (!scanBlock || !Number.isSafeInteger(scanBlock) || scanBlock <= 0) {
            ctx.notifications.error('Invalid block number', 'Enter a positive Polygon block number within JavaScript’s safe integer range.');
            return;
        }
        setScanning(true);
        try {
            const result = await services.polymarket.scanManagedOOBlock(scanBlock);
            const next = new URLSearchParams(params);
            next.set('block', String(scanBlock));
            next.delete('block_number');
            next.set('page', '1');
            setParams(next);
            ctx.notifications.success(`Block ${result.blockNumber} parsed`, `UMA Proposed: ${result.proposalCount}; UMA Disputed: ${result.disputeCount}`);
            data.reload();
        } catch (err: any) {
            ctx.notifications.error('Block parse failed', err?.message || 'Could not parse this Polygon block.');
        } finally {
            setScanning(false);
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

    const columns: ColumnsType<UMAItem> = [
        {title: 'Block', render: item => formatBlockNumber(item.blockNumber)},
        {title: 'Question', dataIndex: 'question', width: 360, render: value => value || '-'},
        {title: props.kind === 'proposal' ? 'Proposer' : 'Disputer', render: item => short(props.kind === 'proposal' ? item.proposer : 'disputer' in item ? item.disputer : '')},
        {title: 'Price', dataIndex: 'proposedPrice'},
        {title: 'Request time', dataIndex: 'requestTimestamp', render: formatTimestamp},
        {title: 'Transaction', render: item => externalLink(explorerURL(item.txHash), short(item.txHash))},
        {title: 'Market', render: item => externalLink(item.polymarketUrl, item.marketSlug || 'Open')}
    ];

    const title = props.kind === 'proposal' ? 'UMA Proposed' : 'UMA Disputed';
    return (
        <AppPage
            title={title}
            subtitle={blockNumber ? `Showing Polygon block ${blockNumber}` : 'Managed Optimistic Oracle events'}
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}>
            {props.canScan && (
                <Section title='Parse one Polygon block'>
                    <Space wrap={true}>
                        <InputNumber
                            aria-label='Polygon block number'
                            value={scanBlock}
                            min={1}
                            max={Number.MAX_SAFE_INTEGER}
                            precision={0}
                            controls={false}
                            placeholder='Block number'
                            disabled={scanning}
                            style={{width: 220}}
                            onChange={value => setScanBlock(typeof value === 'number' ? value : null)}
                            onPressEnter={scan}
                        />
                        <Button type='primary' icon={<SearchOutlined />} loading={scanning} disabled={scanning || !scanBlock} onClick={scan}>
                            Parse block
                        </Button>
                        {blockNumber && (
                            <Button disabled={scanning} onClick={clearBlockFilter}>
                                Clear block filter
                            </Button>
                        )}
                    </Space>
                </Section>
            )}
            <ResourceTable<UMAItem>
                rowKey={item => `${item.txHash}:${item.logIndex}`}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total || 0}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                onItemClick={item => setSelected(item)}
            />
            <UMADetailDrawer item={selected} kind={props.kind} onClose={() => setSelected(undefined)} />
        </AppPage>
    );
};

export const PolymarketUMAProposedPage = (props: {canScan: boolean}) => <PolymarketUMAPage kind='proposal' canScan={props.canScan} />;
export const PolymarketUMADisputedPage = (props: {canScan: boolean}) => <PolymarketUMAPage kind='dispute' canScan={props.canScan} />;
