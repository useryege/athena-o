import {FilterOutlined} from '@ant-design/icons';
import {Badge, Button, Card, Progress, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useSearchParams} from 'react-router-dom';
import {AppPage, KeyValueGrid, ResourceTable, StatusTag, useCachedAsyncData} from '../../components';
import {AccountDataModule} from '../../shared/access-modules';
import {formatBeijingDateTime, formatBeijingUnixSeconds} from '../../shared/format';
import {DEFAULT_PAGE_SIZE, PAGE_SIZE_OPTIONS} from '../../shared/pagination';
import {TokenProjectListItem, TokenProjectPairProfileSummary} from '../../shared/services/token-service';
import {memberServices as services} from '../services';
import {ProjectDetailLink, useRestoreProjectsScroll} from './project-navigation';
import {
    collectionStatuses,
    emptyProjectsFilterState,
    generalFilterCount,
    profilePairFilterCount,
    ProfilePairFilterState,
    profilePairSignalStates,
    profileStates,
    ProjectsFiltersModal,
    ProjectsFilterState
} from './projects-filters-modal';
import {ChainBadge, TokenLogo} from './token-shared';

const PROJECTS_LIST_STALE_TIME_MS = 30_000;
const PROJECTS_POLL_INTERVAL_MS = 30_000;
const PROJECTS_TABLE_WIDTH = 2460;

const positiveIntegerParam = (value: string | null) => {
    const parsed = Number(value);
    return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : undefined;
};
const enumParam = (value: string | null, allowed: readonly string[]) => (value && allowed.includes(value) ? value : '');
const enumListParam = <T extends string>(value: string | null, allowed: readonly T[]) => {
    const selected = new Set((value || '').split(',').map(item => item.trim()));
    return allowed.filter(item => selected.has(item));
};
const unsignedIntegerParam = (value: string | null) => {
    const normalized = (value || '').trim();
    return /^\d+$/.test(normalized) ? BigInt(normalized).toString() : '';
};

interface ProjectsQueryState extends ProjectsFilterState {
    page: number;
    pageSize: number;
}

const pairFilterFromParams = (params: URLSearchParams): ProfilePairFilterState => ({
    balanceSupply: enumListParam(params.get('pairBalanceSupply'), profilePairSignalStates),
    minimumLP: enumListParam(params.get('pairMinimumLP'), profilePairSignalStates),
    feeLPShare: enumListParam(params.get('pairFeeLPShare'), profilePairSignalStates),
    quoteMin: unsignedIntegerParam(params.get('pairQuoteMin')),
    quoteMax: unsignedIntegerParam(params.get('pairQuoteMax'))
});

const serializePairFilter = (params: URLSearchParams, filter: ProfilePairFilterState) => {
    if (filter.balanceSupply.length) params.set('pairBalanceSupply', profilePairSignalStates.filter(value => filter.balanceSupply.includes(value)).join(','));
    if (filter.minimumLP.length) params.set('pairMinimumLP', profilePairSignalStates.filter(value => filter.minimumLP.includes(value)).join(','));
    if (filter.feeLPShare.length) params.set('pairFeeLPShare', profilePairSignalStates.filter(value => filter.feeLPShare.includes(value)).join(','));
    if (filter.quoteMin) params.set('pairQuoteMin', filter.quoteMin);
    if (filter.quoteMax) params.set('pairQuoteMax', filter.quoteMax);
};

const serializeQueryState = (state: ProjectsQueryState) => {
    const next = new URLSearchParams();
    if (state.contract) next.set('contract', state.contract);
    if (state.codeHash) next.set('codeHash', state.codeHash);
    if (state.collectionStatus) next.set('collectionStatus', state.collectionStatus);
    if (state.profileState) next.set('profileState', state.profileState);
    serializePairFilter(next, state.pairFilter);
    if (state.page !== 1) next.set('page', String(state.page));
    if (state.pageSize !== DEFAULT_PAGE_SIZE) next.set('pageSize', String(state.pageSize));
    return next;
};

const hasValue = (value: unknown) => value !== undefined && value !== null && value !== '';
const compactNumber = (value?: string | number, currency = false) => {
    if (!hasValue(value)) return '-';
    const numeric = Number(value);
    if (!Number.isFinite(numeric)) return String(value);
    const formatted = new Intl.NumberFormat(undefined, {notation: Math.abs(numeric) >= 100000 ? 'compact' : 'standard', maximumFractionDigits: Math.abs(numeric) < 1 ? 8 : 2}).format(numeric);
    return currency ? `$${formatted}` : formatted;
};
const formatInteger = (value?: string) => (value ? value.replace(/\B(?=(\d{3})+(?!\d))/g, ',') : '-');

