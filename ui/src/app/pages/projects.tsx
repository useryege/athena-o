import {FilterOutlined, SortAscendingOutlined, SortDescendingOutlined} from '@ant-design/icons';
import {Badge, Button, Card, Select, Tag, Tooltip, Typography} from 'antd';
import type {ColumnsType, TableProps} from 'antd/es/table';
import * as React from 'react';
import {useSearchParams} from 'react-router-dom';
import {AppPage, ChoiceGroup, KeyValueGrid, ResourceTable, StatusTag, useCachedAsyncData} from '../components';
import {formatBeijingDateTime, formatBeijingUnixSeconds} from '../shared/format';
import {DEFAULT_PAGE_SIZE, PAGE_SIZE_OPTIONS} from '../shared/pagination';
import {services} from '../shared/services';
import {TokenProjectListItem, TokenProjectReportPairRisk} from '../shared/services/token-service';
import {ProjectDetailLink, useRestoreProjectsScroll} from './project-navigation';
import {
    emptyProjectsFilterState,
    evaluationStatuses,
    generalFilterCount,
    hasReportPairFilter,
    ProjectsFiltersModal,
    ProjectsFilterState,
    ReportPairFilterState,
    ReportPairKind,
    reportPairFilterCount,
    reportPairMissingStates,
    reportPairRiskStates,
    reportStates,
    researchStatuses,
    selectionOutcomes
} from './projects-filters-modal';
import {ChainBadge, TokenLogo, chainLabel} from './token-shared';

type ProjectsView = 'overview' | 'report-status' | 'wrapped-native' | 'usdt';
type ReportRiskSortKey = 'project' | 'created' | 'removeLiquidity' | 'mint' | 'quoteUsdt' | 'lastSwap';
type ReportRiskSortOrder = 'asc' | 'desc';

const projectsViews = ['overview', 'report-status', 'wrapped-native', 'usdt'] as const;
const reportRiskSortKeys = ['project', 'created', 'removeLiquidity', 'mint', 'quoteUsdt', 'lastSwap'] as const;
const reportRiskSortOrders = ['asc', 'desc'] as const;
const reportRiskSortOptions = [
    {label: 'Default order', value: 'default'},
    {label: 'Project', value: 'project'},
    {label: 'Created', value: 'created'},
    {label: 'Remove Liquidity', value: 'removeLiquidity'},
    {label: 'Mint', value: 'mint'},
    {label: 'Quote USDT', value: 'quoteUsdt'},
    {label: 'Last Swap', value: 'lastSwap'}
];
const PROJECTS_LIST_STALE_TIME_MS = 30_000;
const RUNTIME_CONFIGURATION_STALE_TIME_MS = 5 * 60_000;
const RUNTIME_CONFIGURATION_CACHE_KEY = 'token.runtime-configuration';

const positiveIntegerParam = (value: string | null) => {
    const parsed = Number(value);
    return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : undefined;
};

const enumParam = (value: string | null, allowed: readonly string[]) => (value && allowed.includes(value) ? value : '');

interface ProjectsQueryState extends ProjectsFilterState {
    view: ProjectsView;
    riskSort?: ReportRiskSortKey;
    riskSortOrder?: ReportRiskSortOrder;
    page: number;
    pageSize: number;
}

const enumListParam = <T extends string>(value: string | null, allowed: readonly T[]): T[] => {
    const selected = new Set((value || '').split(',').map(item => item.trim()));
    return allowed.filter(item => selected.has(item));
};

const unsignedIntegerParam = (value: string | null) => {
    const normalized = (value || '').trim();
    return /^\d+$/.test(normalized) ? BigInt(normalized).toString() : '';
};

const reportPairFilterFromParams = (params: URLSearchParams, prefix: ReportPairKind): ReportPairFilterState => ({
    removeLiquidity: enumListParam(params.get(`${prefix}RemoveLiquidity`), reportPairRiskStates),
    mint: enumListParam(params.get(`${prefix}Mint`), reportPairRiskStates),
    quoteMin: unsignedIntegerParam(params.get(`${prefix}QuoteMin`)),
    quoteMax: unsignedIntegerParam(params.get(`${prefix}QuoteMax`)),
    quoteMissing: enumListParam(params.get(`${prefix}QuoteMissing`), reportPairMissingStates)
});

const serializeReportPairFilter = (params: URLSearchParams, prefix: ReportPairKind, filter: ReportPairFilterState) => {
    if (filter.removeLiquidity.length > 0) {
        params.set(`${prefix}RemoveLiquidity`, reportPairRiskStates.filter(value => filter.removeLiquidity.includes(value)).join(','));
    }
    if (filter.mint.length > 0) {
        params.set(`${prefix}Mint`, reportPairRiskStates.filter(value => filter.mint.includes(value)).join(','));
    }
    if (filter.quoteMin) {
        params.set(`${prefix}QuoteMin`, filter.quoteMin);
    }
    if (filter.quoteMax) {
        params.set(`${prefix}QuoteMax`, filter.quoteMax);
    }
    if (filter.quoteMissing.length > 0) {
        params.set(`${prefix}QuoteMissing`, reportPairMissingStates.filter(value => filter.quoteMissing.includes(value)).join(','));
    }
};

