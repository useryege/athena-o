import {FilterOutlined} from '@ant-design/icons';
import {Badge, Button, Card, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useSearchParams} from 'react-router-dom';
import {AppPage, KeyValueGrid, ResourceTable, StatusTag, useCachedAsyncData} from '../components';
import {formatBeijingDateTime, formatBeijingUnixSeconds} from '../shared/format';
import {DEFAULT_PAGE_SIZE, PAGE_SIZE_OPTIONS} from '../shared/pagination';
import {services} from '../shared/services';
import {TokenProjectListItem, TokenProjectReportPairRisk} from '../shared/services/token-service';
import {ProjectDetailLink, useRestoreProjectsScroll} from './project-navigation';
import {
    emptyProjectsFilterState,
    evaluationStatuses,
    generalFilterCount,
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

const PROJECTS_LIST_STALE_TIME_MS = 30_000;
const RUNTIME_CONFIGURATION_STALE_TIME_MS = 5 * 60_000;
const RUNTIME_CONFIGURATION_CACHE_KEY = 'token.runtime-configuration';
const PROJECTS_TABLE_WIDTH = 2584;

const positiveIntegerParam = (value: string | null) => {
    const parsed = Number(value);
    return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : undefined;
};

const enumParam = (value: string | null, allowed: readonly string[]) => (value && allowed.includes(value) ? value : '');

interface ProjectsQueryState extends ProjectsFilterState {
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

const pairColumnGroup = (title: string, pair: (item: TokenProjectListItem) => TokenProjectReportPairRisk | undefined): ColumnsType<TokenProjectListItem>[number] => ({
    title,
    children: [
        {
            title: 'Created',
            width: 95,
            render: item => {
                if (!item.currentReport) {
                    return <Tag>No report</Tag>;
                }
                const pairRisk = pair(item);
                return pairRisk ? pairCreatedTag(pairRisk.isCreated) : unavailablePairValue(false);
            }
        },
        {
            title: 'Remove Liquidity',
            width: 130,
            render: item => {
                const pairRisk = pair(item);
                return !item.currentReport ? <Tag>No report</Tag> : pairRisk ? pairRiskTag(pairRisk.isRemoveLiquidity) : unavailablePairValue(false);
            }
        },
        {
            title: 'Mint',
            width: 85,
            render: item => {
                const pairRisk = pair(item);
                return !item.currentReport ? <Tag>No report</Tag> : pairRisk ? pairRiskTag(pairRisk.isMint) : unavailablePairValue(false);
            }
        },
        {
            title: 'Quote USDT',
            width: 120,
            render: item => (pair(item)?.quoteUsdtValueInt ? formatInteger(pair(item)?.quoteUsdtValueInt) : unavailablePairValue(!item.currentReport))
        },
        {
            title: 'Last Swap',
            width: 155,
            render: item => (pair(item)?.lastSwapAt ? formatBeijingDateTime(pair(item)?.lastSwapAt) || '-' : unavailablePairValue(!item.currentReport))
        }
    ]
});

const projectColumns: ColumnsType<TokenProjectListItem> = [
    {
        title: 'Logo',
        fixed: 'left',
        width: 64,
        align: 'center',
        render: item => <TokenLogo logoURL={item.logoURL} symbol={item.symbol} />
    },
    {title: 'Project', fixed: 'left', width: 220, render: projectIdentity},
    {
        title: 'Overview',
        children: [
            {title: 'Chain', width: 120, render: item => <ChainBadge chainID={item.chainID} />},
            {title: 'Block Time', width: 185, render: item => formatBeijingUnixSeconds(item.blockTime) || '-'},
            {title: 'Created', width: 185, render: item => formatBeijingDateTime(item.createdAt) || '-'}
        ]
    },
    {
        title: 'Report Status',
        children: [
            {title: 'Research', width: 100, render: item => researchTag(item.researchStatus)},
            {
                title: 'Report',
                width: 160,
                render: item =>
                    reportRiskSummary([
                        {label: 'Revision', value: item.currentReport?.revision ?? <Tag>No report</Tag>},
                        {label: 'State', value: reportStateTag(item.currentReport?.completenessStatus)},
                        {label: 'Built', value: formatBeijingDateTime(item.currentReport?.builtAt) || (item.currentReport ? '-' : 'No report')}
                    ])
            },
            {
                title: 'Evaluation',
                width: 220,
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
                width: 160,
                render: item =>
                    reportRiskSummary([
                        {label: 'Outcome', value: outcomeTag(item.currentReport?.evaluation?.outcome)},
                        {label: 'Evaluated', value: formatBeijingDateTime(item.currentReport?.evaluation?.evaluatedAt) || '-'}
                    ])
            }
        ]
    },
    pairColumnGroup('WETH / WBNB', item => item.currentReport?.riskSummary?.wethPair),
    pairColumnGroup('USDT', item => item.currentReport?.riskSummary?.usdtPair)
];

const ProjectUnifiedCard = (props: {item: TokenProjectListItem}) => {
    const report = props.item.currentReport;
    const evaluation = report?.evaluation;
    const wethPair = report?.riskSummary?.wethPair;
    const usdtPair = report?.riskSummary?.usdtPair;
    return (
        <Card
            className='projects-compact-card projects-unified-card'
            size='small'
            title={
                <span className='projects-compact-card__title'>
                    <TokenLogo logoURL={props.item.logoURL} symbol={props.item.symbol} />
                    <span>{props.item.symbol || props.item.name || 'Unnamed token'}</span>
                    <ChainBadge chainID={props.item.chainID} />
                </span>
            }
            extra={projectDetailLink(props.item)}>
            <div className='projects-unified-card__sections'>
                <section aria-label='Project overview'>
                    <Typography.Title level={5}>Overview</Typography.Title>
                    <KeyValueGrid
                        columns={2}
                        items={[
                            {label: 'Project ID', value: props.item.projectID || '-'},
                            {label: 'Token', value: tokenValue(props.item)},
                            {label: 'Block Time', value: formatBeijingUnixSeconds(props.item.blockTime) || '-'},
                            {label: 'Created', value: formatBeijingDateTime(props.item.createdAt) || '-'}
                        ]}
                    />
                </section>
                <section aria-label='Current report status'>
                    <Typography.Title level={5}>Report Status</Typography.Title>
                    <KeyValueGrid
                        columns={2}
                        items={[
                            {label: 'Research', value: researchTag(props.item.researchStatus)},
                            {label: 'Report revision', value: report?.revision ?? '-'},
                            {label: 'Report state', value: reportStateTag(report?.completenessStatus)},
                            {label: 'Built', value: formatBeijingDateTime(report?.builtAt) || '-'},
                            {label: 'Evaluation', value: evaluationTag(evaluation?.status)},
                            {label: 'Failed attempts', value: evaluation?.failedAttempts ?? '-'},
                            {label: 'Evaluation updated', value: formatBeijingDateTime(evaluation?.updatedAt) || '-'},
                            {label: 'Last error', value: evaluationLastError(evaluation?.lastError)},
                            {label: 'Selection', value: outcomeTag(evaluation?.outcome)},
                            {label: 'Evaluated', value: formatBeijingDateTime(evaluation?.evaluatedAt) || '-'}
                        ]}
                    />
                </section>
                <div className='projects-unified-card__pairs'>
                    <section className='projects-unified-card__pair' aria-label='WETH or WBNB pair report snapshot'>
                        <Typography.Title level={5}>WETH / WBNB</Typography.Title>
                        <Typography.Text type='secondary'>{!report ? 'No report' : !wethPair ? 'Risk unavailable' : 'Report snapshot'}</Typography.Text>
                        <KeyValueGrid columns={1} items={pairSnapshotItems(wethPair, !report)} />
                    </section>
                    <section className='projects-unified-card__pair' aria-label='USDT pair report snapshot'>
                        <Typography.Title level={5}>USDT</Typography.Title>
                        <Typography.Text type='secondary'>{!report ? 'No report' : !usdtPair ? 'Risk unavailable' : 'Report snapshot'}</Typography.Text>
                        <KeyValueGrid columns={1} items={pairSnapshotItems(usdtPair, !report)} />
                    </section>
                </div>
            </div>
        </Card>
    );
};

export const ProjectsPage = () => {
    const [params, setParams] = useSearchParams();
    const [filtersOpen, setFiltersOpen] = React.useState(false);
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
    const page = positiveIntegerParam(params.get('page')) || 1;
    const requestedPageSize = positiveIntegerParam(params.get('pageSize')) || DEFAULT_PAGE_SIZE;
    const pageSize = PAGE_SIZE_OPTIONS.includes(requestedPageSize) ? requestedPageSize : DEFAULT_PAGE_SIZE;

    const queryState = React.useMemo<ProjectsQueryState>(
        () => ({
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
            page,
            pageSize
        }),
        [chainID, projectID, contract, codeHash, researchStatus, reportState, evaluationStatus, selectionOutcome, wethPairFilter, usdtPairFilter, page, pageSize]
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

    const setPage = (nextPage: number, nextPageSize: number) => {
        setParams(serializeQueryState({...queryState, page: nextPage, pageSize: nextPageSize}));
    };
    const applyFilters = (nextFilters: ProjectsFilterState) => {
        setFiltersOpen(false);
        setParams(serializeQueryState({...queryState, ...nextFilters, page: 1}));
    };
    const clearFilters = () => {
        setParams(serializeQueryState({...queryState, ...emptyProjectsFilterState(), page: 1}));
    };

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
        wethPairFilter.removeLiquidity,
        wethPairFilter.mint,
        wethPairFilter.quoteMin || null,
        wethPairFilter.quoteMax || null,
        wethPairFilter.quoteMissing,
        usdtPairFilter.removeLiquidity,
        usdtPairFilter.mint,
        usdtPairFilter.quoteMin || null,
        usdtPairFilter.quoteMax || null,
        usdtPairFilter.quoteMissing
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
                wethPairRemoveLiquidityStates: wethPairFilter.removeLiquidity,
                wethPairMintStates: wethPairFilter.mint,
                wethPairQuoteUSDTMin: wethPairFilter.quoteMin || undefined,
                wethPairQuoteUSDTMax: wethPairFilter.quoteMax || undefined,
                wethPairQuoteMissingStates: wethPairFilter.quoteMissing,
                usdtPairRemoveLiquidityStates: usdtPairFilter.removeLiquidity,
                usdtPairMintStates: usdtPairFilter.mint,
                usdtPairQuoteUSDTMin: usdtPairFilter.quoteMin || undefined,
                usdtPairQuoteUSDTMax: usdtPairFilter.quoteMax || undefined,
                usdtPairQuoteMissingStates: usdtPairFilter.quoteMissing
            }),
        {staleTimeMs: PROJECTS_LIST_STALE_TIME_MS}
    );
    useRestoreProjectsScroll(Boolean(data.data));

    const chainOptions = (options.data?.chains || [])
        .filter((item): item is {chainID: number; chainName?: string} => item.chainID !== undefined)
        .map(item => ({value: item.chainID, label: chainLabel(item.chainID)}));
    const generalCount = generalFilterCount(filterState);
    const wethFilterCount = reportPairFilterCount(wethPairFilter);
    const usdtFilterCount = reportPairFilterCount(usdtPairFilter);
    const activeFilterCount = generalCount + wethFilterCount + usdtFilterCount;
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
    const appendPairFilterSummaries = (filterKey: 'wethPairFilter' | 'usdtPairFilter', label: string, filter: ReportPairFilterState) => {
        if (filter.removeLiquidity.length > 0) {
            filterSummaryItems.push({
                key: `${filterKey}-removeLiquidity`,
                label: `${label} Remove Liquidity: ${summarizeStates(filter.removeLiquidity)}`,
                clear: () => clearFilterField(filterKey, {...filter, removeLiquidity: []})
            });
        }
        if (filter.mint.length > 0) {
            filterSummaryItems.push({
                key: `${filterKey}-mint`,
                label: `${label} Mint: ${summarizeStates(filter.mint)}`,
                clear: () => clearFilterField(filterKey, {...filter, mint: []})
            });
        }
        if (filter.quoteMin || filter.quoteMax || filter.quoteMissing.length > 0) {
            filterSummaryItems.push({
                key: `${filterKey}-quote`,
                label: `${label} Quote USDT: ${quoteFilterSummary(filter)}`,
                clear: () => clearFilterField(filterKey, {...filter, quoteMin: '', quoteMax: '', quoteMissing: []})
            });
        }
    };
    appendPairFilterSummaries('wethPairFilter', 'WETH / WBNB', wethPairFilter);
    appendPairFilterSummaries('usdtPairFilter', 'USDT', usdtPairFilter);

    return (
        <AppPage
            title='Projects'
            subtitle='Browse project identity, report status, and both Pair risk snapshots in one unified list.'
            loading={data.loading || data.refreshing || options.loading || options.refreshing}
            error={data.error || options.error}
            onRefresh={() => {
                data.reload();
                options.reload();
            }}
            filters={
                <div className='projects-controls'>
                    <div className='projects-controls__toolbar'>
                        <div className='projects-controls__actions'>
                            <Badge count={activeFilterCount} size='small' overflowCount={99}>
                                <Button
                                    aria-label={`Open filters${activeFilterCount ? `, ${activeFilterCount} active fields` : ''}`}
                                    icon={<FilterOutlined />}
                                    onClick={() => setFiltersOpen(true)}>
                                    Filters
                                </Button>
                            </Badge>
                            <Button disabled={!activeFilterCount} onClick={clearFilters}>
                                Clear all
                            </Button>
                        </div>
                    </div>
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
                    <ProjectsFiltersModal open={filtersOpen} filters={filterState} chainOptions={chainOptions} onCancel={() => setFiltersOpen(false)} onApply={applyFilters} />
                </div>
            }>
            <div className='projects-unified-table-region'>
                <ResourceTable
                    label='Project overview, report status, and Pair risk'
                    rowKey='projectID'
                    items={data.data?.items || []}
                    columns={projectColumns}
                    loading={data.loading}
                    total={data.data?.total}
                    page={page}
                    pageSize={pageSize}
                    onPageChange={setPage}
                    scrollX={PROJECTS_TABLE_WIDTH}
                    stickyHeader={true}
                    compactRender={item => <ProjectUnifiedCard item={item} />}
                    compactEmptyDescription='No projects match the filters'
                />
            </div>
        </AppPage>
    );
};
