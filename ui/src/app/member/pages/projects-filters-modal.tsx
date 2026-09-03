import {Button, Checkbox, Input, InputNumber, Modal, Select, Tabs, Typography} from 'antd';
import * as React from 'react';

export type ProfilePairKind = 'wrappedNative' | 'usdt';
export type ProfilePairSignalState = 'detected' | 'clear' | 'no_profile' | 'signal_unavailable';
export type ProfilePairQuoteMissingState = 'no_profile' | 'value_unavailable';

export interface ProfilePairFilterState {
    balanceSupply: ProfilePairSignalState[];
    minimumLP: ProfilePairSignalState[];
    feeLPShare: ProfilePairSignalState[];
    quoteMin: string;
    quoteMax: string;
    quoteMissing: ProfilePairQuoteMissingState[];
}

export interface ProjectsFilterState {
    chainID?: number;
    projectID?: number;
    contract: string;
    codeHash: string;
    collectionStatus: string;
    profileState: string;
    wrappedNativePairFilter: ProfilePairFilterState;
    usdtPairFilter: ProfilePairFilterState;
}

export const collectionStatuses = ['queued', 'collecting', 'complete', 'needs_attention'] as const;
export const profileStates = ['pending', 'complete', 'incomplete', 'failed'] as const;
export const profilePairSignalStates = ['detected', 'clear', 'no_profile', 'signal_unavailable'] as const;
export const profilePairQuoteMissingStates = ['no_profile', 'value_unavailable'] as const;

const signalStateOptions = [
    {label: 'Clear', value: 'clear'},
    {label: 'Detected', value: 'detected'},
    {label: 'No profile', value: 'no_profile'},
    {label: 'Signal unavailable', value: 'signal_unavailable'}
];
const quoteMissingStateOptions = [
    {label: 'No profile', value: 'no_profile'},
    {label: 'Value unavailable', value: 'value_unavailable'}
];
const statusOptions = (values: readonly string[]) => values.map(value => ({value, label: value.replace(/_/g, ' ')}));

export const emptyProfilePairFilter = (): ProfilePairFilterState => ({balanceSupply: [], minimumLP: [], feeLPShare: [], quoteMin: '', quoteMax: '', quoteMissing: []});

export const emptyProjectsFilterState = (): ProjectsFilterState => ({
    chainID: undefined,
    projectID: undefined,
    contract: '',
    codeHash: '',
    collectionStatus: '',
    profileState: '',
    wrappedNativePairFilter: emptyProfilePairFilter(),
    usdtPairFilter: emptyProfilePairFilter()
});

export const hasProfilePairFilter = (filter: ProfilePairFilterState) =>
    filter.balanceSupply.length > 0 || filter.minimumLP.length > 0 || filter.feeLPShare.length > 0 || Boolean(filter.quoteMin || filter.quoteMax || filter.quoteMissing.length > 0);

export const generalFilterCount = (filter: ProjectsFilterState) =>
    [filter.chainID, filter.projectID, filter.contract, filter.codeHash, filter.collectionStatus, filter.profileState].filter(Boolean).length;

export const profilePairFilterCount = (filter: ProfilePairFilterState) =>
    Number(filter.balanceSupply.length > 0) + Number(filter.minimumLP.length > 0) + Number(filter.feeLPShare.length > 0) + Number(Boolean(filter.quoteMin || filter.quoteMax || filter.quoteMissing.length > 0));

const clonePairFilter = (filter: ProfilePairFilterState): ProfilePairFilterState => ({
    ...filter,
    balanceSupply: [...filter.balanceSupply],
    minimumLP: [...filter.minimumLP],
    feeLPShare: [...filter.feeLPShare],
    quoteMissing: [...filter.quoteMissing]
});

const cloneFilterState = (filter: ProjectsFilterState): ProjectsFilterState => ({
    ...filter,
    wrappedNativePairFilter: clonePairFilter(filter.wrappedNativePairFilter),
    usdtPairFilter: clonePairFilter(filter.usdtPairFilter)
});

const normalizeUnsignedInteger = (value: string) => (value ? BigInt(value).toString() : '');
const normalizeFilterState = (filter: ProjectsFilterState): ProjectsFilterState => ({
    ...filter,
    contract: filter.contract.trim(),
    codeHash: filter.codeHash.trim(),
    wrappedNativePairFilter: {
        ...clonePairFilter(filter.wrappedNativePairFilter),
        quoteMin: normalizeUnsignedInteger(filter.wrappedNativePairFilter.quoteMin),
        quoteMax: normalizeUnsignedInteger(filter.wrappedNativePairFilter.quoteMax)
    },
    usdtPairFilter: {
        ...clonePairFilter(filter.usdtPairFilter),
        quoteMin: normalizeUnsignedInteger(filter.usdtPairFilter.quoteMin),
        quoteMax: normalizeUnsignedInteger(filter.usdtPairFilter.quoteMax)
    }
});

