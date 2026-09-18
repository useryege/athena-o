import {Alert, Button, Descriptions, Drawer, Empty, Input, Select, Space, Table, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useLocation, useNavigate, useParams} from 'react-router-dom';
import {AppPage, Section} from '../../components';
import {formatBeijingDateTime} from '../../shared/format';
import {useVisibleQuery} from '../../shared/use-visible-query';
import {useAdminReadScope} from '../read-scope';
import {adminServices as services} from '../services';
import {operationLogMetricValue, type OperationLogDetail, type OperationLogSummary} from '../operation-log-service';

const outcomeColor = (outcome: string) => {
    if (outcome === 'SUCCEEDED' || outcome === 'ACCEPTED') return 'success';
    if (outcome === 'FAILED' || outcome === 'DENIED') return 'error';
    if (outcome === 'PARTIAL') return 'warning';
    return 'default';
};

const RuntimeStatus = () => {
    const runtime = useVisibleQuery(() => services.operationLogs.getRuntimeStatus(), useAdminReadScope('operation-log-runtime'), 10000);
    const capture = useVisibleQuery(() => services.operationLogs.getCaptureStatus(), useAdminReadScope('operation-log-capture'), 10000);
    const persistenceReachable = operationLogMetricValue(capture.data?.persistenceReachable);
    const confirmedEvents = operationLogMetricValue(capture.data?.confirmedEvents) || '0';
    const inFlightEvents = operationLogMetricValue(capture.data?.inFlightEvents) || '0';
    const lastFailureCode = operationLogMetricValue(capture.data?.lastFailureCode) || '—';
    return (
        <Section
            title='Capture status'
            extra={
                <Button
                    onClick={() => {
                        runtime.reload();
                        capture.reload();
                    }}>
                    Refresh
                </Button>
            }>
            {(runtime.error || capture.error) && <Alert type='warning' showIcon title='Operation log status unavailable' description={(runtime.error || capture.error)?.message} />}
            <Descriptions size='small' column={{xs: 1, sm: 2, md: 4}}>
                <Descriptions.Item label='Projection'>{runtime.data?.projectionState || 'Unavailable'}</Descriptions.Item>
                <Descriptions.Item label='Query ready'>{runtime.data?.queryReady === undefined ? 'Unavailable' : runtime.data.queryReady ? 'Yes' : 'No'}</Descriptions.Item>
                <Descriptions.Item label='Pending'>{runtime.data?.pendingEvents || '0'}</Descriptions.Item>
                <Descriptions.Item label='Observed'>{runtime.data?.totalObserved || '0'}</Descriptions.Item>
                <Descriptions.Item label='Persistence'>{persistenceReachable === undefined ? 'Unavailable' : persistenceReachable ? 'Reachable' : 'Unavailable'}</Descriptions.Item>
                <Descriptions.Item label='Confirmed'>{confirmedEvents}</Descriptions.Item>
                <Descriptions.Item label='In flight'>{inFlightEvents}</Descriptions.Item>
                <Descriptions.Item label='Last failure'>{lastFailureCode}</Descriptions.Item>
            </Descriptions>
        </Section>
    );
};

const DetailDrawer = (props: {item?: OperationLogDetail; open: boolean; onClose: () => void; loading: boolean}) => (
    <Drawer title='Operation details' width={Math.min(760, typeof window === 'undefined' ? 760 : window.innerWidth - 24)} open={props.open} onClose={props.onClose}>
        {props.loading && <Typography.Text>Loading operation…</Typography.Text>}
        {!props.loading && !props.item && <Empty description='Operation detail unavailable' />}
        {props.item && (
            <Space direction='vertical' size='large' style={{width: '100%'}}>
                <Descriptions bordered size='small' column={1}>
                    <Descriptions.Item label='Operation ID'>
                        <span className='athena-identifier'>{props.item.operationId}</span>
                    </Descriptions.Item>
                    <Descriptions.Item label='Action'>{props.item.actionCode}</Descriptions.Item>
                    <Descriptions.Item label='Outcome'>
                        <Tag color={outcomeColor(props.item.outcome)}>{props.item.outcome || 'UNKNOWN'}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label='Actor'>
                        {props.item.actorUsername || props.item.actorAccountId || 'Unknown'} · {props.item.actorRole || 'UNKNOWN'}
                    </Descriptions.Item>
                    <Descriptions.Item label='Started'>{formatBeijingDateTime(props.item.startedAt) || 'Unavailable'}</Descriptions.Item>
                    <Descriptions.Item label='Finished'>{formatBeijingDateTime(props.item.finishedAt) || 'Unavailable'}</Descriptions.Item>
                    <Descriptions.Item label='Reason'>{props.item.reasonCode || '—'}</Descriptions.Item>
                    <Descriptions.Item label='Effect'>{props.item.effects.length ? props.item.effects.join(', ') : '—'}</Descriptions.Item>
                </Descriptions>
                <Section title='Resources'>
                    {props.item.resources.length ? (
                        <Table
                            size='small'
                            pagination={false}
                            rowKey={(row: any, index) => `${row.type}-${row.id}-${index}`}
                            dataSource={props.item.resources}
                            columns={[
                                {title: 'Type', dataIndex: 'type'},
                                {title: 'ID', dataIndex: 'id', render: value => <span className='athena-identifier'>{value}</span>},
                                {title: 'Verified', dataIndex: 'referenceVerified', render: value => (value?.value ?? value ? 'Yes' : 'No')}
                            ]}
                        />
                    ) : (
                        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No resource facts' />
                    )}
                </Section>
                <Section title='Protocol'>
                    <Descriptions size='small' column={1}>
                        <Descriptions.Item label='gRPC'>{props.item.protocol?.grpcCode?.value || '—'}</Descriptions.Item>
                        <Descriptions.Item label='HTTP'>{props.item.protocol?.httpStatus?.value || '—'}</Descriptions.Item>
                        <Descriptions.Item label='Response write'>{props.item.protocol?.responseWriteFailed?.value ? 'Failed' : '—'}</Descriptions.Item>
                    </Descriptions>
                </Section>
            </Space>
        )}
    </Drawer>
);

