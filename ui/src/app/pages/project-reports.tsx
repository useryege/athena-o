import {InputNumber, Space, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import {AppPage, ChoiceGroup, ResourceTable, SearchBar, TruncatedText, useAsyncData} from '../components';
import {services} from '../shared/services';
import {TokenProjectReport} from '../shared/services/token-service';
import {usePagedParams} from './shared';
import {ChainBadge, chainLabel} from './token-shared';

const evaluationStatuses = ['not_started', 'pending', 'succeeded', 'failed'];

const evaluationTag = (status?: string) => {
    const colors: Record<string, string> = {
        pending: 'gold',
        succeeded: 'green',
        failed: 'red'
    };
    return <Tag color={status ? colors[status] : undefined}>{status || 'Unknown'}</Tag>;
};

const createdTag = (value?: boolean) => {
    if (value === undefined) {
        return <Tag>Unknown</Tag>;
    }
    return <Tag color={value ? 'green' : undefined}>{value ? 'Created' : 'Not created'}</Tag>;
};

const riskTag = (value?: boolean) => {
    if (value === undefined) {
        return <Tag>Unknown</Tag>;
    }
    return <Tag color={value ? 'red' : 'green'}>{value ? 'Yes' : 'No'}</Tag>;
};

const formatInteger = (value?: string) => {
    if (!value) {
        return '-';
    }
    return value.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
};

export const ProjectReportsPage = () => {
    const {params, setParams, page, pageSize, setPage} = usePagedParams();
    const chainID = Number(params.get('chainID') || params.get('chain_id')) || undefined;
    const projectID = Number(params.get('projectID') || params.get('project_id')) || undefined;
    const contract = params.get('contract') || '';
    const evaluationStatus = params.get('evaluationStatus') || params.get('evaluation_status') || '';
    const setFilter = (key: string, value?: string | number) => {
        const next = new URLSearchParams(params);
        if (value === undefined || value === '') {
            next.delete(key);
        } else {
            next.set(key, String(value));
        }
        next.set('page', '1');
        setParams(next);
    };
    const options = useAsyncData(() => services.tokenapi.getRuntimeConfiguration(), []);
    const data = useAsyncData(
        () =>
            services.tokenapi.listProjectReports({
                page,
                pageSize,
                chainID,
                projectID,
                contract: contract || undefined,
                evaluationStatus: evaluationStatus || undefined
            }),
        [page, pageSize, chainID, projectID, contract, evaluationStatus]
    );
    const columns: ColumnsType<TokenProjectReport> = [
        {
            title: 'Project',
            children: [
                {title: 'ID', dataIndex: 'projectID'},
                {title: 'Chain', render: item => <ChainBadge chainID={item.chainID} />},
                {
                    title: 'Token',
                    render: item => (
                        <Space orientation='vertical' size={0}>
                            <Typography.Text strong={true}>{item.symbol || '-'}</Typography.Text>
                            <Typography.Text type='secondary'>{item.name || '-'}</Typography.Text>
                        </Space>
                    )
                },
                {title: 'Contract', render: item => <TruncatedText value={item.contract} copyable={true} />}
            ]
        },
        {
            title: 'Evaluation',
            children: [
                {title: 'Status', render: item => evaluationTag(item.evaluationStatus)},
                {title: 'Attempts', dataIndex: 'evaluationAttempts'},
                {title: 'Last Error', render: item => <TruncatedText value={item.evaluationLastError} />}
            ]
        },
        {
            title: 'WETH Pair',
            children: [
                {title: 'Created', render: item => createdTag(item.reportDataAvailable ? item.wethPairIsCreated : undefined)},
                {title: 'Remove Liquidity', render: item => riskTag(item.reportDataAvailable ? item.wethPairIsRemoveLiquidity : undefined)},
                {title: 'Mint', render: item => riskTag(item.reportDataAvailable ? item.wethPairIsMint : undefined)},
                {title: 'Quote USDT', render: item => (item.reportDataAvailable ? formatInteger(item.wethPairQuoteUsdtValueInt) : '-')},
                {title: 'Last Swap', render: item => (item.reportDataAvailable ? item.wethPairLastSwapAt || '-' : '-')}
            ]
        },
        {
            title: 'USDT Pair',
            children: [
                {title: 'Created', render: item => createdTag(item.reportDataAvailable ? item.usdtPairIsCreated : undefined)},
                {title: 'Remove Liquidity', render: item => riskTag(item.reportDataAvailable ? item.usdtPairIsRemoveLiquidity : undefined)},
                {title: 'Mint', render: item => riskTag(item.reportDataAvailable ? item.usdtPairIsMint : undefined)},
                {title: 'Quote USDT', render: item => (item.reportDataAvailable ? formatInteger(item.usdtPairQuoteUsdtValueInt) : '-')},
                {title: 'Last Swap', render: item => (item.reportDataAvailable ? item.usdtPairLastSwapAt || '-' : '-')}
            ]
        },
        {
            title: 'Updated',
            children: [
                {title: 'Source', dataIndex: 'sourceUpdatedAt'},
                {title: 'Evaluated', dataIndex: 'evaluatedAt'},
                {title: 'Task', dataIndex: 'evaluationUpdatedAt'},
                {title: 'Report Created', dataIndex: 'createdAt'}
            ]
        }
    ];
    const chainOptions = (options.data?.chains || [])
        .filter((item): item is {chainID: number; chainName?: string} => item.chainID !== undefined)
        .map(item => ({
            value: item.chainID,
            label: chainLabel(item.chainID)
        }));
    return (
        <AppPage
            title='Project Reports'
            loading={data.loading || options.loading}
            error={data.error || options.error}
            onRefresh={() => {
                data.reload();
                options.reload();
            }}
            filters={
                <Space wrap={true}>
                    <ChoiceGroup<number | 'all'>
                        ariaLabel='Filter by chain'
                        value={chainID ?? 'all'}
                        options={[{label: 'All', value: 'all'}, ...chainOptions]}
                        onChange={value => {
                            setFilter('chainID', value === 'all' ? undefined : value);
                        }}
                    />
                    <InputNumber
                        aria-label='Filter by project ID'
                        value={projectID}
                        min={1}
                        placeholder='Project ID'
                        onChange={value => {
                            setFilter('projectID', typeof value === 'number' ? value : undefined);
                        }}
                    />
                    <SearchBar value={contract} onChange={value => setFilter('contract', value)} placeholder='Contract' />
                    <ChoiceGroup<string>
                        ariaLabel='Filter by evaluation status'
                        value={evaluationStatus || 'all'}
                        options={[{label: 'All', value: 'all'}, ...evaluationStatuses.map(value => ({value, label: value}))]}
                        onChange={value => {
                            setFilter('evaluationStatus', value === 'all' ? '' : value);
                        }}
                    />
                </Space>
            }>
            <ResourceTable
                rowKey={item => item.projectID || `${item.chainID}-${item.contract}`}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                scrollX={2200}
            />
        </AppPage>
    );
};