const serializeQueryState = (state: ProjectsQueryState) => {
    const next = new URLSearchParams();
    if (state.view !== 'overview') {
        next.set('view', state.view);
    }
    if (state.chainID !== undefined) {
        next.set('chainID', String(state.chainID));
    }
    if (state.projectID !== undefined) {
        next.set('projectID', String(state.projectID));
    }
    if (state.contract) {
        next.set('contract', state.contract);
    }
    if (state.codeHash) {
        next.set('codeHash', state.codeHash);
    }
    if (state.researchStatus) {
        next.set('researchStatus', state.researchStatus);
    }
    if (state.reportState) {
        next.set('reportState', state.reportState);
    }
    if (state.evaluationStatus) {
        next.set('evaluationStatus', state.evaluationStatus);
    }
    if (state.selectionOutcome) {
        next.set('selectionOutcome', state.selectionOutcome);
    }
    serializeReportPairFilter(next, 'weth', state.wethPairFilter);
    serializeReportPairFilter(next, 'usdt', state.usdtPairFilter);
    if (state.riskSort && state.riskSortOrder) {
        next.set('riskSort', state.riskSort);
        next.set('riskSortOrder', state.riskSortOrder);
    }
    if (state.page !== 1) {
        next.set('page', String(state.page));
    }
    if (state.pageSize !== DEFAULT_PAGE_SIZE) {
        next.set('pageSize', String(state.pageSize));
    }
    return next;
};

const formatInteger = (value?: string) => (value ? value.replace(/\B(?=(\d{3})+(?!\d))/g, ',') : '-');

const researchTag = (status?: string) => <StatusTag value={status || 'Not started'} positive={status === 'selected'} negative={status === 'rejected' || status === 'expired'} />;

const reportStateTag = (state?: string) => {
    if (!state) {
        return <Tag>No report</Tag>;
    }
    return <Tag color={state === 'complete' ? 'green' : 'gold'}>{state}</Tag>;
};

const evaluationTag = (status?: string) => {
    const colors: Record<string, string> = {pending: 'gold', running: 'blue', succeeded: 'green', failed: 'red'};
    return <Tag color={status ? colors[status] : undefined}>{status || 'No task'}</Tag>;
};

const outcomeTag = (outcome?: string) => (
    <Tag color={outcome === 'selected' ? 'green' : outcome === 'rejected' ? 'red' : outcome === 'deferred' ? 'gold' : undefined}>{outcome || 'No outcome'}</Tag>
);

const pairCreatedTag = (value?: boolean) => {
    if (value === undefined) {
        return <Tag>Unknown</Tag>;
    }
    return <Tag color={value ? 'green' : undefined}>{value ? 'Created' : 'Not created'}</Tag>;
};

const pairRiskTag = (value?: boolean) => {
    if (value === undefined) {
        return <Tag>Unknown</Tag>;
    }
    return <Tag color={value ? 'red' : 'green'}>{value ? 'Detected' : 'Clear'}</Tag>;
};

const projectDetailLink = (item: TokenProjectListItem) =>
    item.projectID ? <ProjectDetailLink projectID={item.projectID} /> : <Typography.Text type='secondary'>Unavailable</Typography.Text>;

const tokenValue = (item: TokenProjectListItem) => (
    <span className='projects-token'>
        <Typography.Text strong={true}>{item.symbol || '-'}</Typography.Text>
        <Typography.Text type='secondary'>{item.name || '-'}</Typography.Text>
    </span>
);

const projectIdentity = (item: TokenProjectListItem) => (
    <span className='projects-token'>
        <Typography.Text strong={true}>{item.symbol || item.name || 'Unnamed token'}</Typography.Text>
        <Typography.Text type='secondary'>Project #{item.projectID || '-'}</Typography.Text>
        {item.name && item.name !== item.symbol && <Typography.Text type='secondary'>{item.name}</Typography.Text>}
        {projectDetailLink(item)}
    </span>
);

const compactProjectIdentity = (item: TokenProjectListItem) => (
    <span className='projects-token'>
        <Typography.Text>Project #{item.projectID || '-'}</Typography.Text>
        {item.name && item.name !== item.symbol && <Typography.Text type='secondary'>{item.name}</Typography.Text>}
    </span>
);

const reportProjectIdentity = (item: TokenProjectListItem) => (
    <span className='projects-token'>
        {projectIdentity(item)}
        <ChainBadge chainID={item.chainID} />
    </span>
);

const reportRiskSummary = (items: Array<{label: string; value: React.ReactNode}>) => (
    <div className='projects-report-risk-summary'>
        {items.map(item => (
            <div className='projects-report-risk-summary__line' key={item.label}>
                <Typography.Text className='projects-report-risk-summary__label' type='secondary'>
                    {item.label}
                </Typography.Text>
                <div className='projects-report-risk-summary__value'>{item.value}</div>
            </div>
        ))}
    </div>
);

const reportRiskProjectContext = (item: TokenProjectListItem, includeReportContext: boolean) => {
    const report = item.currentReport;
    return (
        <div className='projects-report-risk-summary'>
            {reportProjectIdentity(item)}
            {includeReportContext && (
                <>
                    <div className='projects-report-risk-summary__line'>
                        <Typography.Text className='projects-report-risk-summary__label' type='secondary'>
                            Report
                        </Typography.Text>
                        <div className='projects-report-risk-summary__value'>{report ? `Revision ${report.revision ?? '-'}` : <Tag>No report</Tag>}</div>
                    </div>
                    <div className='projects-report-risk-summary__line'>
                        <Typography.Text className='projects-report-risk-summary__label' type='secondary'>
                            State
                        </Typography.Text>
                        <div className='projects-report-risk-summary__value'>{reportStateTag(report?.completenessStatus)}</div>
                    </div>
                </>
            )}
        </div>
    );
};

const evaluationLastError = (value?: string) =>
    value ? (
        <Typography.Paragraph className='projects-report-risk-error' ellipsis={{rows: 2, expandable: 'collapsible', symbol: expanded => (expanded ? 'Show less' : 'Show more')}}>
            {value}
        </Typography.Paragraph>
    ) : (
        '-'
    );

