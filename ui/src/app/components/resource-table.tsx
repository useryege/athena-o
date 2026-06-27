import {Table} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import type {TableRowSelection} from 'antd/es/table/interface';
import * as React from 'react';
import {useKeyboardPaintSelection} from './keyboard-paint-selection';

export const ResourceTable = <T,>(props: {
    rowKey: keyof T | ((record: T) => React.Key);
    items: T[];
    columns: ColumnsType<T>;
    loading?: boolean;
    page?: number;
    pageSize?: number;
    total?: number;
    onPageChange?: (page: number, pageSize: number) => void;
    onItemClick?: (record: T) => void;
    selectedRowKeys?: React.Key[];
    onSelectionChange?: (keys: React.Key[], records: T[]) => void;
    enableHoverKeyboardSelect?: boolean;
    label?: string;
}) => {
    const selectedKeys = props.selectedRowKeys || [];
    const itemKey = (item: T) => (typeof props.rowKey === 'function' ? props.rowKey(item) : (item[props.rowKey] as React.Key));
    const paintEnabled = Boolean(props.enableHoverKeyboardSelect && props.onSelectionChange);
    const keyboardOnlySelection = paintEnabled;
    const selectionVisible = !keyboardOnlySelection || selectedKeys.length > 0;
    const {tableClassName, getRowMouseHandlers} = useKeyboardPaintSelection({
        enabled: paintEnabled,
        items: props.items,
        itemKey,
        selectedKeys,
        onSelectionChange: props.onSelectionChange
    });

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
                            role: 'button',
                            tabIndex: 0,
                            onClick: () => props.onItemClick?.(record),
                            onKeyDown: (event: React.KeyboardEvent) => {
                                if (event.key === 'Enter' || event.key === ' ') {
                                    event.preventDefault();
                                    props.onItemClick?.(record);
                                }
                            }
                        }
                      : {};
                  return {...click, ...mouse};
              }
            : undefined;

    return (
        <div className={tableClassName} role='region' aria-label={props.label || 'Data table'} aria-busy={props.loading || undefined}>
            <Table<T>
                className='resource-table'
                size='small'
                sticky={{offsetHeader: 56}}
                rowKey={props.rowKey as any}
                columns={props.columns as any}
                dataSource={props.items}
                loading={props.loading}
                rowSelection={rowSelection}
                pagination={
                    props.total !== undefined && props.onPageChange
                        ? {
                              current: props.page,
                              pageSize: props.pageSize,
                              total: props.total,
                              showSizeChanger: true,
                              showTotal: total => `${total} items`,
                              onChange: props.onPageChange
                          }
                        : false
                }
                scroll={{x: 'max-content'}}
                onRow={tableOnRow}
            />
        </div>
    );
};
