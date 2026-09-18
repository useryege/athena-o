import {operationLogMetricValue, operationLogSnapshotSearch} from './operation-log-service';

test('operation log metric values accept protobuf wrappers and JSON scalars', () => {
    expect(operationLogMetricValue({value: true})).toBe(true);
    expect(operationLogMetricValue('0')).toBe('0');
    expect(operationLogMetricValue({value: '7'})).toBe('7');
    expect(operationLogMetricValue(undefined)).toBeUndefined();
});

test('operation log detail links preserve the list snapshot token and filters', () => {
    expect(operationLogSnapshotSearch('?outcome=SUCCEEDED', 'snapshot-token')).toBe('?outcome=SUCCEEDED&snapshot_token=snapshot-token');
    expect(operationLogSnapshotSearch('?snapshot_token=old', '')).toBe('');
});
