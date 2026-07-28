import {CodeOutlined, CopyOutlined, ExportOutlined} from '@ant-design/icons';
import {Alert, Button, Card, Drawer, Empty, Flex, Pagination, Select, Space, Table, Tabs, Tag, Timeline, Tooltip, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {Link, useNavigate, useParams} from 'react-router-dom';
import {AppPage, ChoiceGroup, KeyValueGrid, ResourceTable, SearchBar, Section, StatusTag, TruncatedText, useAsyncData} from '../components';
import {services} from '../shared/services';
import {
    TokenAvePair,
    TokenChainPair,
    TokenCollectionSchedule,
    TokenCollectionTask,
    TokenProjectDetail,
    TokenProjectObservation,
    TokenReportRevision,
    TokenSelection,
    TokenSimulationResult,
    TokenWalletNormalTransaction
} from '../shared/services/token-service';
import {ProjectTrendChart} from './project-detail-chart';
import {ChainBadge, chainAssetLabels, tokenExplorerURL} from './token-shared';

const observationTypes = [
    {label: 'All', value: ''},
    {label: 'Ave', value: 'ave'},
    {label: 'Chain state', value: 'chain_state'},
    {label: 'Wallet assets', value: 'wallet_asset_state'},
    {label: 'Simulations', value: 'simulation_result'},
    {label: 'Contract source', value: 'contract_code_source'}
];

const missing = (label = 'Not collected') => <Typography.Text type='secondary'>{label}</Typography.Text>;

const hasValue = (value: unknown) => value !== undefined && value !== null && value !== '';

const addressesEqual = (left?: string, right?: string) => Boolean(left && right && left.toLowerCase() === right.toLowerCase());

const formatCompact = (value?: string | number, currency = false) => {
    if (!hasValue(value)) {
        return missing();
    }
    const numeric = Number(value);
    if (!Number.isFinite(numeric)) {
        return String(value);
    }
    const formatted = new Intl.NumberFormat(undefined, {
        notation: Math.abs(numeric) >= 100000 ? 'compact' : 'standard',
        maximumFractionDigits: Math.abs(numeric) < 1 ? 8 : 2
    }).format(numeric);
    return currency ? `$${formatted}` : formatted;
};

const formatExact = (value?: string | number) =>
    hasValue(value) ? (
        <Tooltip title={String(value)}>
            <span className='project-detail__numeric'>{formatCompact(value)}</span>
        </Tooltip>
    ) : (
        missing()
    );

const formatTokenAmount = (value: string | undefined, decimals: number) => {
    if (!hasValue(value)) {
        return missing();
    }
    const raw = String(value);
    const match = raw.match(/^(-?)(\d+)$/);
    if (!match || decimals < 0) {
        return formatExact(value);
    }
    const digits = match[2].padStart(decimals + 1, '0');
    const whole = decimals === 0 ? digits : digits.slice(0, -decimals);
    const fraction = decimals === 0 ? '' : digits.slice(-decimals).replace(/0+$/, '');
    const display = `${match[1]}${whole}${fraction ? `.${fraction}` : ''}`;
    return (
        <Tooltip title={`${raw} base units`}>
            <span className='project-detail__numeric'>{formatCompact(display)}</span>
        </Tooltip>
    );
};

const prettyJSON = (value?: string) => {
    if (!value) {
        return '';
    }
    try {
        return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
        return value;
    }
};

const TimeValue = (props: {value?: string; unixSeconds?: number}) => {
    const raw = props.value || (props.unixSeconds ? new Date(props.unixSeconds * 1000).toISOString() : undefined);
    if (!raw) {
        return missing();
    }
    const parsed = new Date(raw);
    if (Number.isNaN(parsed.getTime())) {
        return <span>{raw}</span>;
    }
    return (
        <Tooltip title={parsed.toISOString()}>
            <time dateTime={parsed.toISOString()}>{parsed.toLocaleString()}</time>
        </Tooltip>
    );
};

const ExplorerValue = (props: {chainID?: number; kind: 'address' | 'tx'; value?: string}) => {
    if (!props.value) {
        return missing();
    }
    const url = tokenExplorerURL(props.chainID, props.kind, props.value);
    return (
        <span className='project-detail-identifier'>
            <TruncatedText value={props.value} copyable={true} singleLine={true} />
            {url && (
                <Tooltip title='Open in block explorer'>
                    <Typography.Link
                        className='project-detail-identifier__link'
                        href={url}
                        target='_blank'
                        rel='noreferrer'
                        aria-label={`Open ${props.value} in block explorer`}>
                        <ExportOutlined />
                    </Typography.Link>
                </Tooltip>
            )}
        </span>
    );
};

const BooleanState = (props: {value?: boolean; trueLabel?: string; falseLabel?: string; dangerWhenTrue?: boolean}) => {
    if (props.value === undefined) {
        return <StatusTag value='Unknown' />;
    }
    const danger = props.dangerWhenTrue ? props.value : false;
    const positive = props.dangerWhenTrue ? !props.value : props.value;
    return <StatusTag value={props.value ? props.trueLabel || 'Yes' : props.falseLabel || 'No'} positive={positive} negative={danger} />;
};

const ageLabel = (value?: string) => {
    if (!value) {
        return undefined;
    }
    const ageSeconds = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 1000));
    if (!Number.isFinite(ageSeconds)) {
        return undefined;
    }
    if (ageSeconds < 60) {
        return `${ageSeconds}s ago`;
    }
    if (ageSeconds < 3600) {
        return `${Math.floor(ageSeconds / 60)}m ago`;
    }
    if (ageSeconds < 86400) {
        return `${Math.floor(ageSeconds / 3600)}h ago`;
    }
    return `${Math.floor(ageSeconds / 86400)}d ago`;
};