const workflowTag = (value?: string) => {
    const color: Record<string, string> = {queued: 'default', collecting: 'blue', pending: 'gold', complete: 'green', incomplete: 'gold', needs_attention: 'red', failed: 'red'};
    return <Tag color={value ? color[value] : undefined}>{value?.replace(/_/g, ' ') || 'unknown'}</Tag>;
};

const SignalTag = (props: {value?: boolean}) => <StatusTag value={props.value ? 'Detected' : 'Clear'} positive={props.value === false} negative={props.value === true} />;
const pairUnavailable = (profileState?: string) => <Typography.Text type='secondary'>{profileState === 'pending' ? 'Profile pending' : 'Unavailable'}</Typography.Text>;

const PairSnapshot = (props: {pair?: TokenProjectPairProfileSummary; profileState?: string}) => {
    if (!props.pair) return pairUnavailable(props.profileState);
    return (
        <div className='projects-pair-summary'>
            <div className='projects-pair-summary__line'><Typography.Text type='secondary'>Created</Typography.Text><StatusTag value={props.pair.isCreated ? 'Yes' : 'No'} positive={props.pair.isCreated} /></div>
            <div className='projects-pair-summary__line'><Typography.Text type='secondary'>Balance &gt; supply</Typography.Text><SignalTag value={props.pair.pairTokenBalanceExceedsTotalSupply} /></div>
            <div className='projects-pair-summary__line'><Typography.Text type='secondary'>Minimum LP only</Typography.Text><SignalTag value={props.pair.lpMinimumSupplyOnly} /></div>
            <div className='projects-pair-summary__line'><Typography.Text type='secondary'>Fee LP ≥ 90%</Typography.Text><SignalTag value={props.pair.fixedFeeAddressLpShareGte90Percent} /></div>
            <div className='projects-pair-summary__line'><Typography.Text type='secondary'>Quote USDT</Typography.Text><span>{formatInteger(props.pair.quoteUsdtValueInt)}</span></div>
            <div className='projects-pair-summary__line'><Typography.Text type='secondary'>Reserves updated</Typography.Text><span>{formatBeijingUnixSeconds(props.pair.reserveUpdatedAt) || '-'}</span></div>
        </div>
    );
};

const projectDetailLink = (item: TokenProjectListItem) =>
    item.projectID ? <ProjectDetailLink projectID={item.projectID} /> : <Typography.Text type='secondary'>Unavailable</Typography.Text>;

const projectIdentity = (item: TokenProjectListItem) => (
    <span className='projects-token'>
        <Typography.Text strong={true}>{item.symbol || item.name || 'Unnamed token'}</Typography.Text>
        <Typography.Text type='secondary'>Project #{item.projectID || '-'}</Typography.Text>
        {item.name && item.name !== item.symbol && <Typography.Text type='secondary'>{item.name}</Typography.Text>}
        {projectDetailLink(item)}
    </span>
);

const collectionProgress = (item: TokenProjectListItem) => {
    const total = item.collectionTotalCount || 6;
    const terminal = item.collectionTerminalCount || 0;
    return (
        <div className='projects-collection-progress'>
            <span>{workflowTag(item.collectionStatus)}</span>
            <Progress percent={Math.min(100, Math.round((terminal / total) * 100))} size='small' showInfo={false} status={item.collectionStatus === 'needs_attention' ? 'exception' : undefined} />
            <Typography.Text type='secondary'>{item.collectionSucceededCount || 0}/{total} succeeded · {terminal}/{total} terminal</Typography.Text>
        </div>
    );
};

const pairColumn = (title: string, select: (item: TokenProjectListItem) => TokenProjectPairProfileSummary | undefined): ColumnsType<TokenProjectListItem>[number] => ({
    title,
    width: 260,
    render: item => <PairSnapshot pair={select(item)} profileState={item.profileState} />
});

