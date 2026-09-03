import {Alert, Button, Card, Drawer, Empty, Input, Pagination, Select, Skeleton, Space, Table, Tabs, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {Link, useParams} from 'react-router-dom';
import {AppPage, KeyValueGrid, ResourceTable, Section, StatusTag, TruncatedText, useAsyncData} from '../../components';
import {formatBeijingDateTime, formatBlockNumber} from '../../shared/format';
import {
    TokenCollectionTask,
    TokenProjectDetail,
    TokenProjectListItem,
    TokenProjectProfilePair,
    TokenProjectRelatedWallet,
    TokenWalletNormalTransaction
} from '../../shared/services/token-service';
import {memberServices as services} from '../services';
import {ProjectExplorerValue as ExplorerValue, ProjectTimeValue as TimeValue} from './project-detail-values';
import {ProjectJSONDrawer, ProjectJSONDrawerValue} from './project-json-drawer';
import {useProjectDetailReturn, useScrollProjectDetailOnPush} from './project-navigation';
import {ProjectProfileTab} from './project-profile-tab';
import {ProjectSwapActivityTab} from './project-swap-activity';
import {ChainBadge, TokenLogo} from './token-shared';

const DATA_TYPES = ['chain_state', 'wallet_asset_state', 'simulation_result', 'ave', 'contract_code_source', 'wallet_normal_transactions'];
const DETAIL_POLL_INTERVAL_MS = 30_000;
const dataTypeLabel = (value?: string) => ({
    chain_state: 'Chain state',
    wallet_asset_state: 'Wallet assets',
    simulation_result: 'Simulation calls',
    ave: 'Ave market',
    contract_code_source: 'Contract source',
    wallet_normal_transactions: 'Pre-deploy transactions'
}[value || ''] || value || 'Unknown source');
const hasValue = (value: unknown) => value !== undefined && value !== null && value !== '';
const compact = (value?: string | number, currency = false) => {
    if (!hasValue(value)) return '-';
    const numeric = Number(value);
    if (!Number.isFinite(numeric)) return String(value);
    const formatted = new Intl.NumberFormat(undefined, {notation: Math.abs(numeric) >= 100000 ? 'compact' : 'standard', maximumFractionDigits: Math.abs(numeric) < 1 ? 8 : 2}).format(numeric);
    return currency ? `$${formatted}` : formatted;
};
const taskTone = (status?: string) => ({positive: status === 'succeeded', negative: status === 'failed'});
const workflowTag = (value?: string) => {
    const color: Record<string, string> = {queued: 'default', collecting: 'blue', pending: 'gold', complete: 'green', incomplete: 'gold', needs_attention: 'red', failed: 'red'};
    return <Tag color={value ? color[value] : undefined}>{value?.replace(/_/g, ' ') || 'unknown'}</Tag>;
};
const pairSignal = (value?: boolean) => value === undefined
    ? <StatusTag value='Unknown' />
    : <StatusTag value={value ? 'Detected' : 'Clear'} positive={!value} negative={value} />;

const Summary = (props: {detail: TokenProjectDetail; listItem?: TokenProjectListItem}) => {
    const project = props.detail.project;
    const profile = props.detail.profile;
    const market = profile?.market;
    const succeeded = props.detail.collectionTasks.filter(task => task.status === 'succeeded').length;
    return (
        <div className='project-detail-summary'>
            <Card className='project-detail-identity' size='small'>
                <div className='project-detail-identity__heading'>
                    <TokenLogo logoURL={market?.logoURL} symbol={project?.symbol} size='detail' />
                    <div>
                        <Typography.Title level={3}>{project?.name || 'Unnamed token'}</Typography.Title>
                        <Space wrap={true}>
                            <ChainBadge chainID={project?.chainID} />
                            <Tag>Project #{project?.projectID || '-'}</Tag>
                            {workflowTag(props.listItem?.collectionStatus)}
                            {workflowTag(props.listItem?.profileState)}
                        </Space>
                    </div>
                </div>
                <KeyValueGrid columns={2} items={[
                    {label: 'Contract', value: <ExplorerValue chainID={project?.chainID} kind='address' value={project?.contract} />},
                    {label: 'Deployment transaction', value: <ExplorerValue chainID={project?.chainID} kind='tx' value={project?.txHash} />},
                    {label: 'Deployer', value: <ExplorerValue chainID={project?.chainID} kind='address' value={project?.txSender} />},
                    {label: 'Deployment block', value: formatBlockNumber(project?.blockNumber)},
                    {label: 'Deployment nonce', value: project?.deploymentNonce ?? '-'},
                    {label: 'Block time', value: <TimeValue unixSeconds={project?.blockTime} />},
                    {label: 'Code hash', value: project?.codeHash ? <Link to={`/token/contract-codes/${encodeURIComponent(project.codeHash)}`}>{project.codeHash}</Link> : '-'},
                    {label: 'Discovered project saved', value: <TimeValue value={project?.createdAt} />}
                ]} />
            </Card>
            <div className='project-detail-kpis' aria-label='Project profile summary'>
                {[
                    ['Collection', `${succeeded}/6`],
                    ['Profile', profile?.completenessStatus || props.listItem?.profileState || 'Pending'],
                    ['Price', compact(market?.currentPriceUSD, true)],
                    ['Market Cap', compact(market?.marketCapUSD, true)],
                    ['TVL', compact(market?.tvlUSD, true)],
                    ['Holders', compact(market?.holders)]
                ].map(([label, value]) => <Card className='project-detail-kpi' size='small' key={label}><Typography.Text type='secondary'>{label}</Typography.Text><strong>{value}</strong></Card>)}
            </div>
        </div>
    );
};

const PairDetails = (props: {title: string; pair?: TokenProjectProfilePair; chainID?: number}) => {
    const chainState = props.pair?.chainState;
    return (
        <Card
            title={props.title}
            size='small'
            extra={chainState
                ? <StatusTag value={chainState.isCreated ? 'Created' : 'Not created'} positive={chainState.isCreated} />
                : props.pair ? <StatusTag value='Chain state unavailable' /> : undefined}
        >
            {props.pair ? <>
                <KeyValueGrid columns={2} items={[
                    {label: 'Pair address', value: <ExplorerValue chainID={props.chainID} kind='address' value={props.pair.address} />},
                    {label: 'Pair kind', value: props.pair.kind || '-'}
                ]} />
                {chainState ? <KeyValueGrid columns={2} items={[
                    {label: 'Base balance', value: chainState.baseBalance || '-'},
                    {label: 'Quote balance', value: chainState.quoteBalance || '-'},
                    {label: 'Quote value in USDT', value: chainState.quoteUsdtValueInt || chainState.quoteUsdtValue || '-'},
                    {label: 'Reserve updated time', value: <TimeValue unixSeconds={chainState.reserveUpdatedAt} />},
                    {label: 'Pair token balance exceeds total supply', value: pairSignal(chainState.signals?.pairTokenBalanceExceedsTotalSupply)},
                    {label: 'LP minimum supply only', value: pairSignal(chainState.signals?.lpMinimumSupplyOnly)},
                    {label: 'Fixed fee address LP share ≥ 90%', value: pairSignal(chainState.signals?.fixedFeeAddressLpShareGte90Percent)}
                ]} /> : <Alert type='warning' showIcon={true} title='Chain-state evidence unavailable' description='On-chain balances, liquidity, and risk signals are unknown for this pair.' />}
                {props.pair.market && <KeyValueGrid columns={2} items={[
                    {label: 'Ave AMM', value: props.pair.market.amm || '-'},
                    {label: 'Ave volume in USD', value: compact(props.pair.market.volumeUSD, true)},
                    {label: 'Ave market cap', value: compact(props.pair.market.marketCapUSD, true)},
                    {label: 'Ave FDV', value: compact(props.pair.market.fdvUSD, true)}
                ]} />}
            </> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Pair profile is unavailable' />}
        </Card>
    );
};

const MarketLiquidityTab = (props: {detail: TokenProjectDetail}) => {
    const profile = props.detail.profile;
    const market = profile?.market;
    return (
        <div className='project-detail-tab'>
            <Section title='Ave market snapshot'>
                {market ? <KeyValueGrid columns={3} items={[
                    {label: 'Price USD', value: compact(market.currentPriceUSD, true)},
                    {label: 'Price ETH', value: compact(market.currentPriceETH)},
                    {label: 'Market cap', value: compact(market.marketCapUSD, true)},
                    {label: 'FDV', value: compact(market.fdvUSD, true)},
                    {label: 'TVL', value: compact(market.tvlUSD, true)},
                    {label: 'Main pair TVL', value: compact(market.mainPairTVLUSD, true)},
                    {label: 'Holders', value: compact(market.holders)},
                    {label: 'Launch time', value: <TimeValue value={market.launchAt} />},
                    {label: 'Provider updated', value: <TimeValue value={market.providerUpdatedAt} />}
                ]} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Market data is unavailable until a profile is built' />}
            </Section>
            <Section title='Canonical pair snapshots'>
                <div className='project-profile-pair-grid'>
                    <PairDetails title='WETH / WBNB' pair={profile?.wrappedNativePair} chainID={props.detail.project?.chainID} />
                    <PairDetails title='USDT' pair={profile?.usdtPair} chainID={props.detail.project?.chainID} />
                </div>
            </Section>
        </div>
    );
};

interface WalletRow extends TokenProjectRelatedWallet {
    initialRatioBPS?: number;
    transactionCount?: number;
}
const WalletsTab = (props: {detail: TokenProjectDetail}) => {
    const rows = new Map<string, WalletRow>();
    props.detail.relatedWallets.forEach(wallet => rows.set((wallet.wallet || '').toLowerCase(), {...wallet}));
    props.detail.initialRecipients.forEach(recipient => {
        const key = (recipient.wallet || '').toLowerCase();
        rows.set(key, {...rows.get(key), projectID: recipient.projectID, wallet: recipient.wallet, role: rows.get(key)?.role || 'initial_recipient', initialRatioBPS: recipient.ratioBPS});
    });
    props.detail.walletTransactionCounts.forEach(count => {
        const key = (count.wallet || '').toLowerCase();
        rows.set(key, {...rows.get(key), wallet: count.wallet, transactionCount: count.transactionCount});
    });
    const columns: ColumnsType<WalletRow> = [
        {title: 'Wallet', render: item => <ExplorerValue chainID={props.detail.project?.chainID} kind='address' value={item.wallet} compact={true} />},
        {title: 'Role', render: item => <Tag>{item.role || 'related'}</Tag>},
        {title: 'Deployment-time received estimate', render: item => item.initialRatioBPS === undefined ? '-' : `${(item.initialRatioBPS / 100).toFixed(2)}%`},
        {title: 'Sampled pre-deploy transactions', render: item => item.transactionCount ?? 0},
        {title: 'Linked', render: item => formatBeijingDateTime(item.createdAt) || '-'}
    ];
    return <div className='project-detail-tab'><Section title='Related wallet roles'><Table<WalletRow> rowKey={item => item.wallet || item.role || 'wallet'} columns={columns} dataSource={[...rows.values()]} pagination={false} scroll={{x: 980}} locale={{emptyText: 'No related wallets'}} /></Section><Section title='Wallet profile summary'><KeyValueGrid columns={3} items={[
        {label: 'Wallets collected', value: props.detail.profile?.walletSummary?.walletCount ?? '-'},
        {label: 'Tracked asset value in USDT', value: props.detail.profile?.walletSummary?.trackedAssetUsdtValueTotal || '-'},
        {label: 'Wallets with successful simulation-call signals', value: props.detail.profile?.walletSummary?.walletsWithSimulationSignals ?? '-'}
    ]} /></Section></div>;
};

const PreDeployTransactionsTab = (props: {projectID: number; active: boolean; refreshVersion: number; detail: TokenProjectDetail}) => {
    const [page, setPage] = React.useState(1);
    const [wallet, setWallet] = React.useState('');
    const [receiptStatus, setReceiptStatus] = React.useState('');
    const [methodID, setMethodID] = React.useState('');
    const data = useAsyncData(() => props.active ? services.tokenapi.listProjectWalletNormalTransactions(props.projectID, {wallet: wallet || undefined, receiptStatus: receiptStatus || undefined, methodID: methodID || undefined, page, pageSize: 20}) : Promise.resolve({items: [], total: 0, page: 1, pageSize: 20}), [props.projectID, props.active, props.refreshVersion, wallet, receiptStatus, methodID, page]);
    const columns: ColumnsType<TokenWalletNormalTransaction> = [
        {title: 'Wallet', width: 160, render: item => <ExplorerValue chainID={props.detail.project?.chainID} kind='address' value={item.wallet} compact={true} />},
        {title: 'Transaction', width: 170, render: item => <ExplorerValue chainID={props.detail.project?.chainID} kind='tx' value={item.transactionHash} compact={true} />},
        {title: 'Block', width: 100, render: item => formatBlockNumber(item.blockNumber)},
        {title: 'Time', width: 175, render: item => formatBeijingDateTime(item.blockTimestamp) || '-'},
        {title: 'Direction', width: 220, render: item => <span><TruncatedText value={item.fromAddress} singleLine={true} /> → <TruncatedText value={item.toAddress} singleLine={true} /></span>},
        {title: 'Value', width: 150, dataIndex: 'value'},
        {title: 'Method', width: 170, render: item => item.functionName || item.methodID || '-'},
        {title: 'Status', width: 110, render: item => <StatusTag value={item.receiptStatus || (item.isError ? 'failed' : 'unknown')} positive={!item.isError && item.receiptStatus === 'success'} negative={item.isError || item.receiptStatus === 'failed'} />}
    ];
    return <div className='project-detail-tab'><Section title='Sampled pre-deployment transactions' extra={<Typography.Text type='secondary'>Blocks 0–{Math.max(0, (props.detail.project?.blockNumber || 1) - 1)} · up to 300 per wallet</Typography.Text>}>
        <Space wrap={true} className='project-detail-inline-filters'>
            <Select aria-label='Filter by related wallet' allowClear={true} value={wallet || undefined} placeholder='All wallets' style={{minWidth: 220}} options={props.detail.relatedWallets.map(item => ({value: item.wallet || '', label: item.wallet || '-'}))} onChange={value => {setWallet(value || ''); setPage(1);}} />
            <Select aria-label='Filter by receipt status' allowClear={true} value={receiptStatus || undefined} placeholder='All receipt states' options={['success', 'failed'].map(value => ({value, label: value}))} onChange={value => {setReceiptStatus(value || ''); setPage(1);}} />
            <Input aria-label='Filter by method ID' value={methodID} placeholder='Method ID' onChange={event => {setMethodID(event.target.value); setPage(1);}} />
        </Space>
        {data.error && <Alert type='error' showIcon={true} title='Could not load transactions' description={data.error.message} />}
        <ResourceTable rowKey={item => `${item.wallet}-${item.transactionHash}`} items={data.data?.items || []} columns={columns} loading={data.loading} scrollX={1255} />
        {(data.data?.total || 0) > 20 && <Pagination current={page} pageSize={20} total={data.data?.total || 0} showSizeChanger={false} onChange={setPage} />}
    </Section></div>;
};

const ContractTab = (props: {detail: TokenProjectDetail}) => {
    const project = props.detail.project;
    const source = props.detail.profile?.contractSource;
    return <div className='project-detail-tab'><Section title='Contract and source status'><KeyValueGrid columns={2} items={[
        {label: 'Contract', value: <ExplorerValue chainID={project?.chainID} kind='address' value={project?.contract} />},
        {label: 'Code hash', value: project?.codeHash ? <Link to={`/token/contract-codes/${encodeURIComponent(project.codeHash)}`}>{project.codeHash}</Link> : '-'},
        {label: 'Verification status', value: source ? <StatusTag value={source.verificationStatus || 'unknown'} positive={source.verificationStatus === 'verified'} /> : '-'},
        {label: 'Artifact reference', value: source?.artifactReference || '-'},
        {label: 'Token decimals', value: project?.decimals ?? '-'},
        {label: 'Total supply', value: project?.totalSupply || '-'}
    ]} /></Section></div>;
};

const EvidenceDrawer = (props: {task?: TokenCollectionTask; loading: boolean; error?: Error; onClose: () => void}) => (
    <Drawer title={props.task ? `${dataTypeLabel(props.task.dataType)} evidence` : 'Collection evidence'} width='min(780px, calc(100vw - 24px))' open={props.loading || Boolean(props.task) || Boolean(props.error)} onClose={props.onClose} keyboard={true}>
        {props.loading ? <Skeleton active={true} /> : props.error ? <Alert type='error' showIcon={true} title='Could not load evidence' description={props.error.message} /> : props.task ? <div className='collection-task-detail'>
            <KeyValueGrid columns={2} items={[
                {label: 'Task', value: props.task.taskID || '-'}, {label: 'Project', value: props.task.projectID || '-'},
                {label: 'Status', value: <StatusTag value={props.task.status} {...taskTone(props.task.status)} />}, {label: 'Failures', value: `${props.task.failureCount || 0}/3`},
                {label: 'Claim generation', value: props.task.claimGeneration || '-'}, {label: 'Available', value: formatBeijingDateTime(props.task.availableAt) || '-'},
                {label: 'Locked', value: formatBeijingDateTime(props.task.lockedAt) || '-'}, {label: 'Lease expires', value: formatBeijingDateTime(props.task.leaseExpiresAt) || '-'},
                {label: 'Finished', value: formatBeijingDateTime(props.task.finishedAt) || '-'}, {label: 'Collected', value: formatBeijingDateTime(props.task.result?.collectedAt) || '-'},
                {label: 'Block', value: formatBlockNumber(props.task.result?.blockNumber)}, {label: 'Content hash', value: <TruncatedText value={props.task.result?.contentHash} copyable={true} />},
                {label: 'Last error', value: <TruncatedText value={props.task.lastError} />}
            ]} />
            {props.task.result?.payloadJSON ? <pre className='code-block project-json-viewer' tabIndex={0} aria-label={`${dataTypeLabel(props.task.dataType)} normalized evidence JSON`}>{props.task.result.payloadJSON}</pre> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No result payload' />}
        </div> : null}
    </Drawer>
);

const DataSourcesTab = (props: {tasks: TokenCollectionTask[]; openTask: (task: TokenCollectionTask) => void}) => {
    const tasks = new Map(props.tasks.map(task => [task.dataType, task]));
    return <div className='project-detail-tab'><Section title='One-time collection evidence'><div className='project-data-source-grid'>
        {DATA_TYPES.map(dataType => {
            const task = tasks.get(dataType);
            return <Card className='project-data-source-card' size='small' key={dataType} title={dataTypeLabel(dataType)} extra={<StatusTag value={task?.status || 'pending'} {...taskTone(task?.status)} />}>
                <KeyValueGrid columns={1} items={[
                    {label: 'Failure count', value: `${task?.failureCount || 0}/3`},
                    {label: 'Available', value: formatBeijingDateTime(task?.availableAt) || '-'},
                    {label: 'Started', value: formatBeijingDateTime(task?.lockedAt) || '-'},
                    {label: 'Finished', value: formatBeijingDateTime(task?.finishedAt) || '-'},
                    {label: 'Collected', value: formatBeijingDateTime(task?.result?.collectedAt) || '-'},
                    {label: 'Last error', value: <TruncatedText value={task?.lastError} />}
                ]} />
                <Button block={true} disabled={!task?.taskID} onClick={() => task && props.openTask(task)}>Open full evidence</Button>
            </Card>;
        })}
    </div></Section></div>;
};

export const ProjectDetailPage = () => {
    const {projectID: projectIDParam} = useParams();
    const projectID = Number(projectIDParam || 0);
    const returnToProjects = useProjectDetailReturn();
    useScrollProjectDetailOnPush();
    const [activeTab, setActiveTab] = React.useState('profile');
    const [refreshVersion, setRefreshVersion] = React.useState(0);
    const [jsonContent, setJSONContent] = React.useState<ProjectJSONDrawerValue>();
    const [evidenceTask, setEvidenceTask] = React.useState<TokenCollectionTask>();
    const [evidenceLoading, setEvidenceLoading] = React.useState(false);
    const [evidenceError, setEvidenceError] = React.useState<Error>();
    const evidenceRequest = React.useRef<(Promise<TokenCollectionTask | undefined> & {abort?: () => void})>();
    const detail = useAsyncData(() => services.tokenapi.getProjectDetail(projectID), [projectID]);
    const summary = useAsyncData(() => services.tokenapi.listProjects({projectID, page: 1, pageSize: 1}), [projectID]);
    const listItem = summary.data?.items[0];
    const reloadDetailRef = React.useRef(detail.reload);
    const reloadSummaryRef = React.useRef(summary.reload);
    reloadDetailRef.current = detail.reload;
    reloadSummaryRef.current = summary.reload;
    const hasActiveWorkflow = Boolean(detail.data?.collectionTasks.some(task => task.status === 'pending' || task.status === 'running') || listItem?.profileState === 'pending');
    React.useEffect(() => {
        if (!hasActiveWorkflow) return;
        const refresh = () => {
            if (document.visibilityState === 'visible') { reloadDetailRef.current(); reloadSummaryRef.current(); }
        };
        const timer = window.setInterval(refresh, DETAIL_POLL_INTERVAL_MS);
        document.addEventListener('visibilitychange', refresh);
        return () => {window.clearInterval(timer); document.removeEventListener('visibilitychange', refresh);};
    }, [hasActiveWorkflow]);
    React.useEffect(() => () => evidenceRequest.current?.abort?.(), []);

    const refresh = () => {detail.reload(); summary.reload(); setRefreshVersion(value => value + 1);};
    const openTask = (task: TokenCollectionTask) => {
        if (!task.taskID) return;
        evidenceRequest.current?.abort?.(); setEvidenceTask(undefined); setEvidenceError(undefined); setEvidenceLoading(true);
        const request = services.tokenapi.getCollectionTask(task.taskID); evidenceRequest.current = request;
        request.then(next => {if (evidenceRequest.current === request) {setEvidenceTask(next); setEvidenceLoading(false);}}, error => {if (evidenceRequest.current === request) {setEvidenceError(error instanceof Error ? error : new Error(String(error))); setEvidenceLoading(false);}});
    };
    const closeEvidence = () => {evidenceRequest.current?.abort?.(); evidenceRequest.current = undefined; setEvidenceTask(undefined); setEvidenceLoading(false); setEvidenceError(undefined);};

    if (!Number.isSafeInteger(projectID) || projectID <= 0) return <AppPage title='Project'><Alert type='error' showIcon={true} title='Invalid project ID' /><Button onClick={returnToProjects}>Back to projects</Button></AppPage>;
    if (detail.loading && !detail.data) return <AppPage title='Project' loading={true}><Skeleton active={true} paragraph={{rows: 14}} /></AppPage>;
    if (!detail.data) return <AppPage title='Project' error={detail.error} extra={<Button onClick={returnToProjects}>Back to projects</Button>}><Empty description='Project not found' /></AppPage>;

    const current = detail.data;
    return (
        <AppPage title={`${current.project?.symbol || current.project?.name || 'Project'} profile`} subtitle={`Project #${current.project?.projectID || projectID}`} loading={detail.loading || summary.loading} error={detail.error || summary.error} onRefresh={refresh} extra={<Button onClick={returnToProjects}>Back to projects</Button>}>
            <Summary detail={current} listItem={listItem} />
            <Tabs className='project-detail-tabs' activeKey={activeTab} onChange={setActiveTab} items={[
                {key: 'profile', label: 'Project Profile', children: <ProjectProfileTab project={current.project} profile={current.profile} profileState={listItem?.profileState} openJSON={setJSONContent} />},
                {key: 'market', label: 'Market & Liquidity', children: <MarketLiquidityTab detail={current} />},
                {key: 'swap', label: 'Swap Activity', children: <ProjectSwapActivityTab projectID={projectID} active={activeTab === 'swap'} refreshVersion={refreshVersion} />},
                {key: 'wallets', label: 'Wallets', children: <WalletsTab detail={current} />},
                {key: 'transactions', label: 'Pre-deploy Transactions', children: <PreDeployTransactionsTab projectID={projectID} active={activeTab === 'transactions'} refreshVersion={refreshVersion} detail={current} />},
                {key: 'contract', label: 'Contract', children: <ContractTab detail={current} />},
                {key: 'sources', label: 'Data Sources', children: <DataSourcesTab tasks={current.collectionTasks} openTask={openTask} />}
            ]} />
            <ProjectJSONDrawer content={jsonContent} onClose={() => setJSONContent(undefined)} />
            <EvidenceDrawer task={evidenceTask} loading={evidenceLoading} error={evidenceError} onClose={closeEvidence} />
        </AppPage>
    );
};