const Summary = (props: {detail: TokenProjectDetail}) => {
    const project = props.detail.project;
    const ave = props.detail.ave?.token;
    const reportTime = props.detail.currentReport?.builtAt || props.detail.currentReport?.createdAt;
    const chainID = project?.chainID;
    return (
        <div className='project-detail-summary'>
            <Card className='project-detail-identity' size='small'>
                <div className='project-detail-identity__heading'>
                    <span className='project-detail-identity__symbol'>{project?.symbol || '?'}</span>
                    <div>
                        <Typography.Title level={3}>{project?.name || 'Unnamed token'}</Typography.Title>
                        <Space wrap={true}>
                            <ChainBadge chainID={chainID} />
                            <Tag>Project #{project?.projectID}</Tag>
                            <StatusTag value={props.detail.researchState?.status || 'Research not started'} />
                            <StatusTag value={props.detail.currentSelection?.outcome || 'Not selected'} />
                        </Space>
                    </div>
                </div>
                <KeyValueGrid
                    columns={2}
                    items={[
                        {label: 'Contract', value: <ExplorerValue chainID={chainID} kind='address' value={project?.contract} />},
                        {label: 'Deployment tx', value: <ExplorerValue chainID={chainID} kind='tx' value={project?.txHash} />},
                        {label: 'Deployer', value: <ExplorerValue chainID={chainID} kind='address' value={project?.txSender} />},
                        {label: 'Block', value: formatExact(project?.blockNumber)},
                        {label: 'Transaction index', value: formatExact(project?.txIndex)},
                        {label: 'Block time', value: <TimeValue unixSeconds={project?.blockTime} />},
                        {
                            label: 'Code hash',
                            value: project?.codeHash ? (
                                <Tooltip title={project.codeHash}>
                                    <Link className='project-detail-code-hash' to={`/token/contract-codes/${encodeURIComponent(project.codeHash)}`}>
                                        {project.codeHash}
                                    </Link>
                                </Tooltip>
                            ) : (
                                missing()
                            )
                        },
                        {
                            label: 'Report freshness',
                            value: reportTime ? (
                                <Tooltip title={reportTime}>
                                    <span>{ageLabel(reportTime)}</span>
                                </Tooltip>
                            ) : (
                                missing()
                            )
                        },
                        {label: 'Project created', value: <TimeValue value={project?.createdAt} />}
                    ]}
                />
            </Card>
            <div className='project-detail-kpis' aria-label='Current project metrics'>
                {[
                    ['Price', ave?.currentPriceUSD, true],
                    ['Market Cap', ave?.marketCap, true],
                    ['FDV', ave?.fdv, true],
                    ['TVL', ave?.tvl, true],
                    ['Holders', ave ? ave.holders : undefined, false],
                    ['Risk Score', ave?.riskScore, false]
                ].map(([label, value, currency]) => (
                    <Card className='project-detail-kpi' size='small' key={String(label)}>
                        <Typography.Text type='secondary'>{label}</Typography.Text>
                        <strong title={hasValue(value) ? String(value) : undefined}>
                            {hasValue(value) ? formatCompact(value as string | number, currency as boolean) : missing()}
                        </strong>
                    </Card>
                ))}
            </div>
        </div>
    );
};

const PairSummary = (props: {title: string; pair?: TokenChainPair; chainID?: number; baseDecimals?: number; quoteDecimals?: number}) => {
    if (!props.pair) {
        return (
            <Card size='small' title={props.title}>
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Pair not collected' />
            </Card>
        );
    }
    return (
        <Card size='small' title={props.title} extra={<BooleanState value={props.pair.isCreated} trueLabel='Created' falseLabel='Not created' />}>
            <KeyValueGrid
                columns={2}
                items={[
                    {label: 'Pair contract', value: <ExplorerValue chainID={props.chainID} kind='address' value={props.pair.pairContract} />},
                    {label: 'Base balance', value: formatTokenAmount(props.pair.baseBalance, props.baseDecimals ?? 18)},
                    {label: 'Quote balance', value: formatTokenAmount(props.pair.quoteBalance, props.quoteDecimals ?? 18)},
                    {label: 'USDT value', value: formatExact(props.pair.quoteUsdtValueInt || props.pair.quoteUsdtValue)},
                    {label: 'LP total supply', value: formatTokenAmount(props.pair.liquidity?.totalSupply, 18)},
                    {label: 'Locked liquidity', value: formatTokenAmount(props.pair.liquidity?.lockedLiquidity, 18)},
                    {label: 'Fee wallet LP', value: formatTokenAmount(props.pair.liquidity?.feeAddressHoldLiquidityBalance, 18)},
                    {
                        label: 'Fee wallet ratio',
                        value: hasValue(props.pair.liquidity?.feeAddressHoldLiquidityRatio) ? `${props.pair.liquidity?.feeAddressHoldLiquidityRatio}%` : missing()
                    },
                    {label: 'Last swap', value: <TimeValue value={props.pair.lastSwapAt} />},
                    {
                        label: 'Remove-liquidity risk',
                        value: <BooleanState value={props.pair.isRemoveLiquidity} trueLabel='Detected' falseLabel='Clear' dangerWhenTrue={true} />
                    },
                    {label: 'Mint risk', value: <BooleanState value={props.pair.isMint} trueLabel='Detected' falseLabel='Clear' dangerWhenTrue={true} />}
                ]}
            />
        </Card>
    );
};