const projectColumns: ColumnsType<TokenProjectListItem> = [
    {title: 'Logo', fixed: 'left', width: 64, align: 'center', render: item => <TokenLogo logoURL={item.market?.logoURL} symbol={item.symbol} />},
    {title: 'Project', fixed: 'left', width: 220, render: projectIdentity},
    {
        title: 'Discovery',
        children: [
            {title: 'Chain', width: 120, render: item => <ChainBadge chainID={item.chainID} />},
            {title: 'Block', width: 110, dataIndex: 'blockNumber'},
            {title: 'Block Time', width: 180, render: item => formatBeijingUnixSeconds(item.blockTime) || '-'},
            {title: 'Saved', width: 180, render: item => formatBeijingDateTime(item.createdAt) || '-'}
        ]
    },
    {title: 'Collection', width: 245, render: collectionProgress},
    {
        title: 'Profile',
        width: 180,
        render: item => <div className='projects-token'>{workflowTag(item.profileState)}<Typography.Text type='secondary'>{item.completenessStatus || 'Not built'}</Typography.Text><Typography.Text type='secondary'>{formatBeijingDateTime(item.profileBuiltAt) || '-'}</Typography.Text></div>
    },
    {
        title: 'Market',
        children: [
            {title: 'Price', width: 115, render: item => compactNumber(item.market?.currentPriceUSD, true)},
            {title: 'Market Cap', width: 125, render: item => compactNumber(item.market?.marketCapUSD, true)},
            {title: 'TVL', width: 115, render: item => compactNumber(item.market?.tvlUSD, true)},
            {title: 'Holders', width: 105, render: item => compactNumber(item.market?.holders)}
        ]
    },
    pairColumn('WETH / WBNB', item => item.wrappedNativePair),
    pairColumn('USDT', item => item.usdtPair)
];

const ProjectCard = (props: {item: TokenProjectListItem}) => (
    <Card className='projects-compact-card projects-unified-card' size='small' title={
        <span className='projects-compact-card__title'><TokenLogo logoURL={props.item.market?.logoURL} symbol={props.item.symbol} /><span>{props.item.symbol || props.item.name || 'Unnamed token'}</span><ChainBadge chainID={props.item.chainID} /></span>
    } extra={projectDetailLink(props.item)}>
        <div className='projects-unified-card__sections'>
            <section aria-label='Discovery'><Typography.Title level={5}>Discovery</Typography.Title><KeyValueGrid columns={2} items={[
                {label: 'Project', value: `#${props.item.projectID || '-'}`},
                {label: 'Block', value: props.item.blockNumber || '-'},
                {label: 'Block time', value: formatBeijingUnixSeconds(props.item.blockTime) || '-'},
                {label: 'Saved', value: formatBeijingDateTime(props.item.createdAt) || '-'}
            ]} /></section>
            <section aria-label='Collection and profile'><Typography.Title level={5}>Collection & Profile</Typography.Title>{collectionProgress(props.item)}<KeyValueGrid columns={2} items={[
                {label: 'Profile', value: workflowTag(props.item.profileState)},
                {label: 'Completeness', value: props.item.completenessStatus || '-'},
                {label: 'Built', value: formatBeijingDateTime(props.item.profileBuiltAt) || '-'}
            ]} /></section>
            <section aria-label='Market'><Typography.Title level={5}>Market</Typography.Title><KeyValueGrid columns={2} items={[
                {label: 'Price', value: compactNumber(props.item.market?.currentPriceUSD, true)},
                {label: 'Market cap', value: compactNumber(props.item.market?.marketCapUSD, true)},
                {label: 'TVL', value: compactNumber(props.item.market?.tvlUSD, true)},
                {label: 'Holders', value: compactNumber(props.item.market?.holders)}
            ]} /></section>
            <div className='projects-unified-card__pairs'>
                <section className='projects-unified-card__pair' aria-label='WETH or WBNB pair profile'><Typography.Title level={5}>WETH / WBNB</Typography.Title><PairSnapshot pair={props.item.wrappedNativePair} profileState={props.item.profileState} /></section>
                <section className='projects-unified-card__pair' aria-label='USDT pair profile'><Typography.Title level={5}>USDT</Typography.Title><PairSnapshot pair={props.item.usdtPair} profileState={props.item.profileState} /></section>
            </div>
        </div>
    </Card>
);