const unavailablePairValue = (noReport: boolean) => (noReport ? <Tag>No report</Tag> : <Typography.Text type='secondary'>Risk unavailable</Typography.Text>);

const pairSnapshotItems = (pair: TokenProjectReportPairRisk | undefined, noReport: boolean) => [
    {label: 'Created', value: noReport ? <Tag>No report</Tag> : pair ? pairCreatedTag(pair.isCreated) : unavailablePairValue(false)},
    {label: 'Remove liquidity', value: noReport ? <Tag>No report</Tag> : pair ? pairRiskTag(pair.isRemoveLiquidity) : unavailablePairValue(false)},
    {label: 'Mint', value: noReport ? <Tag>No report</Tag> : pair ? pairRiskTag(pair.isMint) : unavailablePairValue(false)},
    {label: 'Quote USDT', value: pair?.quoteUsdtValueInt ? formatInteger(pair.quoteUsdtValueInt) : unavailablePairValue(noReport)},
    {label: 'Last swap', value: pair?.lastSwapAt ? formatBeijingDateTime(pair.lastSwapAt) || '-' : unavailablePairValue(noReport)}
];

const projectCollator = new Intl.Collator(undefined, {numeric: true, sensitivity: 'base'});

const projectPair = (item: TokenProjectListItem, kind: ReportPairKind) => (kind === 'weth' ? item.currentReport?.riskSummary?.wethPair : item.currentReport?.riskSummary?.usdtPair);

const comparePresentValues = <T,>(left: T | undefined, right: T | undefined, order: ReportRiskSortOrder, compare: (a: T, b: T) => number) => {
    if (left === undefined && right === undefined) {
        return 0;
    }
    if (left === undefined) {
        return 1;
    }
    if (right === undefined) {
        return -1;
    }
    const result = compare(left, right);
    return order === 'asc' ? result : -result;
};

const sortReportRiskItems = (items: TokenProjectListItem[], kind: ReportPairKind, key?: ReportRiskSortKey, order?: ReportRiskSortOrder) => {
    if (!key || !order) {
        return items;
    }
    return items
        .map((item, index) => ({item, index}))
        .sort((left, right) => {
            const leftPair = projectPair(left.item, kind);
            const rightPair = projectPair(right.item, kind);
            let result = 0;
            switch (key) {
                case 'project': {
                    const leftLabel = left.item.symbol || left.item.name || 'Unnamed token';
                    const rightLabel = right.item.symbol || right.item.name || 'Unnamed token';
                    result = projectCollator.compare(leftLabel, rightLabel);
                    if (result === 0) {
                        result = (left.item.projectID || 0) - (right.item.projectID || 0);
                    }
                    result = order === 'asc' ? result : -result;
                    break;
                }
                case 'created':
                    result = comparePresentValues(leftPair?.isCreated, rightPair?.isCreated, order, (a, b) => Number(a) - Number(b));
                    break;
                case 'removeLiquidity':
                    result = comparePresentValues(leftPair?.isRemoveLiquidity, rightPair?.isRemoveLiquidity, order, (a, b) => Number(a) - Number(b));
                    break;
                case 'mint':
                    result = comparePresentValues(leftPair?.isMint, rightPair?.isMint, order, (a, b) => Number(a) - Number(b));
                    break;
                case 'quoteUsdt':
                    result = comparePresentValues(
                        leftPair?.quoteUsdtValueInt ? BigInt(leftPair.quoteUsdtValueInt) : undefined,
                        rightPair?.quoteUsdtValueInt ? BigInt(rightPair.quoteUsdtValueInt) : undefined,
                        order,
                        (a, b) => (a < b ? -1 : a > b ? 1 : 0)
                    );
                    break;
                case 'lastSwap': {
                    const leftTime = leftPair?.lastSwapAt ? Date.parse(leftPair.lastSwapAt) : Number.NaN;
                    const rightTime = rightPair?.lastSwapAt ? Date.parse(rightPair.lastSwapAt) : Number.NaN;
                    result = comparePresentValues(Number.isFinite(leftTime) ? leftTime : undefined, Number.isFinite(rightTime) ? rightTime : undefined, order, (a, b) => a - b);
                    break;
                }
            }
            return result || left.index - right.index;
        })
        .map(entry => entry.item);
};

const pairStateLabels: Record<string, string> = {
    detected: 'Detected',
    clear: 'Clear',
    no_report: 'No report',
    risk_unavailable: 'Risk unavailable'
};

const summarizeStates = (states: string[]) => {
    const labels = states.map(state => pairStateLabels[state] || state);
    return labels.length > 1 ? `${labels[0]} +${labels.length - 1}` : labels[0] || '';
};

const quoteFilterSummary = (filter: ReportPairFilterState) => {
    const range =
        filter.quoteMin && filter.quoteMax
            ? `${formatInteger(filter.quoteMin)}–${formatInteger(filter.quoteMax)}`
            : filter.quoteMin
              ? `≥ ${formatInteger(filter.quoteMin)}`
              : filter.quoteMax
                ? `≤ ${formatInteger(filter.quoteMax)}`
                : '';
    const missing = summarizeStates(filter.quoteMissing);
    return [range, missing].filter(Boolean).join(' or ');
};