const OverviewTab = (props: {detail: TokenProjectDetail}) => {
    const ave = props.detail.ave?.token;
    const chainID = props.detail.project?.chainID;
    const labels = chainAssetLabels(chainID);
    const wrappedPair = props.detail.chainState?.wethPair || (props.detail.project?.wethPair ? {pairContract: props.detail.project.wethPair, isCreated: true} : undefined);
    const stablePair = props.detail.chainState?.usdtPair || (props.detail.project?.usdtPair ? {pairContract: props.detail.project.usdtPair, isCreated: true} : undefined);
    const flags = ave
        ? [
              ['Mintable', ave.isMintableKnown ? ave.isMintable : undefined],
              ['Mint method', ave.hasMintMethod],
              ['LP not locked', ave.isLPNotLocked],
              ['Ownership not renounced', ave.hasNotRenounced],
              ['Not audited', ave.hasNotAudited],
              ['Not open source', ave.hasNotOpenSource],
              ['Blacklisted', ave.isInBlacklist],
              ['Honeypot', ave.isHoneypot]
          ]
        : [];
    const observations = new Map(props.detail.currentObservations.map(item => [item.dataType, item]));
    return (
        <div className='project-detail-tab'>
            <Section title='Risk flags'>
                {!ave ? (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Ave risk data has not been collected' />
                ) : (
                    <div className='project-risk-grid'>
                        {flags.map(([label, value]) => (
                            <div className='project-risk-flag' key={String(label)}>
                                <Typography.Text>{label}</Typography.Text>
                                <BooleanState value={value as boolean | undefined} trueLabel='Flagged' falseLabel='Clear' dangerWhenTrue={true} />
                            </div>
                        ))}
                    </div>
                )}
                {ave?.riskInfo && <Alert className='project-detail-inline-alert' type='warning' title='Risk information' description={ave.riskInfo} showIcon={true} />}
            </Section>
            <Section title='Research lifecycle'>
                <KeyValueGrid
                    columns={3}
                    items={[
                        {label: 'Status', value: props.detail.researchState?.status || missing('Not started')},
                        {label: 'Evidence revision', value: formatExact(props.detail.researchState?.evidenceRevision)},
                        {label: 'Current report', value: formatExact(props.detail.researchState?.currentReportRevision)},
                        {label: 'Selection outcome', value: props.detail.currentSelection?.outcome || missing('Not selected')},
                        {label: 'Last evaluated', value: <TimeValue value={props.detail.researchState?.lastEvaluatedAt} />},
                        {label: 'Expires', value: <TimeValue value={props.detail.researchState?.expiresAt} />}
                    ]}
                />
            </Section>
            <div className='project-detail-pair-grid'>
                <PairSummary title={`${labels.wrapped} pair`} pair={wrappedPair} chainID={chainID} baseDecimals={props.detail.project?.decimals} quoteDecimals={18} />
                <PairSummary
                    title={`${labels.stable} pair`}
                    pair={stablePair}
                    chainID={chainID}
                    baseDecimals={props.detail.project?.decimals}
                    quoteDecimals={labels.stableDecimals}
                />
            </div>
            <Section title='Collection freshness'>
                <div className='project-collection-grid'>
                    {props.detail.collectionSchedules.map(schedule => {
                        const observation = observations.get(schedule.dataType);
                        return (
                            <Card size='small' key={schedule.dataType} className='project-collection-card'>
                                <Flex justify='space-between' align='center' gap={8}>
                                    <Typography.Text strong={true}>{schedule.dataType}</Typography.Text>
                                    <StatusTag value={schedule.status} negative={schedule.status === 'failed' || Boolean(schedule.consecutiveFailures)} />
                                </Flex>
                                <Typography.Text type='secondary'>Last checked</Typography.Text>
                                <TimeValue value={observation?.lastCheckedAt || schedule.lastCheckedAt} />
                                <Typography.Text type='secondary'>Next attempt</Typography.Text>
                                <TimeValue value={schedule.status === 'active' ? schedule.nextRunAt : undefined} />
                                {schedule.lastError && <Typography.Text type='danger'>{schedule.lastError}</Typography.Text>}
                            </Card>
                        );
                    })}
                    {props.detail.collectionSchedules.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No collection schedules' />}
                </div>
            </Section>
        </div>
    );
};

const MarketTab = (props: {projectID: number; detail: TokenProjectDetail; refreshVersion: number}) => {
    const [range, setRange] = React.useState('24h');
    const [metric, setMetric] = React.useState('price_usd');
    const [avePairSelection, setAvePairSelection] = React.useState<{projectID: number; kind: 'weth' | 'usdt'}>({projectID: props.projectID, kind: 'weth'});
    const trends = useAsyncData(() => services.tokenapi.listProjectTrends(props.projectID, range), [props.projectID, range, props.refreshVersion]);
    const series = trends.data?.series || [];
    React.useEffect(() => {
        if (series.length > 0 && !series.some(item => item.key === metric)) {
            setMetric(series[0].key || '');
        }
    }, [metric, series]);
    const selectedSeries = series.find(item => item.key === metric);
    const ave = props.detail.ave;
    const token = ave?.token;
    const chainID = props.detail.project?.chainID;
    const labels = chainAssetLabels(chainID);
    const wrappedPair = props.detail.chainState?.wethPair || (props.detail.project?.wethPair ? {pairContract: props.detail.project.wethPair, isCreated: true} : undefined);
    const stablePair = props.detail.chainState?.usdtPair || (props.detail.project?.usdtPair ? {pairContract: props.detail.project.usdtPair, isCreated: true} : undefined);
    const avePairs = ave?.pairs || [];
    const wethAvePair = avePairs.find(item => addressesEqual(item.pair, props.detail.project?.wethPair));
    const usdtAvePair = avePairs.find(item => addressesEqual(item.pair, props.detail.project?.usdtPair));
    const requestedAvePairKind = avePairSelection.projectID === props.projectID ? avePairSelection.kind : 'weth';
    const requestedAvePair = requestedAvePairKind === 'weth' ? wethAvePair : usdtAvePair;
    const selectedAvePairKind = requestedAvePair ? requestedAvePairKind : wethAvePair ? 'weth' : usdtAvePair ? 'usdt' : 'weth';
    const selectedAvePair = selectedAvePairKind === 'weth' ? wethAvePair : usdtAvePair;
    React.useEffect(() => {
        if (avePairSelection.projectID !== props.projectID || avePairSelection.kind !== selectedAvePairKind) {
            setAvePairSelection({projectID: props.projectID, kind: selectedAvePairKind});
        }
    }, [avePairSelection, props.projectID, selectedAvePairKind]);
    const pairTokenValue = (pair: TokenAvePair, tokenIndex: 0 | 1) => {
        const symbol = tokenIndex === 0 ? pair.token0Symbol : pair.token1Symbol;
        const address = tokenIndex === 0 ? pair.token0Address : pair.token1Address;
        return (
            <Space orientation='vertical' size={0}>
                <span>{symbol || '-'}</span>
                <ExplorerValue chainID={chainID} kind='address' value={address} />
            </Space>
        );
    };
    return (
        <div className='project-detail-tab'>
            <Section title='Market snapshot'>
                {!token ? (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Ave market data has not been collected' />
                ) : (
                    <KeyValueGrid
                        columns={4}
                        items={[
                            {label: 'Price USD', value: formatExact(token.currentPriceUSD)},
                            {label: `Price ${labels.native}`, value: formatExact(token.currentPriceETH)},
                            {label: 'Market Cap', value: formatExact(token.marketCap)},
                            {label: 'FDV', value: formatExact(token.fdv)},
                            {label: 'TVL', value: formatExact(token.tvl)},
                            {label: 'Main pair TVL', value: formatExact(token.mainPairTVL)},
                            {label: 'Holders', value: formatExact(token.holders)},
                            {label: 'Risk level', value: formatExact(token.riskLevel)},
                            {label: 'Risk score', value: formatExact(token.riskScore)},
                            {label: 'Audited', value: <BooleanState value={ave?.isAudited} />},
                            {label: 'Total supply', value: formatExact(token.totalSupply)},
                            {label: 'Decimals', value: formatExact(token.decimals)},
                            {label: 'Launch time', value: <TimeValue value={token.launchAt} />},
                            {label: 'Updated', value: <TimeValue value={token.updatedAt} />}
                        ]}
                    />
                )}
            </Section>
            <div className='project-detail-pair-grid'>
                <PairSummary
                    title={`${labels.wrapped} pair on-chain state`}
                    pair={wrappedPair}
                    chainID={chainID}
                    baseDecimals={props.detail.project?.decimals}
                    quoteDecimals={18}
                />
                <PairSummary
                    title={`${labels.stable} pair on-chain state`}
                    pair={stablePair}
                    chainID={chainID}
                    baseDecimals={props.detail.project?.decimals}
                    quoteDecimals={labels.stableDecimals}
                />
            </div>
            <Section
                title='Market trends'
                extra={
                    <Space wrap={true}>
                        <Select aria-label='Trend metric' value={metric} options={series.map(item => ({value: item.key, label: item.label}))} onChange={setMetric} />
                        <ChoiceGroup<string> ariaLabel='Trend range' value={range} options={['1h', '6h', '24h', '7d'].map(value => ({label: value, value}))} onChange={setRange} />
                    </Space>
                }>
                {trends.error && <Alert type='error' title='Trend history unavailable' description={trends.error.message} showIcon={true} />}
                <div aria-busy={trends.loading || undefined}>
                    <ProjectTrendChart series={selectedSeries} />
                </div>
            </Section>
            <Section
                title='AVE key pair data'
                extra={
                    <ChoiceGroup<'weth' | 'usdt'>
                        ariaLabel='AVE key pair'
                        value={selectedAvePairKind}
                        options={[
                            {label: `${labels.wrapped} pair`, value: 'weth', disabled: !wethAvePair},
                            {label: `${labels.stable} pair`, value: 'usdt', disabled: !usdtAvePair}
                        ]}
                        onChange={kind => setAvePairSelection({projectID: props.projectID, kind})}
                    />
                }>
                {selectedAvePair ? (
                    <KeyValueGrid
                        columns={3}
                        items={[
                            {label: 'Pair contract', value: <ExplorerValue chainID={chainID} kind='address' value={selectedAvePair.pair} />},
                            {label: 'AMM', value: selectedAvePair.amm},
                            {label: 'Token 0', value: pairTokenValue(selectedAvePair, 0)},
                            {label: 'Reserve 0', value: formatExact(selectedAvePair.reserve0)},
                            {label: 'Token 1', value: pairTokenValue(selectedAvePair, 1)},
                            {label: 'Reserve 1', value: formatExact(selectedAvePair.reserve1)},
                            {label: 'Volume USD', value: formatExact(selectedAvePair.volumeUSD)},
                            {label: 'Market Cap', value: formatExact(selectedAvePair.marketCap)},
                            {label: 'FDV', value: formatExact(selectedAvePair.fdv)},
                            {label: 'Integrity', value: <BooleanState value={selectedAvePair.isFake} trueLabel='Fake' falseLabel='Verified' dangerWhenTrue={true} />},
                            {label: 'Created', value: <TimeValue value={selectedAvePair.createdAt} />},
                            {label: 'Updated', value: <TimeValue value={selectedAvePair.updatedAt} />}
                        ]}
                    />
                ) : (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={`Ave did not return ${labels.wrapped} or ${labels.stable} pair data`} />
                )}
            </Section>
        </div>
    );
};

interface WalletView {
    wallet: string;
    roles: string[];
    recipientRank?: number;
    recipientRatioBPS?: number;
    wethBalance?: string;
    usdtBalance?: string;
    nativeBalance?: string;
    usdtValue?: string;
    transactionCount?: number;
    simulation?: TokenSimulationResult;
}

const walletRows = (detail: TokenProjectDetail): WalletView[] => {
    const rows = new Map<string, WalletView>();
    const get = (wallet?: string) => {
        const key = (wallet || '').toLowerCase();
        if (!key) {
            return undefined;
        }
        if (!rows.has(key)) {
            rows.set(key, {wallet: wallet || '', roles: []});
        }
        return rows.get(key);
    };
    detail.relatedWallets.forEach(item => {
        const row = get(item.wallet);
        if (row && item.role && !row.roles.includes(item.role)) {
            row.roles.push(item.role);
        }
    });
    detail.initialRecipients.forEach(item => {
        const row = get(item.wallet);
        if (row) {
            row.recipientRank = item.rankIndex;
            row.recipientRatioBPS = item.ratioBPS;
            if (!row.roles.includes('initial_recipient')) {
                row.roles.push('initial_recipient');
            }
        }
    });
    detail.walletAssets.forEach(item => Object.assign(get(item.wallet) || {}, item));
    detail.simulations.forEach(item => {
        const row = get(item.wallet);
        if (row) {
            row.simulation = item;
        }
    });
    detail.walletTransactionCounts.forEach(item => {
        const row = get(item.wallet);
        if (row) {
            row.transactionCount = item.transactionCount;
        }
    });
    return [...rows.values()].sort((left, right) => (left.recipientRank ?? Number.MAX_SAFE_INTEGER) - (right.recipientRank ?? Number.MAX_SAFE_INTEGER));
};

const WalletsTab = (props: {detail: TokenProjectDetail}) => {
    const rows = walletRows(props.detail);
    const chainID = props.detail.project?.chainID;
    const labels = chainAssetLabels(chainID);
    const simulationColumn = (title: string, field: keyof TokenSimulationResult) => ({
        title,
        width: 142,
        render: (item: WalletView) => (
            <BooleanState value={item.simulation ? (item.simulation[field] as boolean) : undefined} trueLabel='Possible' falseLabel='Blocked' dangerWhenTrue={true} />
        )
    });
    const columns: ColumnsType<WalletView> = [
        {title: 'Wallet', fixed: 'left', width: 260, render: item => <ExplorerValue chainID={chainID} kind='address' value={item.wallet} />},
        {
            title: 'Roles',
            width: 220,
            render: (item: WalletView) => (
                <Space wrap={true}>
                    {item.roles.map((role: string) => (
                        <Tag key={role}>{role}</Tag>
                    ))}
                </Space>
            )
        },
        {title: 'Recipient rank', render: item => formatExact(item.recipientRank)},
        {title: 'Share', render: item => (hasValue(item.recipientRatioBPS) ? `${((item.recipientRatioBPS || 0) / 100).toFixed(2)}%` : missing())},
        {title: labels.native, render: item => formatTokenAmount(item.nativeBalance, 18)},
        {title: labels.wrapped, render: item => formatTokenAmount(item.wethBalance, 18)},
        {title: labels.stable, render: item => formatTokenAmount(item.usdtBalance, labels.stableDecimals)},
        {title: 'USDT value', render: item => formatTokenAmount(item.usdtValue, labels.stableDecimals)},
        {title: 'Transactions', render: item => formatExact(item.transactionCount)},
        simulationColumn('From dead', 'canMintFromDeadViaTransferFrom'),
        simulationColumn('From zero', 'canMintFromZeroViaTransferFrom'),
        simulationColumn(`From ${labels.wrapped} pair`, 'canMintFromWethPairViaTransferFrom'),
        simulationColumn(`From ${labels.stable} pair`, 'canMintFromUsdtPairViaTransferFrom'),
        simulationColumn(`Transfer to ${labels.wrapped}`, 'canMintViaTransferToWethPair'),
        simulationColumn(`Transfer to ${labels.stable}`, 'canMintViaTransferToUsdtPair')
    ];
    return (
        <div className='project-detail-tab'>
            <Section title='Associated wallets'>
                {rows.length === 0 ? (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No associated wallet history' />
                ) : (
                    <ResourceTable rowKey='wallet' items={rows} columns={columns} scrollX={2500} />
                )}
            </Section>
        </div>
    );
};

const TransactionsTab = (props: {projectID: number; detail: TokenProjectDetail; refreshVersion: number}) => {
    const [page, setPage] = React.useState(1);
    const [pageSize, setPageSize] = React.useState(20);
    const [wallet, setWallet] = React.useState('');
    const [receiptStatus, setReceiptStatus] = React.useState('');
    const [methodID, setMethodID] = React.useState('');
    const data = useAsyncData(
        () => services.tokenapi.listProjectWalletNormalTransactions(props.projectID, {wallet, receiptStatus, methodID, page, pageSize}),
        [props.projectID, wallet, receiptStatus, methodID, page, pageSize, props.refreshVersion]
    );
    const chainID = props.detail.project?.chainID;
    const walletOptions = walletRows(props.detail).map(item => ({label: item.wallet, value: item.wallet}));
    const columns: ColumnsType<TokenWalletNormalTransaction> = [
        {title: 'Block', width: 120, render: item => formatExact(item.blockNumber)},
        {title: 'Index', width: 80, dataIndex: 'transactionIndex'},
        {title: 'Time', width: 190, render: item => <TimeValue value={item.blockTimestamp} />},
        {title: 'Transaction', width: 260, render: item => <ExplorerValue chainID={chainID} kind='tx' value={item.transactionHash} />},
        {title: 'Wallet', width: 230, render: item => <ExplorerValue chainID={chainID} kind='address' value={item.wallet} />},
        {title: 'From', width: 230, render: item => <ExplorerValue chainID={chainID} kind='address' value={item.fromAddress} />},
        {title: 'To', width: 230, render: item => <ExplorerValue chainID={chainID} kind='address' value={item.toAddress} />},
        {title: 'Method', width: 150, render: item => item.functionName || item.methodID || missing('Unknown')},
        {title: `Value (${chainAssetLabels(chainID).native})`, width: 160, render: item => formatTokenAmount(item.value, 18)},
        {
            title: 'Receipt',
            width: 110,
            render: item => (
                <StatusTag value={item.receiptStatus || 'Unknown'} positive={item.receiptStatus === 'success'} negative={item.receiptStatus === 'failed' || item.isError} />
            )
        }
    ];
    return (
        <div className='project-detail-tab'>
            <Section
                title='Pre-deployment wallet transactions'
                extra={
                    <Space wrap={true}>
                        <Select
                            allowClear={true}
                            showSearch={true}
                            optionFilterProp='label'
                            aria-label='Filter wallet'
                            placeholder='All wallets'
                            value={wallet || undefined}
                            options={walletOptions}
                            onChange={value => {
                                setWallet(value || '');
                                setPage(1);
                            }}
                            style={{width: 220}}
                        />
                        <ChoiceGroup
                            ariaLabel='Receipt status'
                            value={receiptStatus}
                            options={[
                                {label: 'All', value: ''},
                                {label: 'Success', value: 'success'},
                                {label: 'Failed', value: 'failed'},
                                {label: 'Unspecified', value: 'unspecified'}
                            ]}
                            onChange={value => {
                                setReceiptStatus(value);
                                setPage(1);
                            }}
                        />
                        <SearchBar
                            value={methodID}
                            placeholder='Method ID'
                            onChange={value => {
                                setMethodID(value);
                                setPage(1);
                            }}
                        />
                    </Space>
                }>
                {data.error && <Alert type='error' title='Transactions unavailable' description={data.error.message} showIcon={true} />}
                <Table<TokenWalletNormalTransaction>
                    className='resource-table'
                    size='small'
                    rowKey={item => `${item.wallet}-${item.transactionHash}`}
                    dataSource={data.data?.items || []}
                    columns={columns}
                    loading={data.loading}
                    locale={{emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No pre-deployment transactions collected' />}}
                    pagination={{
                        current: page,
                        pageSize,
                        total: data.data?.total,
                        showSizeChanger: true,
                        onChange: (nextPage, nextPageSize) => {
                            setPage(nextPage);
                            setPageSize(nextPageSize);
                        }
                    }}
                    scroll={{x: 1900}}
                    expandable={{
                        expandedRowRender: item => (
                            <KeyValueGrid
                                columns={3}
                                items={[
                                    {label: 'Input', value: <Typography.Text copyable={true}>{item.input || '-'}</Typography.Text>},
                                    {label: 'Function', value: item.functionName || missing('Unknown')},
                                    {label: 'Method ID', value: item.methodID || missing('Unknown')},
                                    {label: 'Gas limit', value: formatExact(item.gas)},
                                    {label: 'Gas used', value: formatExact(item.gasUsed)},
                                    {label: 'Gas price', value: formatExact(item.gasPrice)},
                                    {label: 'Nonce', value: formatExact(item.nonce)},
                                    {label: 'Collected', value: <TimeValue value={item.collectedAt} />}
                                ]}
                            />
                        )
                    }}
                />
            </Section>
        </div>
    );
};

const ContractTab = (props: {detail: TokenProjectDetail; refreshVersion: number}) => {
    const project = props.detail.project;
    const codeHash = project?.codeHash || '';
    const source = useAsyncData(() => services.tokenapi.getContractCode(codeHash), [codeHash, props.refreshVersion]);
    const ave = props.detail.ave?.token;
    return (
        <div className='project-detail-tab'>
            <Section title='ERC-20 metadata'>
                <KeyValueGrid
                    columns={3}
                    items={[
                        {label: 'Name', value: props.detail.chainState?.token?.name || project?.name || missing()},
                        {label: 'Symbol', value: props.detail.chainState?.token?.symbol || project?.symbol || missing()},
                        {label: 'Decimals', value: formatExact(props.detail.chainState?.token?.decimals ?? project?.decimals)},
                        {label: 'Total supply', value: formatExact(props.detail.chainState?.token?.totalSupply || project?.totalSupply)},
                        {label: 'Valid ERC-20', value: <BooleanState value={props.detail.chainState ? props.detail.chainState.isValidERC20 : undefined} />},
                        {label: 'Contract', value: <ExplorerValue chainID={project?.chainID} kind='address' value={project?.contract} />},
                        {
                            label: 'Code hash',
                            value: codeHash ? <Link to={`/token/contract-codes/${encodeURIComponent(codeHash)}`}>{codeHash}</Link> : missing()
                        },
                        {label: 'Source available', value: <BooleanState value={props.detail.contractSource?.sourceAvailable} />}
                    ]}
                />
            </Section>
            <Section title='Contract risk markers'>
                {!ave ? (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Contract risk data has not been collected' />
                ) : (
                    <div className='project-risk-grid'>
                        {[
                            ['Mintable', ave.isMintableKnown ? ave.isMintable : undefined],
                            ['Mint method', ave.hasMintMethod],
                            ['LP not locked', ave.isLPNotLocked],
                            ['Ownership not renounced', ave.hasNotRenounced],
                            ['Not audited', ave.hasNotAudited],
                            ['Not open source', ave.hasNotOpenSource],
                            ['Blacklisted', ave.isInBlacklist],
                            ['Honeypot', ave.isHoneypot]
                        ].map(([label, value]) => (
                            <div className='project-risk-flag' key={String(label)}>
                                <Typography.Text>{label}</Typography.Text>
                                <BooleanState value={value as boolean | undefined} trueLabel='Flagged' falseLabel='Clear' dangerWhenTrue={true} />
                            </div>
                        ))}
                    </div>
                )}
            </Section>
            <Section
                title='Source code'
                extra={
                    codeHash ? (
                        <Link to={`/token/contract-codes/${encodeURIComponent(codeHash)}`}>
                            <CodeOutlined /> Code Hash details
                        </Link>
                    ) : undefined
                }>
                {source.error && <Alert type='error' title='Source code unavailable' description={source.error.message} showIcon={true} />}
                {source.loading && !source.data ? (
                    <Typography.Text type='secondary'>Loading source code…</Typography.Text>
                ) : source.data?.sourceCode ? (
                    <pre className='code-block project-contract-source'>{source.data.sourceCode}</pre>
                ) : (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No verified source code collected' />
                )}
            </Section>
        </div>
    );
};

const JSONDrawer = (props: {title?: string; value?: string; onClose: () => void}) => {
    const formatted = prettyJSON(props.value);
    return (
        <Drawer
            title={props.title || 'Raw JSON'}
            width='min(760px, 80vw)'
            open={Boolean(props.title)}
            onClose={props.onClose}
            extra={
                formatted ? (
                    <Button icon={<CopyOutlined />} onClick={() => navigator.clipboard.writeText(formatted)}>
                        Copy
                    </Button>
                ) : undefined
            }>
            <pre className='code-block project-json-viewer'>{formatted}</pre>
        </Drawer>
    );
};

const ResearchTab = (props: {projectID: number; detail: TokenProjectDetail; refreshVersion: number}) => {
    const [observationPage, setObservationPage] = React.useState(1);
    const [observationPageSize, setObservationPageSize] = React.useState(20);
    const [dataType, setDataType] = React.useState('');
    const [reportPage, setReportPage] = React.useState(1);
    const [reportPageSize, setReportPageSize] = React.useState(20);
    const [selectionPage, setSelectionPage] = React.useState(1);
    const [selectionPageSize, setSelectionPageSize] = React.useState(20);
    const [taskPage, setTaskPage] = React.useState(1);
    const [taskPageSize, setTaskPageSize] = React.useState(20);
    const [drawer, setDrawer] = React.useState<{title: string; value: string}>();
    const observations = useAsyncData(
        () => services.tokenapi.listProjectObservations(props.projectID, {dataType, page: observationPage, pageSize: observationPageSize}),
        [props.projectID, dataType, observationPage, observationPageSize, props.refreshVersion]
    );
    const reports = useAsyncData(
        () => services.tokenapi.listReportRevisions({projectID: props.projectID, page: reportPage, pageSize: reportPageSize}),
        [props.projectID, reportPage, reportPageSize, props.refreshVersion]
    );
    const selections = useAsyncData(
        () => services.tokenapi.listSelections({projectID: props.projectID, page: selectionPage, pageSize: selectionPageSize}),
        [props.projectID, selectionPage, selectionPageSize, props.refreshVersion]
    );
    const tasks = useAsyncData(
        () => services.tokenapi.listCollectionTasks({projectID: props.projectID, page: taskPage, pageSize: taskPageSize}),
        [props.projectID, taskPage, taskPageSize, props.refreshVersion]
    );
    const scheduleColumns: ColumnsType<TokenCollectionSchedule> = [
        {title: 'Data type', dataIndex: 'dataType'},
        {title: 'Status', render: item => <StatusTag value={item.status} negative={item.status === 'failed' || Boolean(item.consecutiveFailures)} />},
        {title: 'Retry interval', render: item => (hasValue(item.retryIntervalSecs) ? `${item.retryIntervalSecs}s` : missing())},
        {title: 'Revision', render: item => formatExact(item.latestTaskRevision)},
        {title: 'Failures', render: item => formatExact(item.consecutiveFailures)},
        {title: 'Last checked', render: item => <TimeValue value={item.lastCheckedAt} />},
        {title: 'Next attempt', render: item => <TimeValue value={item.status === 'active' ? item.nextRunAt : undefined} />},
        {title: 'Last error', render: item => (item.lastError ? <Typography.Text type='danger'>{item.lastError}</Typography.Text> : '-')}
    ];
    const observationColumns: ColumnsType<TokenProjectObservation> = [
        {title: 'ID', dataIndex: 'observationID'},
        {title: 'Data type', dataIndex: 'dataType'},
        {title: 'Schema', dataIndex: 'schemaVersion'},
        {title: 'Block', render: item => formatExact(item.blockNumber)},
        {title: 'Observed', render: item => <TimeValue value={item.observedAt} />},
        {title: 'Last checked', render: item => <TimeValue value={item.lastCheckedAt} />},
        {title: 'Content hash', render: item => <TruncatedText value={item.contentHash} copyable={true} />},
        {
            title: 'Payload',
            render: item => (
                <Button size='small' onClick={() => setDrawer({title: `Observation #${item.observationID} payload`, value: item.payloadJSON || ''})}>
                    View JSON
                </Button>
            )
        }
    ];
    const reportColumns: ColumnsType<TokenReportRevision> = [
        {title: 'Revision', dataIndex: 'revision'},
        {title: 'Completeness', render: item => <StatusTag value={item.completenessStatus} />},
        {title: 'Block', render: item => formatExact(item.observedBlockNumber)},
        {title: 'Content hash', render: item => <TruncatedText value={item.contentHash} copyable={true} />},
        {title: 'Built', render: item => <TimeValue value={item.builtAt} />},
        {
            title: 'Evidence',
            render: item => (
                <Button size='small' onClick={() => setDrawer({title: `Report r${item.revision} evidence`, value: item.evidenceJSON || ''})}>
                    View JSON
                </Button>
            )
        },
        {
            title: 'Report',
            render: item => (
                <Button size='small' onClick={() => setDrawer({title: `Report r${item.revision}`, value: item.reportJSON || ''})}>
                    View JSON
                </Button>
            )
        }
    ];
    const taskColumns: ColumnsType<TokenCollectionTask> = [
        {title: 'Task', dataIndex: 'taskID'},
        {title: 'Data type', dataIndex: 'dataType'},
        {title: 'Status', render: item => <StatusTag value={item.status} positive={item.status === 'succeeded'} negative={item.status === 'failed'} />},
        {title: 'Revision', dataIndex: 'revision'},
        {title: 'Attempts', dataIndex: 'attempts'},
        {title: 'Available', render: item => <TimeValue value={item.availableAt} />},
        {title: 'Updated', render: item => <TimeValue value={item.updatedAt} />},
        {title: 'Last error', render: item => (item.lastError ? <Typography.Text type='danger'>{item.lastError}</Typography.Text> : '-')}
    ];
    const sectionError = (title: string, error?: Error) => (error ? <Alert type='error' title={title} description={error.message} showIcon={true} /> : null);
    return (
        <div className='project-detail-tab'>
            <Section title='Collection schedules'>
                <ResourceTable rowKey={item => item.dataType || String(item.projectID)} items={props.detail.collectionSchedules} columns={scheduleColumns} scrollX={1400} />
            </Section>
            <Section title='Selection timeline'>
                {sectionError('Selection history unavailable', selections.error)}
                {selections.data?.items.length ? (
                    <Timeline
                        items={selections.data.items.map((selection: TokenSelection) => ({
                            color: selection.outcome === 'selected' ? 'green' : selection.outcome === 'rejected' ? 'red' : 'gray',
                            children: (
                                <div className='project-selection-event'>
                                    <Space wrap={true}>
                                        <StatusTag value={selection.outcome} positive={selection.outcome === 'selected'} negative={selection.outcome === 'rejected'} />
                                        <Typography.Text strong={true}>Report r{selection.reportRevision}</Typography.Text>
                                        <TimeValue value={selection.decidedAt} />
                                    </Space>
                                    <Typography.Paragraph>{selection.reasonDetail || 'No reason detail'}</Typography.Paragraph>
                                    <Space wrap={true}>
                                        {selection.reasonCodes.map(code => (
                                            <Tag key={code}>{code}</Tag>
                                        ))}
                                    </Space>
                                </div>
                            )
                        }))}
                    />
                ) : (
                    !selections.loading && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No selection history' />
                )}
                {selections.data && selections.data.total > selectionPageSize && (
                    <Pagination
                        total={selections.data.total}
                        current={selectionPage}
                        pageSize={selectionPageSize}
                        showSizeChanger={true}
                        onChange={(nextPage, nextPageSize) => {
                            setSelectionPage(nextPage);
                            setSelectionPageSize(nextPageSize);
                        }}
                    />
                )}
            </Section>
            <Section
                title='Observation history'
                extra={
                    <ChoiceGroup
                        ariaLabel='Observation data type'
                        value={dataType}
                        options={observationTypes}
                        onChange={value => {
                            setDataType(value);
                            setObservationPage(1);
                        }}
                    />
                }>
                {sectionError('Observation history unavailable', observations.error)}
                <ResourceTable
                    rowKey={item => item.observationID || `${item.dataType}-${item.observedAt}`}
                    items={observations.data?.items || []}
                    columns={observationColumns}
                    loading={observations.loading}
                    total={observations.data?.total}
                    page={observationPage}
                    pageSize={observationPageSize}
                    onPageChange={(nextPage, nextPageSize) => {
                        setObservationPage(nextPage);
                        setObservationPageSize(nextPageSize);
                    }}
                    scrollX={1500}
                />
            </Section>
            <Section title='Report revisions'>
                {sectionError('Report history unavailable', reports.error)}
                <ResourceTable
                    rowKey={item => item.reportRevisionID || `${item.revision}-${item.createdAt}`}
                    items={reports.data?.items || []}
                    columns={reportColumns}
                    loading={reports.loading}
                    total={reports.data?.total}
                    page={reportPage}
                    pageSize={reportPageSize}
                    onPageChange={(nextPage, nextPageSize) => {
                        setReportPage(nextPage);
                        setReportPageSize(nextPageSize);
                    }}
                    scrollX={1300}
                />
            </Section>
            <Section title='Collection task history'>
                {sectionError('Task history unavailable', tasks.error)}
                <ResourceTable
                    rowKey={item => item.taskID || `${item.dataType}-${item.revision}`}
                    items={tasks.data?.items || []}
                    columns={taskColumns}
                    loading={tasks.loading}
                    total={tasks.data?.total}
                    page={taskPage}
                    pageSize={taskPageSize}
                    onPageChange={(nextPage, nextPageSize) => {
                        setTaskPage(nextPage);
                        setTaskPageSize(nextPageSize);
                    }}
                    scrollX={1400}
                />
            </Section>
            <JSONDrawer title={drawer?.title} value={drawer?.value} onClose={() => setDrawer(undefined)} />
        </div>
    );
};

export const ProjectDetailPage = () => {
    const params = useParams();
    const navigate = useNavigate();
    const projectID = Number(params.projectID);
    const [activeTab, setActiveTab] = React.useState('overview');
    const [refreshVersion, setRefreshVersion] = React.useState(0);
    const detail = useAsyncData(() => services.tokenapi.getProjectDetail(projectID), [projectID]);
    const detailReloadRef = React.useRef(detail.reload);
    detailReloadRef.current = detail.reload;

    React.useEffect(() => {
        const timer = window.setInterval(() => {
            if (document.visibilityState === 'visible') {
                detailReloadRef.current();
            }
        }, 30000);
        const refreshWhenVisible = () => {
            if (document.visibilityState === 'visible') {
                detailReloadRef.current();
            }
        };
        document.addEventListener('visibilitychange', refreshWhenVisible);
        return () => {
            window.clearInterval(timer);
            document.removeEventListener('visibilitychange', refreshWhenVisible);
        };
    }, []);

    const refresh = () => {
        detail.reload();
        setRefreshVersion(current => current + 1);
    };
    const project = detail.data?.project;
    const activeRefreshVersion = (key: string) => (activeTab === key ? refreshVersion : 0);

    return (
        <AppPage
            title={project ? `${project.symbol || project.name || 'Token'} project` : 'Token project'}
            subtitle={project ? `${project.name || 'Unnamed token'} · Project #${project.projectID}` : `Project #${params.projectID || '-'}`}
            loading={detail.loading}
            error={detail.error}
            onRefresh={refresh}
            extra={<Button onClick={() => navigate('/token/projects')}>Back to projects</Button>}>
            {!Number.isInteger(projectID) || projectID <= 0 ? (
                <Alert type='error' title='Invalid project ID' description='The project ID must be a positive integer.' showIcon={true} />
            ) : !detail.loading && !detail.error && !detail.data ? (
                <Empty className='project-detail-not-found' image={Empty.PRESENTED_IMAGE_SIMPLE} description='Project not found'>
                    <Button type='primary' onClick={() => navigate('/token/projects')}>
                        Return to projects
                    </Button>
                </Empty>
            ) : detail.data ? (
                <div className='project-detail'>
                    <Summary detail={detail.data} />
                    <Tabs
                        className='project-detail-tabs'
                        activeKey={activeTab}
                        onChange={setActiveTab}
                        items={[
                            {key: 'overview', label: 'Overview', children: <OverviewTab detail={detail.data} />},
                            {
                                key: 'market',
                                label: 'Market & Liquidity',
                                children: <MarketTab projectID={projectID} detail={detail.data} refreshVersion={activeRefreshVersion('market')} />
                            },
                            {key: 'wallets', label: 'Wallets', children: <WalletsTab detail={detail.data} />},
                            {
                                key: 'transactions',
                                label: `Transactions (${detail.data.transactionCount || 0})`,
                                children: <TransactionsTab projectID={projectID} detail={detail.data} refreshVersion={activeRefreshVersion('transactions')} />
                            },
                            {
                                key: 'contract',
                                label: 'Contract',
                                children: <ContractTab detail={detail.data} refreshVersion={activeRefreshVersion('contract')} />
                            },
                            {
                                key: 'research',
                                label: 'Research',
                                children: <ResearchTab projectID={projectID} detail={detail.data} refreshVersion={activeRefreshVersion('research')} />
                            }
                        ]}
                    />
                </div>
            ) : (
                <Card>
                    <Typography.Text type='secondary'>Loading project snapshot…</Typography.Text>
                </Card>
            )}
        </AppPage>
    );
};
