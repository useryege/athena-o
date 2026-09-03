import {Alert, Button, Drawer, Empty, InputNumber, Select, Skeleton, Space, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ChoiceGroup, KeyValueGrid, ResourceTable, StatusTag, TruncatedText, useAsyncData} from '../../components';
import {formatBeijingDateTime, formatBlockNumber} from '../../shared/format';
import {usePagedParams} from '../../shared/pages/shared';
import {TokenCollectionTask} from '../../shared/services/token-service';
import {memberServices as services} from '../services';

const DATA_TYPES = ['chain_state', 'wallet_asset_state', 'simulation_result', 'ave', 'contract_code_source', 'wallet_normal_transactions'];
const POLL_INTERVAL_MS = 30_000;
const dataTypeLabel = (value?: string) => ({
    chain_state: 'Chain state',
    wallet_asset_state: 'Wallet asset state',
    simulation_result: 'Simulation result',
    ave: 'Ave',
    contract_code_source: 'Contract source',
    wallet_normal_transactions: 'Pre-deploy transactions'
}[value || ''] || value || '-');

const statusTone = (status?: string) => ({positive: status === 'succeeded', negative: status === 'failed'});

const TaskDrawer = (props: {task?: TokenCollectionTask; loading: boolean; error?: Error; onClose: () => void}) => (
    <Drawer title={props.task ? `${dataTypeLabel(props.task.dataType)} evidence` : 'Collection evidence'} width='min(780px, calc(100vw - 24px))' open={props.loading || Boolean(props.task) || Boolean(props.error)} onClose={props.onClose} keyboard={true}>
        {props.loading ? <Skeleton active={true} /> : props.error ? <Alert type='error' showIcon={true} title='Could not load collection evidence' description={props.error.message} /> : props.task ? (
            <div className='collection-task-detail'>
                <KeyValueGrid columns={2} items={[
                    {label: 'Task', value: props.task.taskID || '-'},
                    {label: 'Project', value: props.task.projectID || '-'},
                    {label: 'Data type', value: dataTypeLabel(props.task.dataType)},
                    {label: 'Status', value: <StatusTag value={props.task.status} {...statusTone(props.task.status)} />},
                    {label: 'Failure count', value: `${props.task.failureCount || 0}/3`},
                    {label: 'Claim generation', value: props.task.claimGeneration || '-'},
                    {label: 'Available', value: formatBeijingDateTime(props.task.availableAt) || '-'},
                    {label: 'Locked', value: formatBeijingDateTime(props.task.lockedAt) || '-'},
                    {label: 'Lease expires', value: formatBeijingDateTime(props.task.leaseExpiresAt) || '-'},
                    {label: 'Finished', value: formatBeijingDateTime(props.task.finishedAt) || '-'},
                    {label: 'Collected', value: formatBeijingDateTime(props.task.result?.collectedAt) || '-'},
                    {label: 'Block', value: formatBlockNumber(props.task.result?.blockNumber)},
                    {label: 'Schema version', value: props.task.result?.schemaVersion || '-'},
                    {label: 'Content hash', value: <TruncatedText value={props.task.result?.contentHash} copyable={true} />},
                    {label: 'Last error', value: <TruncatedText value={props.task.lastError} />}
                ]} />
                {props.task.result?.payloadJSON ? <pre className='code-block project-json-viewer' tabIndex={0} aria-label={`${dataTypeLabel(props.task.dataType)} normalized evidence JSON`}>{props.task.result.payloadJSON}</pre> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={props.task.status === 'succeeded' ? 'The result has no JSON payload' : 'Evidence is available after a successful collection'} />}
            </div>
        ) : null}
    </Drawer>
);

