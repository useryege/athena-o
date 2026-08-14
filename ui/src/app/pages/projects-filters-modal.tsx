import {Button, Checkbox, Input, InputNumber, Modal, Select, Tabs, Typography} from 'antd';
import * as React from 'react';

export type ReportPairKind = 'weth' | 'usdt';
export type ReportPairRiskState = 'detected' | 'clear' | 'no_report' | 'risk_unavailable';
export type ReportPairMissingState = 'no_report' | 'risk_unavailable';

export interface ReportPairFilterState {
    removeLiquidity: ReportPairRiskState[];
    mint: ReportPairRiskState[];
    quoteMin: string;
    quoteMax: string;
    quoteMissing: ReportPairMissingState[];
}

export interface ProjectsFilterState {
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
}

export const researchStatuses = ['researching', 'selected', 'rejected', 'expired'] as const;
export const reportStates = ['none', 'incomplete', 'complete'] as const;
export const evaluationStatuses = ['none', 'pending', 'running', 'succeeded', 'failed'] as const;
export const selectionOutcomes = ['none', 'selected', 'rejected', 'deferred'] as const;
export const reportPairRiskStates = ['detected', 'clear', 'no_report', 'risk_unavailable'] as const;
export const reportPairMissingStates = ['no_report', 'risk_unavailable'] as const;

const reportPairRiskStateOptions = [
    {label: 'Clear', value: 'clear'},
    {label: 'Detected', value: 'detected'},
    {label: 'No report', value: 'no_report'},
    {label: 'Risk unavailable', value: 'risk_unavailable'}
];
const reportPairMissingStateOptions = [
    {label: 'No report', value: 'no_report'},
    {label: 'Risk unavailable', value: 'risk_unavailable'}
];

const statusOptions = (values: readonly string[], noneLabel = 'None') => values.map(value => ({value, label: value === 'none' ? noneLabel : value.replace(/_/g, ' ')}));

export const emptyReportPairFilter = (): ReportPairFilterState => ({removeLiquidity: [], mint: [], quoteMin: '', quoteMax: '', quoteMissing: []});

export const emptyProjectsFilterState = (): ProjectsFilterState => ({
    chainID: undefined,
    projectID: undefined,
    contract: '',
    codeHash: '',
    researchStatus: '',
    reportState: '',
    evaluationStatus: '',
    selectionOutcome: '',
    wethPairFilter: emptyReportPairFilter(),
    usdtPairFilter: emptyReportPairFilter()
});

export const hasReportPairFilter = (filter: ReportPairFilterState) =>
    filter.removeLiquidity.length > 0 || filter.mint.length > 0 || Boolean(filter.quoteMin || filter.quoteMax) || filter.quoteMissing.length > 0;

export const generalFilterCount = (filter: ProjectsFilterState) =>
    [filter.chainID, filter.projectID, filter.contract, filter.codeHash, filter.researchStatus, filter.reportState, filter.evaluationStatus, filter.selectionOutcome].filter(
        Boolean
    ).length;

export const reportPairFilterCount = (filter: ReportPairFilterState) =>
    Number(filter.removeLiquidity.length > 0) + Number(filter.mint.length > 0) + Number(Boolean(filter.quoteMin || filter.quoteMax || filter.quoteMissing.length > 0));

const cloneReportPairFilter = (filter: ReportPairFilterState): ReportPairFilterState => ({
    ...filter,
    removeLiquidity: [...filter.removeLiquidity],
    mint: [...filter.mint],
    quoteMissing: [...filter.quoteMissing]
});

const cloneProjectsFilterState = (filter: ProjectsFilterState): ProjectsFilterState => ({
    ...filter,
    wethPairFilter: cloneReportPairFilter(filter.wethPairFilter),
    usdtPairFilter: cloneReportPairFilter(filter.usdtPairFilter)
});

const normalizeUnsignedInteger = (value: string) => (value ? BigInt(value).toString() : '');