const quoteFilterError = (minimum: string, maximum: string) => {
    if (minimum && !/^\d+$/.test(minimum)) return 'Minimum must be a non-negative integer.';
    if (maximum && !/^\d+$/.test(maximum)) return 'Maximum must be a non-negative integer.';
    if (minimum && maximum && BigInt(minimum) > BigInt(maximum)) return 'Minimum must not exceed maximum.';
    return '';
};

const FilterTabLabel = (props: {label: string; count: number; error?: boolean}) => (
    <span className={`projects-filters-modal__tab-label${props.error ? ' projects-filters-modal__tab-label--error' : ''}`}>
        <span>{props.label}</span>
        <span className='projects-filters-modal__tab-count' aria-label={`${props.count} filter fields`}>{props.count}</span>
        {props.error && <span className='projects-filters-modal__tab-error'>Error</span>}
    </span>
);

const GeneralFilters = (props: {draft: ProjectsFilterState; chainOptions: Array<{value: number; label: React.ReactNode}>; onChange: (draft: ProjectsFilterState) => void}) => (
    <div className='projects-filters-modal__content'>
        <div className='projects-filters-modal__section-heading'>
            <div>
                <Typography.Title level={4}>Project and workflow</Typography.Title>
                <Typography.Text type='secondary'>Filter by discovery identity, one-time collection progress, and profile state.</Typography.Text>
            </div>
            <Button size='small' onClick={() => {
                const empty = emptyProjectsFilterState();
                props.onChange({...props.draft, ...empty, wrappedNativePairFilter: props.draft.wrappedNativePairFilter, usdtPairFilter: props.draft.usdtPairFilter});
            }}>Clear section</Button>
        </div>
        <div className='projects-filters-modal__field-grid'>
            <label className='projects-filters-modal__field'>
                <span>Chain</span>
                <Select aria-label='Filter by chain' value={props.draft.chainID} allowClear={true} placeholder='All chains' options={props.chainOptions} onChange={value => props.onChange({...props.draft, chainID: value})} />
            </label>
            <label className='projects-filters-modal__field'>
                <span>Project ID</span>
                <InputNumber aria-label='Filter by project ID' value={props.draft.projectID} min={1} precision={0} placeholder='Any project' onChange={value => props.onChange({...props.draft, projectID: typeof value === 'number' ? value : undefined})} />
            </label>
            <label className='projects-filters-modal__field'>
                <span>Contract</span>
                <Input value={props.draft.contract} placeholder='Any contract' onChange={event => props.onChange({...props.draft, contract: event.target.value})} />
            </label>
            <label className='projects-filters-modal__field'>
                <span>Code hash</span>
                <Input value={props.draft.codeHash} placeholder='Any code hash' onChange={event => props.onChange({...props.draft, codeHash: event.target.value})} />
            </label>
            <label className='projects-filters-modal__field'>
                <span>Collection status</span>
                <Select aria-label='Filter by collection status' value={props.draft.collectionStatus || undefined} allowClear={true} placeholder='Any status' options={statusOptions(collectionStatuses)} onChange={value => props.onChange({...props.draft, collectionStatus: value || ''})} />
            </label>
            <label className='projects-filters-modal__field'>
                <span>Profile state</span>
                <Select aria-label='Filter by profile state' value={props.draft.profileState || undefined} allowClear={true} placeholder='Any state' options={statusOptions(profileStates)} onChange={value => props.onChange({...props.draft, profileState: value || ''})} />
            </label>
        </div>
    </div>
);

const SignalChoice = (props: {label: string; ariaLabel: string; value: ProfilePairSignalState[]; onChange: (value: ProfilePairSignalState[]) => void}) => (
    <fieldset className='projects-filters-modal__choice-field'>
        <legend>{props.label}</legend>
        <Checkbox.Group aria-label={props.ariaLabel} options={signalStateOptions} value={props.value} onChange={values => props.onChange(values as ProfilePairSignalState[])} />
    </fieldset>
);