const ProjectOverviewCard = (props: {item: TokenProjectListItem}) => {
    return (
        <Card
            className='projects-compact-card'
            size='small'
            title={
                <span className='projects-compact-card__title'>
                    <TokenLogo logoURL={props.item.logoURL} symbol={props.item.symbol} />
                    <span>{props.item.symbol || props.item.name || 'Unnamed token'}</span>
                    <ChainBadge chainID={props.item.chainID} />
                </span>
            }
            extra={projectDetailLink(props.item)}>
            <KeyValueGrid
                columns={1}
                items={[
                    {label: 'Project', value: compactProjectIdentity(props.item)},
                    {label: 'Block Time', value: formatBeijingUnixSeconds(props.item.blockTime) || '-'},
                    {label: 'Created', value: formatBeijingDateTime(props.item.createdAt) || '-'}
                ]}
            />
        </Card>
    );
};

const ProjectRiskCard = (props: {item: TokenProjectListItem; children: React.ReactNode}) => {
    return (
        <Card
            className='projects-compact-card projects-report-risk-card'
            size='small'
            title={
                <span className='projects-compact-card__title'>
                    <TokenLogo logoURL={props.item.logoURL} symbol={props.item.symbol} />
                    <span>{props.item.symbol || props.item.name || 'Unnamed token'}</span>
                    <ChainBadge chainID={props.item.chainID} />
                </span>
            }
            extra={projectDetailLink(props.item)}>
            {props.children}
        </Card>
    );
};

const ProjectReportStatusCard = (props: {item: TokenProjectListItem}) => {
    const report = props.item.currentReport;
    const evaluation = report?.evaluation;
    return (
        <ProjectRiskCard item={props.item}>
            <div className='projects-report-risk-card__sections'>
                <section aria-label='Project and report state'>
                    <Typography.Title level={5}>Project & report</Typography.Title>
                    <KeyValueGrid
                        columns={2}
                        items={[
                            {label: 'Project ID', value: props.item.projectID || '-'},
                            {label: 'Token', value: tokenValue(props.item)},
                            {label: 'Chain', value: <ChainBadge chainID={props.item.chainID} />},
                            {label: 'Research', value: researchTag(props.item.researchStatus)},
                            {label: 'Report revision', value: report?.revision ?? '-'},
                            {label: 'Report state', value: reportStateTag(report?.completenessStatus)},
                            {label: 'Built', value: formatBeijingDateTime(report?.builtAt) || '-'}
                        ]}
                    />
                </section>
                <section aria-label='Current report evaluation'>
                    <Typography.Title level={5}>Evaluation</Typography.Title>
                    <KeyValueGrid
                        columns={2}
                        items={[
                            {label: 'Status', value: evaluationTag(evaluation?.status)},
                            {label: 'Failed attempts', value: evaluation?.failedAttempts ?? '-'},
                            {label: 'Last error', value: evaluation?.lastError || '-'},
                            {label: 'Updated', value: formatBeijingDateTime(evaluation?.updatedAt) || '-'},
                            {label: 'Outcome', value: outcomeTag(evaluation?.outcome)},
                            {label: 'Evaluated', value: formatBeijingDateTime(evaluation?.evaluatedAt) || '-'}
                        ]}
                    />
                </section>
            </div>
        </ProjectRiskCard>
    );
};

const ProjectPairRiskCard = (props: {item: TokenProjectListItem; kind: ReportPairKind}) => {
    const report = props.item.currentReport;
    const label = props.kind === 'weth' ? 'WETH / WBNB' : 'USDT';
    const pair = props.kind === 'weth' ? report?.riskSummary?.wethPair : report?.riskSummary?.usdtPair;
    return (
        <ProjectRiskCard item={props.item}>
            <div className='projects-report-risk-card__sections'>
                <section aria-label='Project and report context'>
                    <Typography.Title level={5}>Project & report</Typography.Title>
                    <KeyValueGrid
                        columns={2}
                        items={[
                            {label: 'Project ID', value: props.item.projectID || '-'},
                            {label: 'Token', value: tokenValue(props.item)},
                            {label: 'Report revision', value: report?.revision ?? '-'},
                            {label: 'Report state', value: reportStateTag(report?.completenessStatus)},
                            {label: 'Built', value: formatBeijingDateTime(report?.builtAt) || '-'}
                        ]}
                    />
                </section>
                <section aria-label={`${label} pair report snapshot`}>
                    <Typography.Title level={5}>{label} pair</Typography.Title>
                    <Typography.Text type='secondary'>{!report ? 'No report' : !pair ? 'Risk unavailable' : 'Report snapshot'}</Typography.Text>
                    <KeyValueGrid columns={1} items={pairSnapshotItems(pair, !report)} />
                </section>
            </div>
        </ProjectRiskCard>
    );
};