const normalizeProjectsFilterState = (filter: ProjectsFilterState): ProjectsFilterState => ({
    ...filter,
    contract: filter.contract.trim(),
    codeHash: filter.codeHash.trim(),
    wethPairFilter: {
        ...cloneReportPairFilter(filter.wethPairFilter),
        quoteMin: normalizeUnsignedInteger(filter.wethPairFilter.quoteMin),
        quoteMax: normalizeUnsignedInteger(filter.wethPairFilter.quoteMax)
    },
    usdtPairFilter: {
        ...cloneReportPairFilter(filter.usdtPairFilter),
        quoteMin: normalizeUnsignedInteger(filter.usdtPairFilter.quoteMin),
        quoteMax: normalizeUnsignedInteger(filter.usdtPairFilter.quoteMax)
    }
});

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

const FilterTabLabel = (props: {label: string; count: number; state?: 'Active' | 'Saved'; error?: boolean}) => (
    <span className={`projects-filters-modal__tab-label${props.error ? ' projects-filters-modal__tab-label--error' : ''}`}>
        <span>{props.label}</span>
        <span className='projects-filters-modal__tab-count' aria-label={`${props.count} filter fields`}>
            {props.count}
        </span>
        {props.state && <span className='projects-filters-modal__tab-state'>{props.state}</span>}
        {props.error && <span className='projects-filters-modal__tab-error'>Error</span>}
    </span>
);

const GeneralFilters = (props: {draft: ProjectsFilterState; chainOptions: Array<{value: number; label: React.ReactNode}>; onChange: (draft: ProjectsFilterState) => void}) => {
    const clearGeneral = () => {
        const empty = emptyProjectsFilterState();
        props.onChange({
            ...props.draft,
            chainID: undefined,
            projectID: undefined,
            contract: empty.contract,
            codeHash: empty.codeHash,
            researchStatus: empty.researchStatus,
            reportState: empty.reportState,
            evaluationStatus: empty.evaluationStatus,
            selectionOutcome: empty.selectionOutcome
        });
    };
    return (
        <div className='projects-filters-modal__content'>
            <div className='projects-filters-modal__section-heading'>
                <div>
                    <Typography.Title level={4}>General filters</Typography.Title>
                    <Typography.Text type='secondary'>These conditions apply in Overview and every Report Risk section.</Typography.Text>
                </div>
                <Button size='small' onClick={clearGeneral}>
                    Clear section
                </Button>
            </div>
            <div className='projects-filters-modal__section-heading projects-filters-modal__section-heading--subsection'>
                <div>
                    <Typography.Title level={4}>Project</Typography.Title>
                    <Typography.Text type='secondary'>Match project identity and deployment metadata.</Typography.Text>
                </div>
            </div>
            <div className='projects-filters-modal__field-grid'>
                <label className='projects-filters-modal__field'>
                    <span>Chain</span>
                    <Select
                        aria-label='Filter by chain'
                        value={props.draft.chainID}
                        allowClear={true}
                        placeholder='All chains'
                        options={props.chainOptions}
                        onChange={value => props.onChange({...props.draft, chainID: value})}
                    />
                </label>
                <label className='projects-filters-modal__field'>
                    <span>Project ID</span>
                    <InputNumber
                        aria-label='Filter by project ID'
                        value={props.draft.projectID}
                        min={1}
                        precision={0}
                        placeholder='Any project'
                        onChange={value => props.onChange({...props.draft, projectID: typeof value === 'number' ? value : undefined})}
                    />
                </label>
                <label className='projects-filters-modal__field'>
                    <span>Contract</span>
                    <Input value={props.draft.contract} placeholder='Any contract' onChange={event => props.onChange({...props.draft, contract: event.target.value})} />
                </label>
                <label className='projects-filters-modal__field'>
                    <span>Code hash</span>
                    <Input value={props.draft.codeHash} placeholder='Any code hash' onChange={event => props.onChange({...props.draft, codeHash: event.target.value})} />
                </label>
            </div>
            <div className='projects-filters-modal__section-heading projects-filters-modal__section-heading--subsection'>
                <div>
                    <Typography.Title level={4}>Lifecycle</Typography.Title>
                    <Typography.Text type='secondary'>Filter current research, Report, Evaluation, and Selection state.</Typography.Text>
                </div>
            </div>
            <div className='projects-filters-modal__field-grid'>
                <label className='projects-filters-modal__field'>
                    <span>Research status</span>
                    <Select
                        aria-label='Filter by research status'
                        value={props.draft.researchStatus || undefined}
                        allowClear={true}
                        placeholder='Any status'
                        options={statusOptions(researchStatuses)}
                        onChange={value => props.onChange({...props.draft, researchStatus: value || ''})}
                    />
                </label>
                <label className='projects-filters-modal__field'>
                    <span>Report state</span>
                    <Select
                        aria-label='Filter by report state'
                        value={props.draft.reportState || undefined}
                        allowClear={true}
                        placeholder='Any state'
                        options={statusOptions(reportStates, 'No report')}
                        onChange={value => props.onChange({...props.draft, reportState: value || ''})}
                    />
                </label>
                <label className='projects-filters-modal__field'>
                    <span>Evaluation status</span>
                    <Select
                        aria-label='Filter by evaluation status'
                        value={props.draft.evaluationStatus || undefined}
                        allowClear={true}
                        placeholder='Any status'
                        options={statusOptions(evaluationStatuses, 'No evaluation task')}
                        onChange={value => props.onChange({...props.draft, evaluationStatus: value || ''})}
                    />
                </label>
                <label className='projects-filters-modal__field'>
                    <span>Selection outcome</span>
                    <Select
                        aria-label='Filter by selection outcome'
                        value={props.draft.selectionOutcome || undefined}
                        allowClear={true}
                        placeholder='Any outcome'
                        options={statusOptions(selectionOutcomes, 'No outcome')}
                        onChange={value => props.onChange({...props.draft, selectionOutcome: value || ''})}
                    />
                </label>
            </div>
        </div>
    );
};