const PairFilters = (props: {kind: ProfilePairKind; label: string; draft: ProjectsFilterState; error: string; onChange: (draft: ProjectsFilterState) => void}) => {
    const errorID = React.useId();
    const filterKey = props.kind === 'wrappedNative' ? 'wrappedNativePairFilter' : 'usdtPairFilter';
    const filter = props.draft[filterKey];
    const setFilter = (next: ProfilePairFilterState) => props.onChange({...props.draft, [filterKey]: next});
    return (
        <div className='projects-filters-modal__content'>
            <div className='projects-filters-modal__section-heading'>
                <div>
                    <Typography.Title level={4}>{props.label} pair signals</Typography.Title>
                    <Typography.Text type='secondary'>States within a signal are ORed. Different signals are applied together.</Typography.Text>
                </div>
                <Button size='small' onClick={() => setFilter(emptyProfilePairFilter())}>Clear section</Button>
            </div>
            <div className='projects-filters-modal__pair-grid'>
                <SignalChoice label='Pair token balance exceeds total supply' ariaLabel={`Filter ${props.label} pair token balance signal`} value={filter.balanceSupply} onChange={value => setFilter({...filter, balanceSupply: value})} />
                <SignalChoice label='LP minimum supply only' ariaLabel={`Filter ${props.label} LP minimum supply signal`} value={filter.minimumLP} onChange={value => setFilter({...filter, minimumLP: value})} />
                <SignalChoice label='Fixed fee address LP share ≥ 90%' ariaLabel={`Filter ${props.label} fixed fee address LP share signal`} value={filter.feeLPShare} onChange={value => setFilter({...filter, feeLPShare: value})} />
                <fieldset className='projects-filters-modal__quote-field'>
                    <legend>Quote value in USDT</legend>
                    <div className='projects-filters-modal__quote-range'>
                        <label className='projects-filters-modal__field'>
                            <span>Minimum</span>
                            <Input aria-describedby={props.error ? errorID : undefined} aria-invalid={Boolean(props.error)} inputMode='numeric' pattern='[0-9]*' placeholder='No minimum' value={filter.quoteMin} onChange={event => setFilter({...filter, quoteMin: event.target.value.trim()})} />
                        </label>
                        <span className='projects-filters-modal__range-divider' aria-hidden='true'>—</span>
                        <label className='projects-filters-modal__field'>
                            <span>Maximum</span>
                            <Input aria-describedby={props.error ? errorID : undefined} aria-invalid={Boolean(props.error)} inputMode='numeric' pattern='[0-9]*' placeholder='No maximum' value={filter.quoteMax} onChange={event => setFilter({...filter, quoteMax: event.target.value.trim()})} />
                        </label>
                    </div>
                    <div className='projects-filters-modal__missing-field'>
                        <Typography.Text strong={true}>Include missing</Typography.Text>
                        <Checkbox.Group aria-label={`Filter ${props.label} quote value missing states`} options={quoteMissingStateOptions} value={filter.quoteMissing} onChange={values => setFilter({...filter, quoteMissing: values as ProfilePairQuoteMissingState[]})} />
                    </div>
                    {props.error && <Typography.Text id={errorID} type='danger' role='alert'>{props.error}</Typography.Text>}
                </fieldset>
            </div>
        </div>
    );
};

export const ProjectsFiltersModal = (props: {open: boolean; filters: ProjectsFilterState; chainOptions: Array<{value: number; label: React.ReactNode}>; onCancel: () => void; onApply: (filters: ProjectsFilterState) => void}) => {
    const [draft, setDraft] = React.useState<ProjectsFilterState>(() => cloneFilterState(props.filters));
    const [activeTab, setActiveTab] = React.useState<'general' | ProfilePairKind>('general');
    React.useEffect(() => {
        if (props.open) {
            setDraft(cloneFilterState(props.filters));
            setActiveTab('general');
        }
    }, [props.filters, props.open]);
    const wrappedError = quoteFilterError(draft.wrappedNativePairFilter.quoteMin, draft.wrappedNativePairFilter.quoteMax);
    const usdtError = quoteFilterError(draft.usdtPairFilter.quoteMin, draft.usdtPairFilter.quoteMax);
    const hasError = Boolean(wrappedError || usdtError);
    return (
        <Modal className='projects-filters-modal' centered={true} destroyOnHidden={true} open={props.open} title='Filters' width={920} onCancel={props.onCancel} footer={
            <div className='projects-filters-modal__footer'>
                <Button onClick={() => setDraft(emptyProjectsFilterState())}>Reset all draft</Button>
                <div className='projects-filters-modal__footer-actions'>
                    <Button onClick={props.onCancel}>Cancel</Button>
                    <Button type='primary' disabled={hasError} onClick={() => props.onApply(normalizeFilterState(draft))}>Apply filters</Button>
                </div>
            </div>
        }>
            <Tabs activeKey={activeTab} onChange={key => setActiveTab(key as 'general' | ProfilePairKind)} items={[
                {key: 'general', label: <FilterTabLabel label='General' count={generalFilterCount(draft)} />, children: <GeneralFilters draft={draft} chainOptions={props.chainOptions} onChange={setDraft} />},
                {key: 'wrappedNative', label: <FilterTabLabel label='WETH / WBNB' count={profilePairFilterCount(draft.wrappedNativePairFilter)} error={Boolean(wrappedError)} />, children: <PairFilters kind='wrappedNative' label='WETH / WBNB' draft={draft} error={wrappedError} onChange={setDraft} />},
                {key: 'usdt', label: <FilterTabLabel label='USDT' count={profilePairFilterCount(draft.usdtPairFilter)} error={Boolean(usdtError)} />, children: <PairFilters kind='usdt' label='USDT' draft={draft} error={usdtError} onChange={setDraft} />}
            ]} />
            <Typography.Text className='projects-filters-modal__scope-note' type='secondary'>Both pair filter groups are applied together. A project must match every non-empty signal group.</Typography.Text>
        </Modal>
    );
};