export const ProjectsPage = () => {
    const [params, setParams] = useSearchParams();
    const [filtersOpen, setFiltersOpen] = React.useState(false);
    const view = (enumParam(params.get('view'), projectsViews) || 'overview') as ProjectsView;
    const reportProjection = view !== 'overview';
    const chainID = positiveIntegerParam(params.get('chainID'));
    const projectID = positiveIntegerParam(params.get('projectID'));
    const contract = params.get('contract') || '';
    const codeHash = params.get('codeHash') || '';
    const researchStatus = enumParam(params.get('researchStatus'), researchStatuses);
    const reportState = enumParam(params.get('reportState'), reportStates);
    const evaluationStatus = enumParam(params.get('evaluationStatus'), evaluationStatuses);
    const selectionOutcome = enumParam(params.get('selectionOutcome'), selectionOutcomes);
    const wethPairFilterParams = ['wethRemoveLiquidity', 'wethMint', 'wethQuoteMin', 'wethQuoteMax', 'wethQuoteMissing'].map(key => params.get(key) || '').join('|');
    const usdtPairFilterParams = ['usdtRemoveLiquidity', 'usdtMint', 'usdtQuoteMin', 'usdtQuoteMax', 'usdtQuoteMissing'].map(key => params.get(key) || '').join('|');
    const wethPairFilter = React.useMemo(() => reportPairFilterFromParams(params, 'weth'), [wethPairFilterParams]);
    const usdtPairFilter = React.useMemo(() => reportPairFilterFromParams(params, 'usdt'), [usdtPairFilterParams]);
    const parsedRiskSort = enumParam(params.get('riskSort'), reportRiskSortKeys) as ReportRiskSortKey | '';
    const parsedRiskSortOrder = enumParam(params.get('riskSortOrder'), reportRiskSortOrders) as ReportRiskSortOrder | '';
    const riskSort = parsedRiskSort && parsedRiskSortOrder ? parsedRiskSort : undefined;
    const riskSortOrder = parsedRiskSort && parsedRiskSortOrder ? parsedRiskSortOrder : undefined;
    const page = positiveIntegerParam(params.get('page')) || 1;
    const requestedPageSize = positiveIntegerParam(params.get('pageSize')) || DEFAULT_PAGE_SIZE;
    const pageSize = PAGE_SIZE_OPTIONS.includes(requestedPageSize) ? requestedPageSize : DEFAULT_PAGE_SIZE;

    const queryState = React.useMemo<ProjectsQueryState>(
        () => ({
            view,
            chainID,
            projectID,
            contract,
            codeHash,
            researchStatus,
            reportState,
            evaluationStatus,
            selectionOutcome,
            wethPairFilter,
            usdtPairFilter,
            riskSort,
            riskSortOrder,
            page,
            pageSize
        }),
        [
            view,
            chainID,
            projectID,
            contract,
            codeHash,
            researchStatus,
            reportState,
            evaluationStatus,
            selectionOutcome,
            wethPairFilter,
            usdtPairFilter,
            riskSort,
            riskSortOrder,
            page,
            pageSize
        ]
    );
    const filterState = React.useMemo<ProjectsFilterState>(
        () => ({chainID, projectID, contract, codeHash, researchStatus, reportState, evaluationStatus, selectionOutcome, wethPairFilter, usdtPairFilter}),
        [chainID, projectID, contract, codeHash, researchStatus, reportState, evaluationStatus, selectionOutcome, wethPairFilter, usdtPairFilter]
    );
    const rawSearch = params.toString();

    React.useEffect(() => {
        const canonical = serializeQueryState(queryState);
        if (canonical.toString() !== rawSearch) {
            setParams(canonical, {replace: true});
        }
    }, [queryState, rawSearch, setParams]);

    const setView = (nextView: ProjectsView) => {
        setParams(serializeQueryState({...queryState, view: nextView}));
    };
    const setPage = (nextPage: number, nextPageSize: number) => {
        setParams(serializeQueryState({...queryState, page: nextPage, pageSize: nextPageSize}));
    };
    const applyFilters = (nextFilters: ProjectsFilterState) => {
        setFiltersOpen(false);
        setParams(serializeQueryState({...queryState, ...nextFilters, page: 1}));
    };
    const setRiskSort = (nextSort?: ReportRiskSortKey, nextOrder?: ReportRiskSortOrder) => {
        setParams(serializeQueryState({...queryState, riskSort: nextSort, riskSortOrder: nextSort ? nextOrder || 'asc' : undefined}));
    };
    const clearFilters = () => {
        setParams(serializeQueryState({...queryState, ...emptyProjectsFilterState(), page: 1}));
    };

    const activeReportPairKind: ReportPairKind | undefined = view === 'wrapped-native' ? 'weth' : view === 'usdt' ? 'usdt' : undefined;
    const activeReportPairFilter = activeReportPairKind === 'weth' ? wethPairFilter : activeReportPairKind === 'usdt' ? usdtPairFilter : undefined;
    const effectiveReportPairKind = activeReportPairFilter && hasReportPairFilter(activeReportPairFilter) ? activeReportPairKind : undefined;

    const listCacheKey = JSON.stringify([
        'token.projects',
        page,
        pageSize,
        chainID ?? null,
        projectID ?? null,
        contract || null,
        codeHash || null,
        researchStatus || null,
        reportState || null,
        evaluationStatus || null,
        selectionOutcome || null,
        effectiveReportPairKind || null,
        effectiveReportPairKind ? activeReportPairFilter?.removeLiquidity : null,
        effectiveReportPairKind ? activeReportPairFilter?.mint : null,
        effectiveReportPairKind ? activeReportPairFilter?.quoteMin || null : null,
        effectiveReportPairKind ? activeReportPairFilter?.quoteMax || null : null,
        effectiveReportPairKind ? activeReportPairFilter?.quoteMissing : null
    ]);
    const options = useCachedAsyncData(RUNTIME_CONFIGURATION_CACHE_KEY, () => services.tokenapi.getRuntimeConfiguration(), {
        staleTimeMs: RUNTIME_CONFIGURATION_STALE_TIME_MS
    });
    const data = useCachedAsyncData(
        listCacheKey,
        () =>
            services.tokenapi.listProjects({
                page,
                pageSize,
                chainID,
                projectID,
                contract: contract || undefined,
                codeHash: codeHash || undefined,
                researchStatus: researchStatus || undefined,
                reportState: reportState || undefined,
                evaluationStatus: evaluationStatus || undefined,
                selectionOutcome: selectionOutcome || undefined,
                reportPairKind: effectiveReportPairKind,
                reportPairRemoveLiquidityStates: effectiveReportPairKind ? activeReportPairFilter?.removeLiquidity : undefined,
                reportPairMintStates: effectiveReportPairKind ? activeReportPairFilter?.mint : undefined,
                reportPairQuoteUSDTMin: effectiveReportPairKind ? activeReportPairFilter?.quoteMin || undefined : undefined,
                reportPairQuoteUSDTMax: effectiveReportPairKind ? activeReportPairFilter?.quoteMax || undefined : undefined,
                reportPairQuoteMissingStates: effectiveReportPairKind ? activeReportPairFilter?.quoteMissing : undefined
            }),
        {staleTimeMs: PROJECTS_LIST_STALE_TIME_MS}
    );
    useRestoreProjectsScroll(Boolean(data.data));

    const sortColumn = (key: ReportRiskSortKey) => ({
        key,
        sorter: true,
        sortOrder: riskSort === key ? (riskSortOrder === 'asc' ? ('ascend' as const) : ('descend' as const)) : null,
        showSorterTooltip: {title: 'Sorts current page only'}
    });
    const handleReportRiskTableChange: TableProps<TokenProjectListItem>['onChange'] = (_pagination, _filters, sorter) => {
        const activeSorter = Array.isArray(sorter) ? sorter[0] : sorter;
        const nextKey = activeSorter?.columnKey;
        if (!nextKey || !reportRiskSortKeys.includes(nextKey as ReportRiskSortKey) || !activeSorter.order) {
            setRiskSort();
            return;
        }
        setRiskSort(nextKey as ReportRiskSortKey, activeSorter.order === 'ascend' ? 'asc' : 'desc');
    };

    const logoColumn: ColumnsType<TokenProjectListItem>[number] = {
        title: 'Logo',
        fixed: 'left',
        width: 64,
        align: 'center',
        render: item => <TokenLogo logoURL={item.logoURL} symbol={item.symbol} />
    };

    const overviewColumns: ColumnsType<TokenProjectListItem> = [
        logoColumn,
        {title: 'Project', fixed: 'left', width: 220, render: projectIdentity},
        {title: 'Chain', width: 120, render: item => <ChainBadge chainID={item.chainID} />},
        {title: 'Block Time', width: 185, render: item => formatBeijingUnixSeconds(item.blockTime) || '-'},
        {title: 'Created', width: 185, render: item => formatBeijingDateTime(item.createdAt) || '-'}
    ];

    const pairColumns = (pair: (item: TokenProjectListItem) => TokenProjectReportPairRisk | undefined): ColumnsType<TokenProjectListItem> => [
        {...sortColumn('created'), title: 'Created', width: 90, render: item => (item.currentReport ? pairCreatedTag(pair(item)?.isCreated) : <Tag>No report</Tag>)},
        {
            ...sortColumn('removeLiquidity'),
            title: 'Remove Liquidity',
            width: 125,
            render: item => {
                const pairRisk = pair(item);
                return !item.currentReport ? <Tag>No report</Tag> : pairRisk ? pairRiskTag(pairRisk.isRemoveLiquidity) : unavailablePairValue(false);
            }
        },
        {
            ...sortColumn('mint'),
            title: 'Mint',
            width: 80,
            render: item => {
                const pairRisk = pair(item);
                return !item.currentReport ? <Tag>No report</Tag> : pairRisk ? pairRiskTag(pairRisk.isMint) : unavailablePairValue(false);
            }
        },
        {
            ...sortColumn('quoteUsdt'),
            title: 'Quote USDT',
            width: 110,
            render: item => (pair(item)?.quoteUsdtValueInt ? formatInteger(pair(item)?.quoteUsdtValueInt) : unavailablePairValue(!item.currentReport))
        },
        {
            ...sortColumn('lastSwap'),
            title: 'Last Swap',
            width: 145,
            render: item => (pair(item)?.lastSwapAt ? formatBeijingDateTime(pair(item)?.lastSwapAt) || '-' : unavailablePairValue(!item.currentReport))
        }
    ];

    const reportRiskStatusColumns: ColumnsType<TokenProjectListItem> = [
        logoColumn,
        {title: 'Project', width: 200, render: item => reportRiskProjectContext(item, false)},
        {title: 'Research', width: 90, render: item => researchTag(item.researchStatus)},
        {
            title: 'Report',
            width: 150,
            render: item =>
                reportRiskSummary([
                    {label: 'Revision', value: item.currentReport?.revision ?? <Tag>No report</Tag>},
                    {label: 'Completeness', value: reportStateTag(item.currentReport?.completenessStatus)},
                    {label: 'Built', value: formatBeijingDateTime(item.currentReport?.builtAt) || (item.currentReport ? '-' : 'No report')}
                ])
        },
        {
            title: 'Evaluation',
            width: 210,
            render: item =>
                reportRiskSummary([
                    {label: 'Status', value: evaluationTag(item.currentReport?.evaluation?.status)},
                    {label: 'Failed attempts', value: item.currentReport?.evaluation?.failedAttempts ?? '-'},
                    {label: 'Updated', value: formatBeijingDateTime(item.currentReport?.evaluation?.updatedAt) || '-'},
                    {label: 'Last error', value: evaluationLastError(item.currentReport?.evaluation?.lastError)}
                ])
        },
        {
            title: 'Selection',
            width: 150,
            render: item =>
                reportRiskSummary([
                    {label: 'Outcome', value: outcomeTag(item.currentReport?.evaluation?.outcome)},
                    {label: 'Evaluated', value: formatBeijingDateTime(item.currentReport?.evaluation?.evaluatedAt) || '-'}
                ])
        }
    ];

    const reportRiskPairColumns = (pair: (item: TokenProjectListItem) => TokenProjectReportPairRisk | undefined): ColumnsType<TokenProjectListItem> => [
        logoColumn,
        {...sortColumn('project'), title: 'Project', width: 200, render: item => reportRiskProjectContext(item, true)},
        ...pairColumns(pair)
    ];

    const viewColumns =
        view === 'overview'
            ? overviewColumns
            : view === 'report-status'
              ? reportRiskStatusColumns
              : view === 'wrapped-native'
                ? reportRiskPairColumns(item => item.currentReport?.riskSummary?.wethPair)
                : reportRiskPairColumns(item => item.currentReport?.riskSummary?.usdtPair);

    const chainOptions = (options.data?.chains || [])
        .filter((item): item is {chainID: number; chainName?: string} => item.chainID !== undefined)
        .map(item => ({value: item.chainID, label: chainLabel(item.chainID)}));
    const generalCount = generalFilterCount(filterState);
    const wethFilterCount = reportPairFilterCount(wethPairFilter);
    const usdtFilterCount = reportPairFilterCount(usdtPairFilter);
    const effectiveFilterCount = generalCount + (activeReportPairKind === 'weth' ? wethFilterCount : activeReportPairKind === 'usdt' ? usdtFilterCount : 0);
    const hasSavedFilters = generalCount + wethFilterCount + usdtFilterCount > 0;
    const pairLabel = activeReportPairKind === 'weth' ? 'WETH / WBNB' : activeReportPairKind === 'usdt' ? 'USDT' : '';
    const displayStatus = (value: string, noneLabel: string) => (value === 'none' ? noneLabel : value.replace(/_/g, ' '));
    const filterSummaryItems: Array<{key: string; label: string; clear: () => void}> = [];
    const clearFilterField = (key: keyof ProjectsFilterState, value: ProjectsFilterState[keyof ProjectsFilterState]) =>
        applyFilters({...filterState, [key]: value} as ProjectsFilterState);
    if (chainID) {
        filterSummaryItems.push({key: 'chain', label: `Chain: ${chainLabel(chainID)}`, clear: () => clearFilterField('chainID', undefined)});
    }
    if (projectID) {
        filterSummaryItems.push({key: 'project', label: `Project ID: ${projectID}`, clear: () => clearFilterField('projectID', undefined)});
    }
    if (contract) {
        filterSummaryItems.push({key: 'contract', label: `Contract: ${contract}`, clear: () => clearFilterField('contract', '')});
    }
    if (codeHash) {
        filterSummaryItems.push({key: 'codeHash', label: `Code hash: ${codeHash}`, clear: () => clearFilterField('codeHash', '')});
    }
    if (researchStatus) {
        filterSummaryItems.push({key: 'research', label: `Research: ${displayStatus(researchStatus, 'None')}`, clear: () => clearFilterField('researchStatus', '')});
    }
    if (reportState) {
        filterSummaryItems.push({key: 'report', label: `Report: ${displayStatus(reportState, 'No report')}`, clear: () => clearFilterField('reportState', '')});
    }
    if (evaluationStatus) {
        filterSummaryItems.push({
            key: 'evaluation',
            label: `Evaluation: ${displayStatus(evaluationStatus, 'No evaluation task')}`,
            clear: () => clearFilterField('evaluationStatus', '')
        });
    }
    if (selectionOutcome) {
        filterSummaryItems.push({key: 'selection', label: `Selection: ${displayStatus(selectionOutcome, 'No outcome')}`, clear: () => clearFilterField('selectionOutcome', '')});
    }
    if (activeReportPairKind && activeReportPairFilter) {
        const filterKey = activeReportPairKind === 'weth' ? 'wethPairFilter' : 'usdtPairFilter';
        if (activeReportPairFilter.removeLiquidity.length > 0) {
            filterSummaryItems.push({
                key: `${filterKey}-removeLiquidity`,
                label: `${pairLabel} Remove Liquidity: ${summarizeStates(activeReportPairFilter.removeLiquidity)}`,
                clear: () => clearFilterField(filterKey, {...activeReportPairFilter, removeLiquidity: []})
            });
        }
        if (activeReportPairFilter.mint.length > 0) {
            filterSummaryItems.push({
                key: `${filterKey}-mint`,
                label: `${pairLabel} Mint: ${summarizeStates(activeReportPairFilter.mint)}`,
                clear: () => clearFilterField(filterKey, {...activeReportPairFilter, mint: []})
            });
        }
        if (activeReportPairFilter.quoteMin || activeReportPairFilter.quoteMax || activeReportPairFilter.quoteMissing.length > 0) {
            filterSummaryItems.push({
                key: `${filterKey}-quote`,
                label: `${pairLabel} Quote USDT: ${quoteFilterSummary(activeReportPairFilter)}`,
                clear: () => clearFilterField(filterKey, {...activeReportPairFilter, quoteMin: '', quoteMax: '', quoteMissing: []})
            });
        }
    }
    const serverItems = data.data?.items || [];
    const items = activeReportPairKind ? sortReportRiskItems(serverItems, activeReportPairKind, riskSort, riskSortOrder) : serverItems;
    const pairViewLabel = (label: string, hasSaved: boolean) => (
        <span className='projects-view-label' aria-label={`${label}${hasSaved ? ', saved filters' : ''}`}>
            {label}
            {hasSaved && <span className='projects-view-label__dot' aria-hidden='true' />}
        </span>
    );
    const viewOptions = [
        {label: 'Overview', value: 'overview' as const},
        {label: 'Report Status', value: 'report-status' as const},
        {label: pairViewLabel('WETH / WBNB', wethFilterCount > 0), value: 'wrapped-native' as const},
        {label: pairViewLabel('USDT', usdtFilterCount > 0), value: 'usdt' as const}
    ];
    const tableLabel =
        view === 'overview'
            ? 'Project overview'
            : view === 'report-status'
              ? 'Project report status'
              : view === 'wrapped-native'
                ? 'Project WETH or WBNB pair risk'
                : 'Project USDT pair risk';
    const compactEmptyDescription =
        view === 'overview'
            ? 'No projects match the filters'
            : view === 'report-status'
              ? 'No projects match the report status filters'
              : `No projects match the ${view === 'wrapped-native' ? 'WETH / WBNB' : 'USDT'} pair risk filters`;
    const compactProject = (item: TokenProjectListItem) => {
        if (view === 'overview') {
            return <ProjectOverviewCard item={item} />;
        }
        if (view === 'report-status') {
            return <ProjectReportStatusCard item={item} />;
        }
        return <ProjectPairRiskCard item={item} kind={view === 'wrapped-native' ? 'weth' : 'usdt'} />;
    };
    return (
        <AppPage
            title='Projects'
            subtitle='Browse project identity and current report risk from one project read model.'
            loading={data.loading || data.refreshing || options.loading || options.refreshing}
            error={data.error || options.error}
            onRefresh={() => {
                data.reload();
                options.reload();
            }}
            filters={
                <div className='projects-controls'>
                    <div className='projects-controls__toolbar'>
                        <div className='projects-controls__selectors'>
                            <div className='projects-view-control'>
                                <Typography.Text strong={true}>View</Typography.Text>
                                <ChoiceGroup<ProjectsView>
                                    className='projects-view-control__desktop'
                                    ariaLabel='Projects view'
                                    value={view}
                                    options={viewOptions}
                                    onChange={setView}
                                />
                                <Select<ProjectsView> className='projects-view-control__compact' aria-label='Projects view' value={view} options={viewOptions} onChange={setView} />
                            </div>
                        </div>
                        <div className='projects-controls__actions'>
                            <Badge count={effectiveFilterCount} size='small' overflowCount={99}>
                                <Button
                                    aria-label={`Open filters${effectiveFilterCount ? `, ${effectiveFilterCount} active fields` : ''}`}
                                    icon={<FilterOutlined />}
                                    onClick={() => setFiltersOpen(true)}>
                                    Filters
                                </Button>
                            </Badge>
                            <Button disabled={!hasSavedFilters} onClick={clearFilters}>
                                Clear all
                            </Button>
                        </div>
                    </div>
                    {activeReportPairKind && (
                        <div className='projects-compact-sort-control' aria-label={`${pairLabel} current page sorting`}>
                            <Typography.Text strong={true}>Sort current page</Typography.Text>
                            <Select
                                aria-label='Sort field'
                                value={riskSort || 'default'}
                                options={reportRiskSortOptions}
                                onChange={value => (value === 'default' ? setRiskSort() : setRiskSort(value as ReportRiskSortKey, riskSortOrder || 'asc'))}
                            />
                            <Tooltip title='Ascending; missing values remain last'>
                                <Button
                                    aria-label='Sort current page ascending'
                                    aria-pressed={Boolean(riskSort && riskSortOrder === 'asc')}
                                    disabled={!riskSort}
                                    type={riskSort && riskSortOrder === 'asc' ? 'primary' : 'default'}
                                    icon={<SortAscendingOutlined />}
                                    onClick={() => setRiskSort(riskSort, 'asc')}
                                />
                            </Tooltip>
                            <Tooltip title='Descending; missing values remain last'>
                                <Button
                                    aria-label='Sort current page descending'
                                    aria-pressed={Boolean(riskSort && riskSortOrder === 'desc')}
                                    disabled={!riskSort}
                                    type={riskSort && riskSortOrder === 'desc' ? 'primary' : 'default'}
                                    icon={<SortDescendingOutlined />}
                                    onClick={() => setRiskSort(riskSort, 'desc')}
                                />
                            </Tooltip>
                        </div>
                    )}
                    {filterSummaryItems.length > 0 && (
                        <div className='projects-filter-summary' aria-label='Applied filters'>
                            <Typography.Text type='secondary'>Applied</Typography.Text>
                            <div className='projects-filter-summary__items'>
                                {filterSummaryItems.map(item => (
                                    <Tag
                                        key={item.key}
                                        closable={true}
                                        onClose={event => {
                                            event.preventDefault();
                                            item.clear();
                                        }}>
                                        <span className='projects-filter-summary__label' title={item.label}>
                                            {item.label}
                                        </span>
                                    </Tag>
                                ))}
                            </div>
                        </div>
                    )}
                    <ProjectsFiltersModal
                        open={filtersOpen}
                        filters={filterState}
                        activePairKind={activeReportPairKind}
                        chainOptions={chainOptions}
                        onCancel={() => setFiltersOpen(false)}
                        onApply={applyFilters}
                    />
                </div>
            }>
            <div className={reportProjection ? 'projects-report-risk-table-region' : undefined}>
                <ResourceTable
                    label={tableLabel}
                    rowKey='projectID'
                    items={items}
                    columns={viewColumns}
                    onChange={activeReportPairKind ? handleReportRiskTableChange : undefined}
                    loading={data.loading}
                    total={data.data?.total}
                    page={page}
                    pageSize={pageSize}
                    onPageChange={setPage}
                    scrollX={reportProjection ? 820 : 774}
                    stickyHeader={reportProjection}
                    compactRender={compactProject}
                    compactEmptyDescription={compactEmptyDescription}
                />
            </div>
        </AppPage>
    );
};
