import {Card, Checkbox, Empty, Pagination, Skeleton, Table} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import type {TableRowSelection} from 'antd/es/table/interface';
import * as React from 'react';
import {useBreakpoint} from './data';
import {useKeyboardPaintSelection} from './keyboard-paint-selection';

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
    enableHoverKeyboardSelect?: boolean;
}) => {
    const {isMobile} = useBreakpoint();
    const selectedKeys = props.selectedRowKeys || [];
    const selectedKeySet = React.useMemo(() => new Set(selectedKeys), [selectedKeys]);
    const itemKey = (item: T) => (typeof props.rowKey === 'function' ? props.rowKey(item) : (item[props.rowKey] as React.Key));
    const paintEnabled = Boolean(props.enableHoverKeyboardSelect && props.onSelectionChange);
    const keyboardOnlySelection = paintEnabled;
    const selectionVisible = !keyboardOnlySelection || selectedKeys.length > 0;
    const {listClassName, getRowMouseHandlers} = useKeyboardPaintSelection({
        enabled: paintEnabled,
        items: props.items,
        itemKey,
        selectedKeys,
        onSelectionChange: props.onSelectionChange
    });
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
    const listClassNames = ['resource-list', isMobile ? 'resource-list--mobile' : undefined, listClassName].filter(Boolean).join(' ');

    if (isMobile) {
        return (
            <div className={listClassNames}>
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
                        <Card key={key} size='small' {...clickProps} {...(paintEnabled ? getRowMouseHandlers(key) : {})}>
                            {props.onSelectionChange && selectionVisible && (
                                <Checkbox
                                    className='resource-card__select'
                                    checked={selectedKeySet.has(key)}
                                    disabled={keyboardOnlySelection}
                                    onClick={event => event.stopPropagation()}
                                    onChange={keyboardOnlySelection ? undefined : event => handleSelectionChange(key, item, event.target.checked)}
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

    const rowSelection: TableRowSelection<T> | undefined =
        props.onSelectionChange && selectionVisible
            ? {
                  selectedRowKeys: selectedKeys,
                  hideSelectAll: keyboardOnlySelection,
                  getCheckboxProps: keyboardOnlySelection ? () => ({disabled: true}) : undefined,
                  onChange: keyboardOnlySelection ? () => {} : (keys, records) => props.onSelectionChange?.(keys, records)
              }
            : undefined;
    const tableOnRow =
        paintEnabled || props.onItemClick
            ? (record: T) => {
                  const key = itemKey(record);
                  const mouse = paintEnabled ? getRowMouseHandlers(key) : {};
                  const click = props.onItemClick
                      ? {
                            className: 'resource-table__row--clickable',
                            onClick: () => props.onItemClick?.(record)
                        }
                      : {};
                  return {...click, ...mouse};
              }
            : undefined;

    return (
        <div className={listClassNames}>
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
                onRow={tableOnRow}
            />
        </div>
    );
};
