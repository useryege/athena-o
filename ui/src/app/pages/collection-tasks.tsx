import {Input, InputNumber, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ChoiceGroup, ResourceTable, TruncatedText, useAsyncData} from '../components';
import {services} from '../shared/services';
import {TokenCollectionTask} from '../shared/services/token-service';
import {usePagedParams} from './shared';

export const CollectionTasksPage = () => {
    const {page, pageSize, setPage} = usePagedParams();
    const [projectID, setProjectID] = React.useState<number>();
    const [dataType, setDataType] = React.useState('');
    const [status, setStatus] = React.useState('');
    const data = useAsyncData(
        () =>
            services.tokenapi.listCollectionTasks({
                page,
                pageSize,
                projectID,
                dataType: dataType || undefined,
                status: status || undefined
            }),
        [page, pageSize, projectID, dataType, status]
    );
    const columns: ColumnsType<TokenCollectionTask> = [
		{title: 'Task', dataIndex: 'taskID'},
        {title: 'Project', dataIndex: 'projectID'},
        {title: 'Data Type', dataIndex: 'dataType'},
		{title: 'Revision', dataIndex: 'revision'},
        {title: 'Status', dataIndex: 'status'},
        {title: 'Attempts', dataIndex: 'attempts'},
		{title: 'Available At', dataIndex: 'availableAt'},
		{title: 'Lease Expires', dataIndex: 'leaseExpiresAt'},
        {title: 'Last Error', render: item => <TruncatedText value={item.lastError} />},
        {title: 'Created', dataIndex: 'createdAt'}
    ];
    return (
        <AppPage
            title='Collection Tasks'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            filters={
                <Space wrap={true}>
                    <InputNumber
                        aria-label='Filter by project ID'
                        value={projectID}
                        placeholder='Project ID'
                        onChange={value => setProjectID(typeof value === 'number' ? value : undefined)}
                    />
                    <Input aria-label='Filter by data type' value={dataType} placeholder='Data type' onChange={event => setDataType(event.target.value)} />
                    <ChoiceGroup<string>
                        ariaLabel='Filter by status'
                        value={status || 'all'}
						options={[{label: 'All', value: 'all'}, ...['pending', 'running', 'succeeded', 'failed'].map(value => ({value, label: value}))]}
                        onChange={value => {
                            setStatus(value === 'all' ? '' : value);
                        }}
                    />
                </Space>
            }>
            <ResourceTable
				rowKey={item => item.taskID || `${item.projectID}-${item.dataType}-${item.revision}`}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
            />
        </AppPage>
    );
};