const PairFilters = (props: {kind: ReportPairKind; label: string; draft: ProjectsFilterState; error: string; onChange: (draft: ProjectsFilterState) => void}) => {
    const errorID = React.useId();
    const filterKey = props.kind === 'weth' ? 'wethPairFilter' : 'usdtPairFilter';
    const filter = props.draft[filterKey];
    const setFilter = (next: ReportPairFilterState) => props.onChange({...props.draft, [filterKey]: next});
    return (
        <div className='projects-filters-modal__content'>
            <div className='projects-filters-modal__section-heading'>
                <div>
                    <Typography.Title level={4}>{props.label} Pair risk</Typography.Title>
                    <Typography.Text type='secondary'>States within one field are ORed; Remove Liquidity, Mint, and Quote are ANDed.</Typography.Text>
                </div>
                <Button size='small' onClick={() => setFilter(emptyReportPairFilter())}>
                    Clear section
                </Button>
            </div>
            <div className='projects-filters-modal__pair-grid'>
                <fieldset className='projects-filters-modal__choice-field'>
                    <legend>Remove Liquidity</legend>
                    <Checkbox.Group
                        aria-label={`Filter ${props.label} Remove Liquidity`}
                        options={reportPairRiskStateOptions}
                        value={filter.removeLiquidity}
                        onChange={values => setFilter({...filter, removeLiquidity: values as ReportPairRiskState[]})}
                    />
                </fieldset>
                <fieldset className='projects-filters-modal__choice-field'>
                    <legend>Mint</legend>
                    <Checkbox.Group
                        aria-label={`Filter ${props.label} Mint`}
                        options={reportPairRiskStateOptions}
                        value={filter.mint}
                        onChange={values => setFilter({...filter, mint: values as ReportPairRiskState[]})}
                    />
                </fieldset>
                <fieldset className='projects-filters-modal__quote-field'>
                    <legend>Quote USDT</legend>
                    <div className='projects-filters-modal__quote-range'>
                        <label className='projects-filters-modal__field'>
                            <span>Minimum</span>
                            <Input
                                aria-describedby={props.error ? errorID : undefined}
                                aria-invalid={Boolean(props.error)}
                                inputMode='numeric'
                                pattern='[0-9]*'
                                placeholder='No minimum'
                                value={filter.quoteMin}
                                onChange={event => setFilter({...filter, quoteMin: event.target.value.trim()})}
                            />
                        </label>
                        <span className='projects-filters-modal__range-divider' aria-hidden='true'>
                            —
                        </span>
                        <label className='projects-filters-modal__field'>
                            <span>Maximum</span>
                            <Input
                                aria-describedby={props.error ? errorID : undefined}
                                aria-invalid={Boolean(props.error)}
                                inputMode='numeric'
                                pattern='[0-9]*'
                                placeholder='No maximum'
                                value={filter.quoteMax}
                                onChange={event => setFilter({...filter, quoteMax: event.target.value.trim()})}
                            />
                        </label>
                    </div>
                    <div className='projects-filters-modal__missing-field'>
                        <Typography.Text strong={true}>Include missing</Typography.Text>
                        <Checkbox.Group
                            aria-label={`Filter ${props.label} Quote USDT missing states`}
                            options={reportPairMissingStateOptions}
                            value={filter.quoteMissing}
                            onChange={values => setFilter({...filter, quoteMissing: values as ReportPairMissingState[]})}
                        />
                    </div>
                    {props.error && (
                        <Typography.Text id={errorID} type='danger' role='alert'>
                            {props.error}
                        </Typography.Text>
                    )}
                </fieldset>
            </div>
        </div>
    );
};

