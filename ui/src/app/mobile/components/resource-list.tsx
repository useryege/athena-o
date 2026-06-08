import {Card, Checkbox, Empty, Pagination, Skeleton, Table} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import type {TableRowSelection} from 'antd/es/table/interface';
import * as React from 'react';
import {useBreakpoint} from './data';

export const ResponsiveResourceList = <T,>(props: {
    rowKey: keyof T | ((record: T) => React.Key);
    items: T[];
    columns: ColumnsType<T>;
    loading?: boolean;
    emptyText?: string;
    card: (record: T) => React.ReactNode;
    page?: number;
    pageSize?: number;
    total?: number;
    onPageChange?: (page: number, pageSize: number) => void;
    onItemClick?: (record: T) => void;
    selectedRowKeys?: React.Key[];
    onSelectionChange?: (keys: React.Key[], records: T[]) => void;
}) => {
    const {isMobile} = useBreakpoint();
    const selectedKeys = props.selectedRowKeys || [];
    const selectedKeySet = React.useMemo(() => new Set(selectedKeys), [selectedKeys]);
    const itemKey = (item: T) => (typeof props.rowKey === 'function' ? props.rowKey(item) : (item[props.rowKey] as React.Key));
    const handleSelectionChange = (key: React.Key, item: T, checked: boolean) => {
        const nextKeys = checked ? [...selectedKeys, key] : selectedKeys.filter(value => value !== key);
        const nextKeySet = new Set(nextKeys);
        props.onSelectionChange?.(
            nextKeys,
            props.items.filter(value => nextKeySet.has(itemKey(value)))
        );
    };
    const handleCardKeyDown = (event: React.KeyboardEvent, item: T) => {
        if (!props.onItemClick || (event.key !== 'Enter' && event.key !== ' ')) {
            return;
        }
        event.preventDefault();
        props.onItemClick(item);
    };
    if (isMobile) {
        return (
            <div className='resource-list resource-list--mobile'>
                {props.loading && <Skeleton active={true} />}
                {!props.loading && props.items.length === 0 && <Empty description={props.emptyText || 'No data'} />}
                {props.items.map(item => {
                    const key = itemKey(item);
                    const clickProps = props.onItemClick
                        ? {
                              className: 'resource-card resource-card--clickable',
                              onClick: () => props.onItemClick?.(item),
                              onKeyDown: (event: React.KeyboardEvent) => handleCardKeyDown(event, item),
                              role: 'link',
                              tabIndex: 0
                          }
                        : {className: 'resource-card'};
                    return (
                        <Card key={key} size='small' {...clickProps}>
                            {props.onSelectionChange && (
                                <Checkbox
                                    className='resource-card__select'
                                    checked={selectedKeySet.has(key)}
                                    onClick={event => event.stopPropagation()}
                                    onChange={event => handleSelectionChange(key, item, event.target.checked)}
                                />
                            )}
                            {props.card(item)}
                        </Card>
                    );
                })}
                {props.total !== undefined && props.onPageChange && (
                    <Pagination className='resource-list__pager' current={props.page} pageSize={props.pageSize} total={props.total} simple={true} onChange={props.onPageChange} />
                )}
            </div>
        );
    }
    const rowSelection: TableRowSelection<T> | undefined = props.onSelectionChange
        ? {
              selectedRowKeys: selectedKeys,
              onChange: (keys, records) => props.onSelectionChange?.(keys, records)
          }
        : undefined;
    return (
        <Table<T>
            className='resource-table'
            rowKey={props.rowKey as any}
            columns={props.columns as any}
            dataSource={props.items}
            loading={props.loading}
            rowSelection={rowSelection}
            pagination={
                props.total !== undefined && props.onPageChange
                    ? {current: props.page, pageSize: props.pageSize, total: props.total, showSizeChanger: true, onChange: props.onPageChange}
                    : false
            }
            scroll={{x: 'max-content'}}
            onRow={
                props.onItemClick
                    ? record => ({
                          className: 'resource-table__row--clickable',
                          onClick: () => props.onItemClick?.(record)
                      })
                    : undefined
            }
        />
    );
};
