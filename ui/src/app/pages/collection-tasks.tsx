import {Input, InputNumber, Select, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ResourceTable, TruncatedText, useAsyncData} from '../components';
import {services} from '../shared/services';
import {TokenAPIProjectDataCollectionTask} from '../shared/services/tokenapi-service';
import {usePagedParams} from './shared';

export const CollectionTasksPage = () => {
    const {page, pageSize, setPage} = usePagedParams();
    const [projectID, setProjectID] = React.useState<number>();
    const [dataType, setDataType] = React.useState('');
    const [status, setStatus] = React.useState('');
    const data = useAsyncData(
        () =>
            services.tokenapi.listProjectDataCollectionTasks({
                page,
                pageSize,
                projectID,
                dataType: dataType || undefined,
                status: status || undefined
            }),
        [page, pageSize, projectID, dataType, status]
    );
    const columns: ColumnsType<TokenAPIProjectDataCollectionTask> = [
        {title: 'Project', dataIndex: 'projectID'},
        {title: 'Data Type', dataIndex: 'dataType'},
        {title: 'Status', dataIndex: 'status'},
        {title: 'Attempts', dataIndex: 'attempts'},
        {title: 'Next Attempt', dataIndex: 'nextAttemptAt'},
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
                    <Select
                        allowClear={true}
                        aria-label='Filter by status'
                        value={status || undefined}
                        placeholder='Status'
                        style={{width: 150}}
                        onChange={value => setStatus(value || '')}
                        options={['pending', 'succeeded', 'failed'].map(value => ({value, label: value}))}
                    />
                </Space>
            }>
            <ResourceTable
                rowKey={item => `${item.projectID}-${item.dataType}`}
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