export const CollectionTasksPage = () => {
    const {page, pageSize, setPage} = usePagedParams();
    const [projectID, setProjectID] = React.useState<number>();
    const [dataType, setDataType] = React.useState('');
    const [status, setStatus] = React.useState('');
    const [selectedTask, setSelectedTask] = React.useState<TokenCollectionTask>();
    const [detailLoading, setDetailLoading] = React.useState(false);
    const [detailError, setDetailError] = React.useState<Error>();
    const detailRequest = React.useRef<(Promise<TokenCollectionTask | undefined> & {abort?: () => void})>();
    const data = useAsyncData(() => services.tokenapi.listCollectionTasks({page, pageSize, projectID, dataType: dataType || undefined, status: status || undefined}), [page, pageSize, projectID, dataType, status]);
    const reloadRef = React.useRef(data.reload);
    reloadRef.current = data.reload;
    const hasActiveTasks = Boolean(data.data?.items.some(task => task.status === 'pending' || task.status === 'running'));
    React.useEffect(() => {
        if (!hasActiveTasks) return;
        const timer = window.setInterval(() => reloadRef.current(), POLL_INTERVAL_MS);
        return () => window.clearInterval(timer);
    }, [hasActiveTasks]);
    React.useEffect(() => () => detailRequest.current?.abort?.(), []);

    const openTask = (task: TokenCollectionTask) => {
        if (!task.taskID) return;
        detailRequest.current?.abort?.();
        setSelectedTask(undefined);
        setDetailError(undefined);
        setDetailLoading(true);
        const request = services.tokenapi.getCollectionTask(task.taskID);
        detailRequest.current = request;
        request.then(next => {
            if (detailRequest.current !== request) return;
            setSelectedTask(next);
            setDetailLoading(false);
        }, error => {
            if (detailRequest.current !== request) return;
            setDetailError(error instanceof Error ? error : new Error(String(error)));
            setDetailLoading(false);
        });
    };
    const closeTask = () => {
        detailRequest.current?.abort?.();
        detailRequest.current = undefined;
        setSelectedTask(undefined);
        setDetailLoading(false);
        setDetailError(undefined);
    };
    const columns: ColumnsType<TokenCollectionTask> = [
        {title: 'Task', width: 90, render: item => <Button type='link' onClick={() => openTask(item)}>{item.taskID || '-'}</Button>},
        {title: 'Project', dataIndex: 'projectID', width: 100},
        {title: 'Data Type', width: 190, render: item => dataTypeLabel(item.dataType)},
        {title: 'Status', width: 110, render: item => <StatusTag value={item.status} {...statusTone(item.status)} />},
        {title: 'Failures', width: 90, render: item => `${item.failureCount || 0}/3`},
        {title: 'Available At', width: 180, render: item => formatBeijingDateTime(item.availableAt) || '-'},
        {title: 'Lease Expires', width: 180, render: item => formatBeijingDateTime(item.leaseExpiresAt) || '-'},
        {title: 'Finished', width: 180, render: item => formatBeijingDateTime(item.finishedAt) || '-'},
        {title: 'Last Error', width: 240, render: item => <TruncatedText value={item.lastError} />},
        {title: 'Evidence', width: 110, render: item => item.status === 'succeeded' ? <Tag color='green'>Available</Tag> : <Tag>{item.status === 'failed' ? 'No result' : 'Pending'}</Tag>}
    ];
    return (
        <AppPage title='Collection Tasks' subtitle='Each project owns exactly one task for each of the six one-time data sources.' loading={data.loading} error={data.error} onRefresh={data.reload} filters={
            <Space wrap={true}>
                <InputNumber aria-label='Filter by project ID' value={projectID} placeholder='Project ID' onChange={value => setProjectID(typeof value === 'number' ? value : undefined)} />
                <Select aria-label='Filter by data type' value={dataType || undefined} allowClear={true} placeholder='All data types' options={DATA_TYPES.map(value => ({value, label: dataTypeLabel(value)}))} onChange={value => setDataType(value || '')} style={{minWidth: 210}} />
                <ChoiceGroup<string> ariaLabel='Filter by status' value={status || 'all'} options={[{label: 'All', value: 'all'}, ...['pending', 'running', 'succeeded', 'failed'].map(value => ({value, label: value}))]} onChange={value => setStatus(value === 'all' ? '' : value)} />
            </Space>
        }>
            <ResourceTable rowKey={item => item.taskID || `${item.projectID}-${item.dataType}`} items={data.data?.items || []} columns={columns} loading={data.loading} total={data.data?.total} page={page} pageSize={pageSize} onPageChange={setPage} scrollX={1390} compactRender={item => (
                <Button className='collection-task-compact-button' type='text' onClick={() => openTask(item)}>
                    <span><Typography.Text strong={true}>{dataTypeLabel(item.dataType)}</Typography.Text><Typography.Text type='secondary'>Project #{item.projectID || '-'}</Typography.Text></span>
                    <span><StatusTag value={item.status} {...statusTone(item.status)} /><Typography.Text type='secondary'>{item.failureCount || 0}/3 failures</Typography.Text></span>
                </Button>
            )} compactEmptyDescription='No collection tasks match the filters' />
            <TaskDrawer task={selectedTask} loading={detailLoading} error={detailError} onClose={closeTask} />
        </AppPage>
    );
};