export const ProjectsFiltersModal = (props: {
    open: boolean;
    filters: ProjectsFilterState;
    activePairKind?: ReportPairKind;
    chainOptions: Array<{value: number; label: React.ReactNode}>;
    onCancel: () => void;
    onApply: (filters: ProjectsFilterState) => void;
}) => {
    const [draft, setDraft] = React.useState<ProjectsFilterState>(() => cloneProjectsFilterState(props.filters));
    const [activeTab, setActiveTab] = React.useState<'general' | ReportPairKind>('general');
    React.useEffect(() => {
        if (props.open) {
            setDraft(cloneProjectsFilterState(props.filters));
            setActiveTab(props.activePairKind || 'general');
        }
    }, [props.activePairKind, props.filters, props.open]);

    const wethError = quoteFilterError(draft.wethPairFilter.quoteMin, draft.wethPairFilter.quoteMax);
    const usdtError = quoteFilterError(draft.usdtPairFilter.quoteMin, draft.usdtPairFilter.quoteMax);
    const hasError = Boolean(wethError || usdtError);
    const tabState = (kind: ReportPairKind) =>
        props.activePairKind === kind ? 'Active' : reportPairFilterCount(draft[kind === 'weth' ? 'wethPairFilter' : 'usdtPairFilter']) > 0 ? 'Saved' : undefined;

    return (
        <Modal
            className='projects-filters-modal'
            centered={true}
            destroyOnHidden={true}
            open={props.open}
            title='Filters'
            width={920}
            onCancel={props.onCancel}
            footer={
                <div className='projects-filters-modal__footer'>
                    <Button onClick={() => setDraft(emptyProjectsFilterState())}>Reset all draft</Button>
                    <div className='projects-filters-modal__footer-actions'>
                        <Button onClick={props.onCancel}>Cancel</Button>
                        <Button type='primary' disabled={hasError} onClick={() => props.onApply(normalizeProjectsFilterState(draft))}>
                            Apply filters
                        </Button>
                    </div>
                </div>
            }>
            <Tabs
                activeKey={activeTab}
                onChange={key => setActiveTab(key as 'general' | ReportPairKind)}
                items={[
                    {
                        key: 'general',
                        label: <FilterTabLabel label='General' count={generalFilterCount(draft)} />,
                        children: <GeneralFilters draft={draft} chainOptions={props.chainOptions} onChange={setDraft} />
                    },
                    {
                        key: 'weth',
                        label: <FilterTabLabel label='WETH / WBNB' count={reportPairFilterCount(draft.wethPairFilter)} state={tabState('weth')} error={Boolean(wethError)} />,
                        children: <PairFilters kind='weth' label='WETH / WBNB' draft={draft} error={wethError} onChange={setDraft} />
                    },
                    {
                        key: 'usdt',
                        label: <FilterTabLabel label='USDT' count={reportPairFilterCount(draft.usdtPairFilter)} state={tabState('usdt')} error={Boolean(usdtError)} />,
                        children: <PairFilters kind='usdt' label='USDT' draft={draft} error={usdtError} onChange={setDraft} />
                    }
                ]}
            />
            <Typography.Text className='projects-filters-modal__scope-note' type='secondary'>
                Pair filters are saved independently and only affect results while their matching Report Risk section is active.
            </Typography.Text>
        </Modal>
    );
};
