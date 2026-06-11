import {InputNumber, Select, Space, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import {AppPage, CardTitle, KeyValueGrid, ResponsiveResourceList, SearchBar, TruncatedText, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {TokenAPIProjectReport} from '../../shared/services/tokenapi-service';
import {usePagedParams} from './shared';
import {ChainBadge} from './token-shared';

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

const PairSummary = (props: {title: string; available?: boolean; created?: boolean; removeLiquidity?: boolean; mint?: boolean; quoteUsdtValue?: string; lastSwapAt?: string}) => (
    <div className='project-report-pair'>
        <Typography.Text strong={true}>{props.title}</Typography.Text>
        <KeyValueGrid
            columns={1}
            items={[
                {label: 'Created', value: createdTag(props.available ? props.created : undefined)},
                {label: 'Remove Liquidity', value: riskTag(props.available ? props.removeLiquidity : undefined)},
                {label: 'Mint', value: riskTag(props.available ? props.mint : undefined)},
                {label: 'Quote USDT', value: props.available ? formatInteger(props.quoteUsdtValue) : '-'},
                {label: 'Last Swap', value: props.available ? props.lastSwapAt : '-'}
            ]}
        />
    </div>
);

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
    const options = useAsyncData(() => services.tokenapi.getOptions(), []);
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
    const columns: ColumnsType<TokenAPIProjectReport> = [
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
    const chainOptions = (options.data?.chains || []).map(item => ({
        value: item.chainID,
        label: `${item.chainName || item.chainID} (${item.chainID})`
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
                    <Select
                        allowClear={true}
                        value={chainID}
                        placeholder='Chain'
                        style={{width: 220}}
                        options={chainOptions}
                        onChange={value => {
                            setFilter('chainID', value);
                        }}
                    />
                    <InputNumber
                        value={projectID}
                        min={1}
                        placeholder='Project ID'
                        onChange={value => {
                            setFilter('projectID', typeof value === 'number' ? value : undefined);
                        }}
                    />
                    <SearchBar value={contract} onChange={value => setFilter('contract', value)} placeholder='Contract' />
                    <Select
                        allowClear={true}
                        value={evaluationStatus || undefined}
                        placeholder='Evaluation status'
                        style={{width: 190}}
                        options={evaluationStatuses.map(value => ({value, label: value}))}
                        onChange={value => {
                            setFilter('evaluationStatus', value || '');
                        }}
                    />
                </Space>
            }>
            <ResponsiveResourceList
                rowKey={item => item.projectID || `${item.chainID}-${item.contract}`}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                card={item => (
                    <>
                        <CardTitle
                            title={`${item.symbol || '-'} #${item.projectID || '-'}`}
                            subtitle={<TruncatedText value={item.contract} copyable={true} />}
                            tags={
                                <Space size={4} wrap={true}>
                                    <ChainBadge chainID={item.chainID} />
                                    {evaluationTag(item.evaluationStatus)}
                                </Space>
                            }
                        />
                        <div className='project-report-pairs'>
                            <PairSummary
                                title='WETH Pair'
                                available={item.reportDataAvailable}
                                created={item.wethPairIsCreated}
                                removeLiquidity={item.wethPairIsRemoveLiquidity}
                                mint={item.wethPairIsMint}
                                quoteUsdtValue={item.wethPairQuoteUsdtValueInt}
                                lastSwapAt={item.wethPairLastSwapAt}
                            />
                            <PairSummary
                                title='USDT Pair'
                                available={item.reportDataAvailable}
                                created={item.usdtPairIsCreated}
                                removeLiquidity={item.usdtPairIsRemoveLiquidity}
                                mint={item.usdtPairIsMint}
                                quoteUsdtValue={item.usdtPairQuoteUsdtValueInt}
                                lastSwapAt={item.usdtPairLastSwapAt}
                            />
                        </div>
                        <KeyValueGrid
                            columns={1}
                            items={[
                                {label: 'Name', value: item.name},
                                {label: 'Attempts', value: item.evaluationAttempts},
                                {label: 'Last Error', value: item.evaluationLastError},
                                {label: 'Source Updated', value: item.sourceUpdatedAt},
                                {label: 'Evaluated', value: item.evaluatedAt},
                                {label: 'Task Updated', value: item.evaluationUpdatedAt},
                                {label: 'Report Created', value: item.createdAt}
                            ]}
                        />
                    </>
                )}
            />
        </AppPage>
    );
};
