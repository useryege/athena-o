import {FilterFilled} from '@ant-design/icons';
import {Button, Card, Checkbox, Input, InputNumber, Select, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import type {FilterDropdownProps} from 'antd/es/table/interface';
import * as React from 'react';
import {useSearchParams} from 'react-router-dom';
import {AppPage, ChoiceGroup, KeyValueGrid, ResourceTable, SearchBar, StatusTag, TruncatedText, useCachedAsyncData} from '../components';
import {formatBeijingDateTime, formatBlockNumber} from '../shared/format';
import {DEFAULT_PAGE_SIZE, PAGE_SIZE_OPTIONS} from '../shared/pagination';
import {services} from '../shared/services';
import {TokenProjectListItem, TokenProjectReportPairRisk} from '../shared/services/token-service';
import {ProjectDetailLink, useRestoreProjectsScroll} from './project-navigation';
import {ChainBadge, TokenLogo, chainLabel} from './token-shared';

type ProjectsView = 'overview' | 'report-risk';
type ReportRiskSection = 'status' | 'wrapped-native' | 'usdt';
type ReportPairKind = 'weth' | 'usdt';
type ReportPairRiskState = 'detected' | 'clear' | 'no_report' | 'risk_unavailable';
type ReportPairMissingState = 'no_report' | 'risk_unavailable';
type ProjectsFilterKey = 'chainID' | 'projectID' | 'contract' | 'codeHash' | 'researchStatus' | 'reportState' | 'evaluationStatus' | 'selectionOutcome';

const researchStatuses = ['researching', 'selected', 'rejected', 'expired'] as const;
const reportStates = ['none', 'incomplete', 'complete'] as const;
const evaluationStatuses = ['none', 'pending', 'running', 'succeeded', 'failed'] as const;
const selectionOutcomes = ['none', 'selected', 'rejected', 'deferred'] as const;
const reportRiskSections = ['status', 'wrapped-native', 'usdt'] as const;
const reportPairRiskStates = ['detected', 'clear', 'no_report', 'risk_unavailable'] as const;
const reportPairMissingStates = ['no_report', 'risk_unavailable'] as const;
const reportPairRiskStateOptions = [
    {label: 'Detected', value: 'detected'},
    {label: 'Clear', value: 'clear'},
    {label: 'No report', value: 'no_report'},
    {label: 'Risk unavailable', value: 'risk_unavailable'}
];
const reportPairMissingStateOptions = [
    {label: 'No report', value: 'no_report'},
    {label: 'Risk unavailable', value: 'risk_unavailable'}
];
const PROJECTS_LIST_STALE_TIME_MS = 30_000;
const RUNTIME_CONFIGURATION_STALE_TIME_MS = 5 * 60_000;
const RUNTIME_CONFIGURATION_CACHE_KEY = 'token.runtime-configuration';

const positiveIntegerParam = (value: string | null) => {
    const parsed = Number(value);
    return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : undefined;
};

const enumParam = (value: string | null, allowed: readonly string[]) => (value && allowed.includes(value) ? value : '');

interface ProjectsQueryState {
    view: ProjectsView;
    riskSection: ReportRiskSection;
    chainID?: number;
    projectID?: number;
    contract: string;
    codeHash: string;
    researchStatus: string;
    reportState: string;
    evaluationStatus: string;
    selectionOutcome: string;
    wethPairFilter: ReportPairFilterState;
    usdtPairFilter: ReportPairFilterState;
    page: number;
    pageSize: number;
}

interface ReportPairFilterState {
    removeLiquidity: ReportPairRiskState[];
    mint: ReportPairRiskState[];
    quoteMin: string;
    quoteMax: string;
    quoteMissing: ReportPairMissingState[];
}

const emptyReportPairFilter = (): ReportPairFilterState => ({removeLiquidity: [], mint: [], quoteMin: '', quoteMax: '', quoteMissing: []});

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

const hasReportPairFilter = (filter: ReportPairFilterState) =>
    filter.removeLiquidity.length > 0 || filter.mint.length > 0 || Boolean(filter.quoteMin || filter.quoteMax) || filter.quoteMissing.length > 0;

const serializeQueryState = (state: ProjectsQueryState) => {
    const next = new URLSearchParams();
    if (state.view === 'report-risk') {
        next.set('view', state.view);
        if (state.riskSection !== 'status') {
            next.set('riskSection', state.riskSection);
        }
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
    item.project?.projectID ? <ProjectDetailLink projectID={item.project.projectID} /> : <Typography.Text type='secondary'>Unavailable</Typography.Text>;

const tokenValue = (item: TokenProjectListItem) => (
    <span className='projects-token'>
        <Typography.Text strong={true}>{item.project?.symbol || '-'}</Typography.Text>
        <Typography.Text type='secondary'>{item.project?.name || '-'}</Typography.Text>
    </span>
);

const projectIdentity = (item: TokenProjectListItem) => (
    <span className='projects-token'>
        <Typography.Text strong={true}>{item.project?.symbol || item.project?.name || 'Unnamed token'}</Typography.Text>
        <Typography.Text type='secondary'>Project #{item.project?.projectID || '-'}</Typography.Text>
        {item.project?.name && item.project.name !== item.project.symbol && <Typography.Text type='secondary'>{item.project.name}</Typography.Text>}
        {projectDetailLink(item)}
    </span>
);

const reportProjectIdentity = (item: TokenProjectListItem) => (
    <span className='projects-token'>
        {projectIdentity(item)}
        <ChainBadge chainID={item.project?.chainID} />
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
            <div className='projects-report-risk-summary__line'>
                <Typography.Text className='projects-report-risk-summary__label' type='secondary'>
                    Contract
                </Typography.Text>
                <div className='projects-report-risk-summary__value'>
                    <TruncatedText value={item.project?.contract} copyable={true} singleLine={true} />
                </div>
            </div>
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

const normalizeUnsignedInteger = (value: string) => (value ? BigInt(value).toString() : '');

const quoteFilterError = (minimum: string, maximum: string) => {
    if (minimum && !/^\d+$/.test(minimum)) {
        return 'Minimum must be a non-negative integer.';
    }
    if (maximum && !/^\d+$/.test(maximum)) {
        return 'Maximum must be a non-negative integer.';
    }
    if (minimum && maximum && BigInt(minimum) > BigInt(maximum)) {
        return 'Minimum must not exceed maximum.';
    }
    return '';
};

const pairFilterIcon = (label: string) => (filtered: boolean) => (
    <FilterFilled aria-label={`${label} filter${filtered ? ' active' : ''}`} style={{color: filtered ? 'var(--athena-blue)' : undefined}} />
);

const ReportPairStateFilterDropdown = (props: {
    label: string;
    selected: ReportPairRiskState[];
    onApply: (states: ReportPairRiskState[]) => void;
    close: FilterDropdownProps['close'];
}) => {
    const [draft, setDraft] = React.useState<ReportPairRiskState[]>(props.selected);
    const selectedKey = props.selected.join(',');
    React.useEffect(() => setDraft(props.selected), [selectedKey]);
    return (
        <fieldset className='projects-pair-filter-dropdown' onKeyDown={event => event.stopPropagation()}>
            <legend>{props.label}</legend>
            <Checkbox.Group
                aria-label={`${props.label} states`}
                className='projects-pair-filter-dropdown__choices'
                options={reportPairRiskStateOptions}
                value={draft}
                onChange={values => setDraft(values as ReportPairRiskState[])}
            />
            <div className='projects-pair-filter-dropdown__actions'>
                <Button
                    type='primary'
                    size='small'
                    onClick={() => {
                        props.onApply(draft);
                        props.close();
                    }}>
                    Apply
                </Button>
                <Button
                    size='small'
                    onClick={() => {
                        setDraft([]);
                        props.onApply([]);
                        props.close();
                    }}>
                    Clear
                </Button>
            </div>
        </fieldset>
    );
};

const ReportPairQuoteFilterDropdown = (props: {
    selected: Pick<ReportPairFilterState, 'quoteMin' | 'quoteMax' | 'quoteMissing'>;
    onApply: (filter: Pick<ReportPairFilterState, 'quoteMin' | 'quoteMax' | 'quoteMissing'>) => void;
    close: FilterDropdownProps['close'];
}) => {
    const [minimum, setMinimum] = React.useState(props.selected.quoteMin);
    const [maximum, setMaximum] = React.useState(props.selected.quoteMax);
    const [missing, setMissing] = React.useState<ReportPairMissingState[]>(props.selected.quoteMissing);
    const selectedKey = `${props.selected.quoteMin}|${props.selected.quoteMax}|${props.selected.quoteMissing.join(',')}`;
    React.useEffect(() => {
        setMinimum(props.selected.quoteMin);
        setMaximum(props.selected.quoteMax);
        setMissing(props.selected.quoteMissing);
    }, [selectedKey]);
    const error = quoteFilterError(minimum, maximum);
    const errorID = React.useId();
    return (
        <fieldset className='projects-pair-filter-dropdown projects-pair-filter-dropdown--quote' onKeyDown={event => event.stopPropagation()}>
            <legend>Quote USDT</legend>
            <label>
                <span>Minimum</span>
                <Input
                    aria-describedby={error ? errorID : undefined}
                    aria-invalid={Boolean(error)}
                    inputMode='numeric'
                    pattern='[0-9]*'
                    placeholder='No minimum'
                    value={minimum}
                    onChange={event => setMinimum(event.target.value.trim())}
                />
            </label>
            <label>
                <span>Maximum</span>
                <Input
                    aria-describedby={error ? errorID : undefined}
                    aria-invalid={Boolean(error)}
                    inputMode='numeric'
                    pattern='[0-9]*'
                    placeholder='No maximum'
                    value={maximum}
                    onChange={event => setMaximum(event.target.value.trim())}
                />
            </label>
            <Typography.Text strong={true}>Include missing</Typography.Text>
            <Checkbox.Group
                aria-label='Quote USDT missing states'
                className='projects-pair-filter-dropdown__choices'
                options={reportPairMissingStateOptions}
                value={missing}
                onChange={values => setMissing(values as ReportPairMissingState[])}
            />
            {error && (
                <Typography.Text id={errorID} type='danger' role='alert'>
                    {error}
                </Typography.Text>
            )}
            <div className='projects-pair-filter-dropdown__actions'>
                <Button
                    type='primary'
                    size='small'
                    disabled={Boolean(error)}
                    onClick={() => {
                        props.onApply({quoteMin: normalizeUnsignedInteger(minimum), quoteMax: normalizeUnsignedInteger(maximum), quoteMissing: missing});
                        props.close();
                    }}>
                    Apply
                </Button>
                <Button
                    size='small'
                    onClick={() => {
                        setMinimum('');
                        setMaximum('');
                        setMissing([]);
                        props.onApply({quoteMin: '', quoteMax: '', quoteMissing: []});
                        props.close();
                    }}>
                    Clear
                </Button>
            </div>
        </fieldset>
    );
};

const CompactReportPairFilters = (props: {pairLabel: string; filter: ReportPairFilterState; onApply: (filter: ReportPairFilterState) => void}) => {
    const [draft, setDraft] = React.useState(props.filter);
    const selectedKey = [
        props.filter.removeLiquidity.join(','),
        props.filter.mint.join(','),
        props.filter.quoteMin,
        props.filter.quoteMax,
        props.filter.quoteMissing.join(',')
    ].join('|');
    React.useEffect(() => setDraft(props.filter), [selectedKey]);
    const error = quoteFilterError(draft.quoteMin, draft.quoteMax);
    const errorID = React.useId();
    return (
        <fieldset className='projects-compact-pair-filters'>
            <legend>{props.pairLabel} Pair filters</legend>
            <Typography.Text type='secondary'>Compact cards still show Status and both Pair snapshots. These controls choose which projects are included.</Typography.Text>
            <div className='projects-compact-pair-filters__grid'>
                <label>
                    <span>Remove Liquidity</span>
                    <Select
                        aria-label={`Filter ${props.pairLabel} Remove Liquidity`}
                        mode='multiple'
                        allowClear={true}
                        placeholder='All states'
                        options={reportPairRiskStateOptions}
                        value={draft.removeLiquidity}
                        onChange={values => setDraft({...draft, removeLiquidity: values as ReportPairRiskState[]})}
                    />
                </label>
                <label>
                    <span>Mint</span>
                    <Select
                        aria-label={`Filter ${props.pairLabel} Mint`}
                        mode='multiple'
                        allowClear={true}
                        placeholder='All states'
                        options={reportPairRiskStateOptions}
                        value={draft.mint}
                        onChange={values => setDraft({...draft, mint: values as ReportPairRiskState[]})}
                    />
                </label>
                <label>
                    <span>Quote minimum</span>
                    <Input
                        aria-describedby={error ? errorID : undefined}
                        aria-invalid={Boolean(error)}
                        inputMode='numeric'
                        pattern='[0-9]*'
                        placeholder='No minimum'
                        value={draft.quoteMin}
                        onChange={event => setDraft({...draft, quoteMin: event.target.value.trim()})}
                    />
                </label>
                <label>
                    <span>Quote maximum</span>
                    <Input
                        aria-describedby={error ? errorID : undefined}
                        aria-invalid={Boolean(error)}
                        inputMode='numeric'
                        pattern='[0-9]*'
                        placeholder='No maximum'
                        value={draft.quoteMax}
                        onChange={event => setDraft({...draft, quoteMax: event.target.value.trim()})}
                    />
                </label>
                <label>
                    <span>Quote missing</span>
                    <Select
                        aria-label={`Filter ${props.pairLabel} Quote USDT missing states`}
                        mode='multiple'
                        allowClear={true}
                        placeholder='Do not include'
                        options={reportPairMissingStateOptions}
                        value={draft.quoteMissing}
                        onChange={values => setDraft({...draft, quoteMissing: values as ReportPairMissingState[]})}
                    />
                </label>
            </div>
            {error && (
                <Typography.Text id={errorID} type='danger' role='alert'>
                    {error}
                </Typography.Text>
            )}
            <div className='projects-compact-pair-filters__actions'>
                <Button
                    type='primary'
                    disabled={Boolean(error)}
                    onClick={() =>
                        props.onApply({
                            ...draft,
                            quoteMin: normalizeUnsignedInteger(draft.quoteMin),
                            quoteMax: normalizeUnsignedInteger(draft.quoteMax)
                        })
                    }>
                    Apply Pair filters
                </Button>
                <Button
                    disabled={!hasReportPairFilter(props.filter)}
                    onClick={() => {
                        const empty = emptyReportPairFilter();
                        setDraft(empty);
                        props.onApply(empty);
                    }}>
                    Clear Pair filters
                </Button>
            </div>
        </fieldset>
    );
};

const ProjectOverviewCard = (props: {item: TokenProjectListItem}) => {
    const project = props.item.project;
    return (
        <Card
            className='projects-compact-card'
            size='small'
            title={
                <span className='projects-compact-card__title'>
                    <TokenLogo logoURL={props.item.logoURL} symbol={project?.symbol} />
                    <span>{project?.symbol || project?.name || 'Unnamed token'}</span>
                    <ChainBadge chainID={project?.chainID} />
                </span>
            }
            extra={projectDetailLink(props.item)}>
            <KeyValueGrid
                columns={2}
                items={[
                    {label: 'Project', value: projectIdentity(props.item)},
                    {label: 'Chain', value: <ChainBadge chainID={project?.chainID} />},
                    {label: 'Contract', value: <TruncatedText value={project?.contract} copyable={true} />},
                    {label: 'Tx sender', value: <TruncatedText value={project?.txSender} copyable={true} />},
                    {label: 'Block', value: formatBlockNumber(project?.blockNumber)},
                    {label: 'Created', value: formatBeijingDateTime(project?.createdAt) || '-'}
                ]}
            />
        </Card>
    );
};

const ProjectReportRiskCard = (props: {item: TokenProjectListItem}) => {
    const project = props.item.project;
    const report = props.item.currentReport;
    const evaluation = report?.evaluation;
    return (
        <Card
            className='projects-compact-card projects-report-risk-card'
            size='small'
            title={
                <span className='projects-compact-card__title'>
                    <TokenLogo logoURL={props.item.logoURL} symbol={project?.symbol} />
                    <span>{project?.symbol || project?.name || 'Unnamed token'}</span>
                    <ChainBadge chainID={project?.chainID} />
                </span>
            }
            extra={projectDetailLink(props.item)}>
            <div className='projects-report-risk-card__sections'>
                <section aria-label='Project and report state'>
                    <Typography.Title level={5}>Project & report</Typography.Title>
                    <KeyValueGrid
                        columns={2}
                        items={[
                            {label: 'Project ID', value: project?.projectID || '-'},
                            {label: 'Token', value: tokenValue(props.item)},
                            {label: 'Chain', value: <ChainBadge chainID={project?.chainID} />},
                            {label: 'Contract', value: <TruncatedText value={project?.contract} copyable={true} />},
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
                <div className='projects-report-risk-card__pair-grid'>
                    <section aria-label='WETH or WBNB pair report snapshot'>
                        <Typography.Title level={5}>WETH / WBNB pair</Typography.Title>
                        <Typography.Text type='secondary'>{!report ? 'No report' : !report.riskSummary?.wethPair ? 'Risk unavailable' : 'Report snapshot'}</Typography.Text>
                        <KeyValueGrid columns={1} items={pairSnapshotItems(report?.riskSummary?.wethPair, !report)} />
                    </section>
                    <section aria-label='USDT pair report snapshot'>
                        <Typography.Title level={5}>USDT pair</Typography.Title>
                        <Typography.Text type='secondary'>{!report ? 'No report' : !report.riskSummary?.usdtPair ? 'Risk unavailable' : 'Report snapshot'}</Typography.Text>
                        <KeyValueGrid columns={1} items={pairSnapshotItems(report?.riskSummary?.usdtPair, !report)} />
                    </section>
                </div>
            </div>
        </Card>
    );
};

export const ProjectsPage = () => {
    const [params, setParams] = useSearchParams();
    const view: ProjectsView = params.get('view') === 'report-risk' ? 'report-risk' : 'overview';
    const reportRiskView = view === 'report-risk';
    const riskSection = (enumParam(params.get('riskSection'), reportRiskSections) || 'status') as ReportRiskSection;
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
            view,
            riskSection,
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
        [view, riskSection, chainID, projectID, contract, codeHash, researchStatus, reportState, evaluationStatus, selectionOutcome, wethPairFilter, usdtPairFilter, page, pageSize]
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
    const setRiskSection = (nextRiskSection: ReportRiskSection) => {
        setParams(serializeQueryState({...queryState, riskSection: nextRiskSection}));
    };
    const setFilter = (key: ProjectsFilterKey, value?: string | number) => {
        const nextValue = value === undefined || value === '' ? undefined : value;
        const next = {...queryState, page: 1, [key]: nextValue} as ProjectsQueryState;
        setParams(serializeQueryState(next));
    };
    const setPage = (nextPage: number, nextPageSize: number) => {
        setParams(serializeQueryState({...queryState, page: nextPage, pageSize: nextPageSize}));
    };
    const setReportPairFilter = (kind: ReportPairKind, filter: ReportPairFilterState) => {
        setParams(
            serializeQueryState({
                ...queryState,
                page: 1,
                wethPairFilter: kind === 'weth' ? filter : queryState.wethPairFilter,
                usdtPairFilter: kind === 'usdt' ? filter : queryState.usdtPairFilter
            })
        );
    };
    const clearFilters = () => {
        setParams(
            serializeQueryState({
                view,
                riskSection,
                contract: '',
                codeHash: '',
                researchStatus: '',
                reportState: '',
                evaluationStatus: '',
                selectionOutcome: '',
                wethPairFilter: emptyReportPairFilter(),
                usdtPairFilter: emptyReportPairFilter(),
                page: 1,
                pageSize
            })
        );
    };

    const activeReportPairKind: ReportPairKind | undefined =
        reportRiskView && riskSection === 'wrapped-native' ? 'weth' : reportRiskView && riskSection === 'usdt' ? 'usdt' : undefined;
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

    const logoColumn: ColumnsType<TokenProjectListItem>[number] = {
        title: 'Logo',
        fixed: 'left',
        width: 64,
        align: 'center',
        render: item => <TokenLogo logoURL={item.logoURL} symbol={item.project?.symbol} />
    };

    const overviewColumns: ColumnsType<TokenProjectListItem> = [
        logoColumn,
        {title: 'Project', fixed: 'left', width: 220, render: projectIdentity},
        {title: 'Chain', width: 120, render: item => <ChainBadge chainID={item.project?.chainID} />},
        {title: 'Contract', width: 250, render: item => <TruncatedText value={item.project?.contract} copyable={true} />},
        {title: 'Tx Sender', width: 240, render: item => <TruncatedText value={item.project?.txSender} copyable={true} />},
        {title: 'Block', width: 130, render: item => formatBlockNumber(item.project?.blockNumber)},
        {title: 'Created', width: 185, render: item => formatBeijingDateTime(item.project?.createdAt) || '-'}
    ];

    const pairColumns = (
        pair: (item: TokenProjectListItem) => TokenProjectReportPairRisk | undefined,
        kind: ReportPairKind,
        filter: ReportPairFilterState
    ): ColumnsType<TokenProjectListItem> => [
        {title: 'Created', width: 90, render: item => (item.currentReport ? pairCreatedTag(pair(item)?.isCreated) : <Tag>No report</Tag>)},
        {
            title: 'Remove Liquidity',
            key: 'removeLiquidity',
            width: 125,
            filteredValue: filter.removeLiquidity.length > 0 ? filter.removeLiquidity : null,
            filterOnClose: false,
            filterIcon: pairFilterIcon('Remove Liquidity'),
            filterDropdown: ({close}) => (
                <ReportPairStateFilterDropdown
                    label='Remove Liquidity'
                    selected={filter.removeLiquidity}
                    close={close}
                    onApply={states => setReportPairFilter(kind, {...filter, removeLiquidity: states})}
                />
            ),
            render: item => {
                const pairRisk = pair(item);
                return !item.currentReport ? <Tag>No report</Tag> : pairRisk ? pairRiskTag(pairRisk.isRemoveLiquidity) : unavailablePairValue(false);
            }
        },
        {
            title: 'Mint',
            key: 'mint',
            width: 80,
            filteredValue: filter.mint.length > 0 ? filter.mint : null,
            filterOnClose: false,
            filterIcon: pairFilterIcon('Mint'),
            filterDropdown: ({close}) => (
                <ReportPairStateFilterDropdown label='Mint' selected={filter.mint} close={close} onApply={states => setReportPairFilter(kind, {...filter, mint: states})} />
            ),
            render: item => {
                const pairRisk = pair(item);
                return !item.currentReport ? <Tag>No report</Tag> : pairRisk ? pairRiskTag(pairRisk.isMint) : unavailablePairValue(false);
            }
        },
        {
            title: 'Quote USDT',
            key: 'quoteUsdt',
            width: 110,
            filteredValue: filter.quoteMin || filter.quoteMax || filter.quoteMissing.length > 0 ? ['active'] : null,
            filterOnClose: false,
            filterIcon: pairFilterIcon('Quote USDT'),
            filterDropdown: ({close}) => <ReportPairQuoteFilterDropdown selected={filter} close={close} onApply={next => setReportPairFilter(kind, {...filter, ...next})} />,
            render: item => (pair(item)?.quoteUsdtValueInt ? formatInteger(pair(item)?.quoteUsdtValueInt) : unavailablePairValue(!item.currentReport))
        },
        {
            title: 'Last Swap',
            width: 145,
            render: item => (pair(item)?.lastSwapAt ? formatBeijingDateTime(pair(item)?.lastSwapAt) || '-' : unavailablePairValue(!item.currentReport))
        }
    ];

    const reportRiskStatusColumns: ColumnsType<TokenProjectListItem> = [
        logoColumn,
        {title: 'Project & Contract', width: 200, render: item => reportRiskProjectContext(item, false)},
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

    const reportRiskPairColumns = (
        pair: (item: TokenProjectListItem) => TokenProjectReportPairRisk | undefined,
        kind: ReportPairKind,
        filter: ReportPairFilterState
    ): ColumnsType<TokenProjectListItem> => [
        logoColumn,
        {title: 'Project & Contract', width: 200, render: item => reportRiskProjectContext(item, true)},
        ...pairColumns(pair, kind, filter)
    ];

    const reportRiskColumns =
        riskSection === 'status'
            ? reportRiskStatusColumns
            : riskSection === 'wrapped-native'
              ? reportRiskPairColumns(item => item.currentReport?.riskSummary?.wethPair, 'weth', wethPairFilter)
              : reportRiskPairColumns(item => item.currentReport?.riskSummary?.usdtPair, 'usdt', usdtPairFilter);

    const chainOptions = (options.data?.chains || [])
        .filter((item): item is {chainID: number; chainName?: string} => item.chainID !== undefined)
        .map(item => ({value: item.chainID, label: chainLabel(item.chainID)}));
    const statusOptions = (values: readonly string[], noneLabel = 'None') => values.map(value => ({value, label: value === 'none' ? noneLabel : value.replace(/_/g, ' ')}));
    const hasFilters = Boolean(
        chainID ||
            projectID ||
            contract ||
            codeHash ||
            researchStatus ||
            reportState ||
            evaluationStatus ||
            selectionOutcome ||
            hasReportPairFilter(wethPairFilter) ||
            hasReportPairFilter(usdtPairFilter)
    );
    const items = data.data?.items || [];
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
                    <div className='projects-view-control'>
                        <Typography.Text strong={true}>View</Typography.Text>
                        <ChoiceGroup<ProjectsView>
                            ariaLabel='Projects view'
                            value={view}
                            options={[
                                {label: 'Overview', value: 'overview'},
                                {label: 'Report Risk', value: 'report-risk'}
                            ]}
                            onChange={setView}
                        />
                    </div>
                    {reportRiskView && (
                        <div className='projects-report-risk-section-control'>
                            <Typography.Text strong={true}>Risk section</Typography.Text>
                            <ChoiceGroup<ReportRiskSection>
                                ariaLabel='Report risk section'
                                value={riskSection}
                                options={[
                                    {label: 'Status', value: 'status'},
                                    {label: 'WETH / WBNB', value: 'wrapped-native'},
                                    {label: 'USDT', value: 'usdt'}
                                ]}
                                onChange={setRiskSection}
                            />
                        </div>
                    )}
                    {activeReportPairKind && activeReportPairFilter && (
                        <div className='projects-compact-pair-filter-region'>
                            <CompactReportPairFilters
                                pairLabel={activeReportPairKind === 'weth' ? 'WETH / WBNB' : 'USDT'}
                                filter={activeReportPairFilter}
                                onApply={filter => setReportPairFilter(activeReportPairKind, filter)}
                            />
                        </div>
                    )}
                    <div className='projects-filter-grid'>
                        <Select
                            aria-label='Filter by chain'
                            value={chainID}
                            allowClear={true}
                            placeholder='All chains'
                            options={chainOptions}
                            onChange={value => setFilter('chainID', value)}
                        />
                        <InputNumber
                            aria-label='Filter by project ID'
                            value={projectID}
                            min={1}
                            precision={0}
                            placeholder='Project ID'
                            onChange={value => setFilter('projectID', typeof value === 'number' ? value : undefined)}
                        />
                        <SearchBar value={contract} onChange={value => setFilter('contract', value)} placeholder='Contract' />
                        <SearchBar value={codeHash} onChange={value => setFilter('codeHash', value)} placeholder='Code hash' />
                        <Select
                            aria-label='Filter by research status'
                            value={researchStatus || undefined}
                            allowClear={true}
                            placeholder='Research status'
                            options={statusOptions(researchStatuses)}
                            onChange={value => setFilter('researchStatus', value)}
                        />
                        <Select
                            aria-label='Filter by report state'
                            value={reportState || undefined}
                            allowClear={true}
                            placeholder='Report state'
                            options={statusOptions(reportStates, 'No report')}
                            onChange={value => setFilter('reportState', value)}
                        />
                        <Select
                            aria-label='Filter by evaluation status'
                            value={evaluationStatus || undefined}
                            allowClear={true}
                            placeholder='Evaluation status'
                            options={statusOptions(evaluationStatuses, 'No evaluation task')}
                            onChange={value => setFilter('evaluationStatus', value)}
                        />
                        <Select
                            aria-label='Filter by selection outcome'
                            value={selectionOutcome || undefined}
                            allowClear={true}
                            placeholder='Selection outcome'
                            options={statusOptions(selectionOutcomes, 'No outcome')}
                            onChange={value => setFilter('selectionOutcome', value)}
                        />
                        <Button disabled={!hasFilters} onClick={clearFilters}>
                            Clear filters
                        </Button>
                    </div>
                </div>
            }>
            <div className={reportRiskView ? 'projects-report-risk-table-region' : undefined}>
                <ResourceTable
                    label={reportRiskView ? 'Project report risk' : 'Project overview'}
                    rowKey={item => item.project?.projectID || `${item.project?.chainID}-${item.project?.contract}`}
                    items={items}
                    columns={reportRiskView ? reportRiskColumns : overviewColumns}
                    loading={data.loading}
                    total={data.data?.total}
                    page={page}
                    pageSize={pageSize}
                    onPageChange={setPage}
                    scrollX={reportRiskView ? 820 : 1224}
                    stickyHeader={reportRiskView}
                    compactRender={item => (reportRiskView ? <ProjectReportRiskCard item={item} /> : <ProjectOverviewCard item={item} />)}
                    compactEmptyDescription={reportRiskView ? 'No projects match the report risk filters' : 'No projects match the filters'}
                />
            </div>
        </AppPage>
    );
};