export const OperationLogsPage = () => {
    const scope = useAdminReadScope('operation-logs');
    return scope.isCurrent() ? (
        <OperationLogsWorkspace />
    ) : (
        <AppPage title='Operation Logs' loading>
            <p>Checking administrator access…</p>
        </AppPage>
    );
};

const OperationLogsWorkspace = () => {
    const navigate = useNavigate();
    const location = useLocation();
    const params = useParams<{id?: string}>();
    const [outcome, setOutcome] = React.useState('');
    const [moduleCode, setModuleCode] = React.useState('');
    const [actorQuery, setActorQuery] = React.useState('');
    const [cursor, setCursor] = React.useState('');
    const data = useVisibleQuery(
        () => services.operationLogs.list({pageSize: 30, cursor, outcome: outcome || undefined, moduleCode: moduleCode || undefined, actorQuery: actorQuery || undefined}),
        useAdminReadScope(`operation-logs-${cursor}-${outcome}-${moduleCode}-${actorQuery}`),
        10000
    );
    const detail = useVisibleQuery(
        () => (params.id ? services.operationLogs.get(params.id) : Promise.resolve(undefined)),
        useAdminReadScope(`operation-log-detail-${params.id || 'none'}`),
        10000
    );
    const closeDetail = () => navigate(`/operation-logs${location.search}`);
    const columns: ColumnsType<OperationLogSummary> = [
        {title: 'Started', render: item => formatBeijingDateTime(item.startedAt) || '—'},
        {
            title: 'Action',
            render: item => (
                <Button type='link' onClick={() => navigate(`/operation-logs/${encodeURIComponent(item.operationId)}${location.search}`)}>
                    {item.actionCode}
                </Button>
            )
        },
        {title: 'Actor', render: item => item.actorUsername || item.actorAccountId || 'Unknown'},
        {title: 'Resource', render: item => (item.primaryResourceType ? `${item.primaryResourceType} · ${item.primaryResourceId}` : '—')},
        {title: 'Outcome', dataIndex: 'outcome', render: value => <Tag color={outcomeColor(String(value || 'UNKNOWN'))}>{value || 'UNKNOWN'}</Tag>},
        {title: 'Duration', render: item => (item.durationMs ? `${item.durationMs} ms` : '—')}
    ];
    return (
        <AppPage
            title='Operation Logs'
            subtitle='Immutable business operation history with an independent capture and projection status source.'
            onRefresh={data.reload}
            loading={data.loading}
            error={data.error}
            stale={data.stale}
            filters={
                <Space wrap>
                    <Input
                        allowClear
                        placeholder='Actor or account'
                        value={actorQuery}
                        onChange={event => {
                            setActorQuery(event.target.value);
                            setCursor('');
                        }}
                    />
                    <Select
                        allowClear
                        placeholder='Outcome'
                        value={outcome || undefined}
                        onChange={value => {
                            setOutcome(value || '');
                            setCursor('');
                        }}
                        options={['SUCCEEDED', 'ACCEPTED', 'FAILED', 'DENIED', 'PARTIAL', 'UNKNOWN'].map(value => ({label: value, value}))}
                    />
                    <Input
                        allowClear
                        placeholder='Module code'
                        value={moduleCode}
                        onChange={event => {
                            setModuleCode(event.target.value);
                            setCursor('');
                        }}
                    />
                    <Button
                        onClick={() => {
                            setActorQuery('');
                            setOutcome('');
                            setModuleCode('');
                            setCursor('');
                        }}>
                        Clear
                    </Button>
                </Space>
            }>
            <Space direction='vertical' size='large' style={{width: '100%'}}>
                <RuntimeStatus />
                <Section title='History' extra={<span>{data.data?.items.length || 0} shown</span>}>
                    <Table
                        rowKey='operationId'
                        dataSource={data.data?.items || []}
                        columns={columns}
                        pagination={false}
                        locale={{emptyText: data.loading ? 'Loading operation history…' : 'No operation history'}}
                    />
                    <Space style={{marginTop: 16}}>
                        <Button disabled={!cursor} onClick={() => setCursor('')}>
                            First page
                        </Button>
                        <Button disabled={!data.data?.nextCursor} onClick={() => setCursor(data.data?.nextCursor || '')}>
                            Next page
                        </Button>
                    </Space>
                </Section>
            </Space>
            <DetailDrawer item={detail.data} open={Boolean(params.id)} onClose={closeDetail} loading={detail.loading} />
        </AppPage>
    );
};
