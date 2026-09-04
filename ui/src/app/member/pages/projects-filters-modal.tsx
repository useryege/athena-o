import {Button, Checkbox, Input, Modal, Select, Tabs, Typography} from 'antd';
import * as React from 'react';
import {pairRiskSignalCopy} from './token-shared';

export type ProfilePairSignalState = 'detected' | 'not_detected';

export interface ProfilePairFilterState {
    balanceSupply: ProfilePairSignalState[];
    minimumLP: ProfilePairSignalState[];
    feeLPShare: ProfilePairSignalState[];
    quoteMin: string;
    quoteMax: string;
}

export interface ProjectsFilterState {
    contract: string;
    codeHash: string;
    collectionStatus: string;
    profileState: string;
    pairFilter: ProfilePairFilterState;
}

export const collectionStatuses = ['queued', 'collecting', 'complete', 'needs_attention'] as const;
export const profileStates = ['pending', 'complete', 'incomplete', 'failed'] as const;
export const profilePairSignalStates = ['detected', 'not_detected'] as const;

const signalStateOptions = [
    {label: 'Detected', value: 'detected'},
    {label: 'Not detected', value: 'not_detected'}
];
const statusOptions = (values: readonly string[]) => values.map(value => ({value, label: value.replace(/_/g, ' ')}));

export const emptyProfilePairFilter = (): ProfilePairFilterState => ({balanceSupply: [], minimumLP: [], feeLPShare: [], quoteMin: '', quoteMax: ''});

export const emptyProjectsFilterState = (): ProjectsFilterState => ({
    contract: '',
    codeHash: '',
    collectionStatus: '',
    profileState: '',
    pairFilter: emptyProfilePairFilter()
});

export const hasProfilePairFilter = (filter: ProfilePairFilterState) =>
    filter.balanceSupply.length > 0 || filter.minimumLP.length > 0 || filter.feeLPShare.length > 0 || Boolean(filter.quoteMin || filter.quoteMax);

export const generalFilterCount = (filter: ProjectsFilterState) =>
    [filter.contract, filter.codeHash, filter.collectionStatus, filter.profileState].filter(Boolean).length;

export const profilePairFilterCount = (filter: ProfilePairFilterState) =>
    Number(filter.balanceSupply.length > 0) + Number(filter.minimumLP.length > 0) + Number(filter.feeLPShare.length > 0) + Number(Boolean(filter.quoteMin || filter.quoteMax));

const clonePairFilter = (filter: ProfilePairFilterState): ProfilePairFilterState => ({
    ...filter,
    balanceSupply: [...filter.balanceSupply],
    minimumLP: [...filter.minimumLP],
    feeLPShare: [...filter.feeLPShare]
});

const cloneFilterState = (filter: ProjectsFilterState): ProjectsFilterState => ({
    ...filter,
    pairFilter: clonePairFilter(filter.pairFilter)
});