export const ProjectsPage = () => {
    const [params, setParams] = useSearchParams();
    const [filtersOpen, setFiltersOpen] = React.useState(false);
    const contract = params.get('contract') || '';
    const codeHash = params.get('codeHash') || '';
    const collectionStatus = enumParam(params.get('collectionStatus'), collectionStatuses);
    const profileState = enumParam(params.get('profileState'), profileStates);
    const pairParams = ['pairBalanceSupply', 'pairMinimumLP', 'pairFeeLPShare', 'pairQuoteMin', 'pairQuoteMax'].map(key => params.get(key) || '').join('|');
    const pairFilter = React.useMemo(() => pairFilterFromParams(params), [pairParams]);
    const page = positiveIntegerParam(params.get('page')) || 1;
    const requestedPageSize = positiveIntegerParam(params.get('pageSize')) || DEFAULT_PAGE_SIZE;
    const pageSize = PAGE_SIZE_OPTIONS.includes(requestedPageSize) ? requestedPageSize : DEFAULT_PAGE_SIZE;
    const queryState = React.useMemo<ProjectsQueryState>(() => ({contract, codeHash, collectionStatus, profileState, pairFilter, page, pageSize}), [contract, codeHash, collectionStatus, profileState, pairFilter, page, pageSize]);
    const filterState = React.useMemo<ProjectsFilterState>(() => ({contract, codeHash, collectionStatus, profileState, pairFilter}), [contract, codeHash, collectionStatus, profileState, pairFilter]);
    const rawSearch = params.toString();
    React.useEffect(() => {
        const canonical = serializeQueryState(queryState);
        if (canonical.toString() !== rawSearch) setParams(canonical, {replace: true});
    }, [queryState, rawSearch, setParams]);

    const listCacheKey = JSON.stringify(['token.projects', queryState]);
    const data = useCachedAsyncData(listCacheKey, () => services.tokenapi.listProjects({
        page, pageSize, contract: contract || undefined, codeHash: codeHash || undefined,
        collectionStatus: collectionStatus || undefined, profileState: profileState || undefined,
        pairBalanceSupplyStates: pairFilter.balanceSupply,
        pairMinimumLPStates: pairFilter.minimumLP,
        pairFeeLPShareStates: pairFilter.feeLPShare,
        pairQuoteUSDTMin: pairFilter.quoteMin || undefined,
        pairQuoteUSDTMax: pairFilter.quoteMax || undefined
    }), {staleTimeMs: PROJECTS_LIST_STALE_TIME_MS, module: AccountDataModule.Token});
    const reloadRef = React.useRef(data.reload);
    reloadRef.current = data.reload;
    const needsPolling = Boolean(data.data?.items.some(item => item.collectionStatus === 'queued' || item.collectionStatus === 'collecting' || item.profileState === 'pending'));
    React.useEffect(() => {
        if (!needsPolling) return;
        const timer = window.setInterval(() => reloadRef.current(), PROJECTS_POLL_INTERVAL_MS);
        return () => window.clearInterval(timer);
    }, [needsPolling]);
    useRestoreProjectsScroll(Boolean(data.data));

    const activeFilterCount = generalFilterCount(filterState) + profilePairFilterCount(pairFilter);
    const applyFilters = (nextFilters: ProjectsFilterState) => {setFiltersOpen(false); setParams(serializeQueryState({...queryState, ...nextFilters, page: 1}));};
    const setPage = (nextPage: number, nextPageSize: number) => setParams(serializeQueryState({...queryState, page: nextPage, pageSize: nextPageSize}));

    return (
        <AppPage title='Projects' subtitle='Discovered projects, one-time collection progress, and the resulting immutable profile.' loading={data.loading || data.refreshing} error={data.error} onRefresh={data.reload} filters={
            <div className='projects-controls'>
                <div className='projects-controls__toolbar'><div className='projects-controls__actions'>
                    <Badge count={activeFilterCount} size='small' overflowCount={99}><Button aria-label={`Open filters${activeFilterCount ? `, ${activeFilterCount} active fields` : ''}`} icon={<FilterOutlined />} onClick={() => setFiltersOpen(true)}>Filters</Button></Badge>
                    <Button disabled={!activeFilterCount} onClick={() => setParams(serializeQueryState({...queryState, ...emptyProjectsFilterState(), page: 1}))}>Clear all</Button>
                </div></div>
                <ProjectsFiltersModal open={filtersOpen} filters={filterState} onCancel={() => setFiltersOpen(false)} onApply={applyFilters} />
            </div>
        }>
            <div className='projects-unified-table-region'>
                <ResourceTable label='Projects with collection, profile, market, and pair summaries' rowKey='projectID' items={data.data?.items || []} columns={projectColumns} loading={data.loading} total={data.data?.total} page={page} pageSize={pageSize} onPageChange={setPage} scrollX={PROJECTS_TABLE_WIDTH} stickyHeader={true} compactRender={item => <ProjectCard item={item} />} compactEmptyDescription='No projects match the filters' />
            </div>
        </AppPage>
    );
};
