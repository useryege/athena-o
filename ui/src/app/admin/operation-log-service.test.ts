import {operationLogMetricValue} from './operation-log-service';

test('operation log metric values accept protobuf wrappers and JSON scalars', () => {
    expect(operationLogMetricValue({value: true})).toBe(true);
    expect(operationLogMetricValue('0')).toBe('0');
    expect(operationLogMetricValue({value: '7'})).toBe('7');
    expect(operationLogMetricValue(undefined)).toBeUndefined();
});