const normalizeUnsignedInteger = (value: string) => (value ? BigInt(value).toString() : '');
const normalizeFilterState = (filter: ProjectsFilterState): ProjectsFilterState => ({
    ...filter,
    contract: filter.contract.trim(),
    codeHash: filter.codeHash.trim(),
    pairFilter: {
        ...clonePairFilter(filter.pairFilter),
        quoteMin: normalizeUnsignedInteger(filter.pairFilter.quoteMin),
        quoteMax: normalizeUnsignedInteger(filter.pairFilter.quoteMax)
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

const GeneralFilters = (props: {draft: ProjectsFilterState; onChange: (draft: ProjectsFilterState) => void}) => (
    <div className='projects-filters-modal__content'>
        <div className='projects-filters-modal__section-heading'>
            <div>
                <Typography.Title level={4}>Project and workflow</Typography.Title>
                <Typography.Text type='secondary'>Filter by discovery identity, one-time collection progress, and profile state.</Typography.Text>
            </div>
            <Button size='small' onClick={() => {
                const empty = emptyProjectsFilterState();
                props.onChange({...props.draft, ...empty, pairFilter: props.draft.pairFilter});
            }}>Clear section</Button>
        </div>
        <div className='projects-filters-modal__field-grid'>
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

const PairFilters = (props: {draft: ProjectsFilterState; error: string; onChange: (draft: ProjectsFilterState) => void}) => {
    const errorID = React.useId();
    const filter = props.draft.pairFilter;
    const setFilter = (next: ProfilePairFilterState) => props.onChange({...props.draft, pairFilter: next});
    return (
        <div className='projects-filters-modal__content'>
            <div className='projects-filters-modal__section-heading'>
                <div>
                    <Typography.Title level={4}>WETH / WBNB or USDT pair signals</Typography.Title>
                    <Typography.Text type='secondary'>The same created pair must match every enabled condition. Selected states within one condition are ORed.</Typography.Text>
                </div>
                <Button size='small' onClick={() => setFilter(emptyProfilePairFilter())}>Clear section</Button>
            </div>
            <div className='projects-filters-modal__pair-grid'>
                <SignalChoice label={pairRiskSignalCopy.pairTokenBalanceExceedsTotalSupply.label} ariaLabel={pairRiskSignalCopy.pairTokenBalanceExceedsTotalSupply.filterAriaLabel} value={filter.balanceSupply} onChange={value => setFilter({...filter, balanceSupply: value})} />
                <SignalChoice label={pairRiskSignalCopy.lpMinimumSupplyOnly.label} ariaLabel={pairRiskSignalCopy.lpMinimumSupplyOnly.filterAriaLabel} value={filter.minimumLP} onChange={value => setFilter({...filter, minimumLP: value})} />
                <SignalChoice label={pairRiskSignalCopy.fixedFeeAddressLpShareGte90Percent.label} ariaLabel={pairRiskSignalCopy.fixedFeeAddressLpShareGte90Percent.filterAriaLabel} value={filter.feeLPShare} onChange={value => setFilter({...filter, feeLPShare: value})} />
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
                    {props.error && <Typography.Text id={errorID} type='danger' role='alert'>{props.error}</Typography.Text>}
                </fieldset>
            </div>
        </div>
    );
};

export const ProjectsFiltersModal = (props: {open: boolean; filters: ProjectsFilterState; onCancel: () => void; onApply: (filters: ProjectsFilterState) => void}) => {
    const [draft, setDraft] = React.useState<ProjectsFilterState>(() => cloneFilterState(props.filters));
    const [activeTab, setActiveTab] = React.useState<'general' | 'pair'>('general');
    React.useEffect(() => {
        if (props.open) {
            setDraft(cloneFilterState(props.filters));
            setActiveTab('general');
        }
    }, [props.filters, props.open]);
    const pairError = quoteFilterError(draft.pairFilter.quoteMin, draft.pairFilter.quoteMax);
    return (
        <Modal className='projects-filters-modal' centered={true} destroyOnHidden={true} open={props.open} title='Filters' width={920} onCancel={props.onCancel} footer={
            <div className='projects-filters-modal__footer'>
                <Button onClick={() => setDraft(emptyProjectsFilterState())}>Reset all draft</Button>
                <div className='projects-filters-modal__footer-actions'>
                    <Button onClick={props.onCancel}>Cancel</Button>
                    <Button type='primary' disabled={Boolean(pairError)} onClick={() => props.onApply(normalizeFilterState(draft))}>Apply filters</Button>
                </div>
            </div>
        }>
            <Tabs activeKey={activeTab} onChange={key => setActiveTab(key as 'general' | 'pair')} items={[
                {key: 'general', label: <FilterTabLabel label='General' count={generalFilterCount(draft)} />, children: <GeneralFilters draft={draft} onChange={setDraft} />},
                {key: 'pair', label: <FilterTabLabel label='Pair signals' count={profilePairFilterCount(draft.pairFilter)} error={Boolean(pairError)} />, children: <PairFilters draft={draft} error={pairError} onChange={setDraft} />}
            ]} />
            <Typography.Text className='projects-filters-modal__scope-note' type='secondary'>A project matches when one created WETH / WBNB or USDT pair satisfies all enabled pair conditions.</Typography.Text>
        </Modal>
    );
};
