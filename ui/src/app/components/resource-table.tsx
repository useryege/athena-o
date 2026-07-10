import {Pagination, Segmented, Table, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import type {TableRowSelection} from 'antd/es/table/interface';
import * as React from 'react';
import {PAGE_SIZE_OPTIONS} from '../shared/pagination';
import {useKeyboardPaintSelection} from './keyboard-paint-selection';

export const ResourceTable = <T,>(props: {
    rowKey: keyof T | ((record: T) => React.Key);
    items: T[];
    columns: ColumnsType<T>;
    loading?: boolean;
    page?: number;
    pageSize?: number;
    pageSizeOptions?: number[];
    total?: number;
    onPageChange?: (page: number, pageSize: number) => void;
    onItemClick?: (record: T) => void;
    selectedRowKeys?: React.Key[];
    onSelectionChange?: (keys: React.Key[], records: T[]) => void;
    enableHoverKeyboardSelect?: boolean;
    label?: string;
    scrollX?: number | string;
    stickyHeader?: boolean | {offsetHeader?: number};
    rowClassName?: (record: T, index: number) => string;
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
    const scroll = props.scrollX === undefined ? undefined : {x: props.scrollX};
    const sticky = props.stickyHeader === true ? {offsetHeader: 56} : props.stickyHeader ? {offsetHeader: props.stickyHeader.offsetHeader ?? 56} : undefined;
    const hasPagination = props.total !== undefined && props.onPageChange;
    const regionClassName = ['resource-table-region', hasPagination ? 'resource-table-region--paginated' : undefined, tableClassName].filter(Boolean).join(' ');
    const pageSizeOptions = props.pageSizeOptions || PAGE_SIZE_OPTIONS;
    const currentPage = props.page || 1;
    const currentPageSize = props.pageSize || pageSizeOptions[0];

    return (
        <div className={regionClassName} role='region' aria-label={props.label || 'Data table'} aria-busy={props.loading || undefined}>
            {hasPagination && (
                <div className='resource-table-pagination' aria-label='Table pagination'>
                    <Typography.Text className='resource-table-pagination__total' type='secondary'>
                        {props.total} {props.total === 1 ? 'item' : 'items'}
                    </Typography.Text>
                    <Pagination
                        current={currentPage}
                        pageSize={currentPageSize}
                        total={props.total}
                        size='small'
                        showLessItems={true}
                        showSizeChanger={false}
                        onChange={nextPage => props.onPageChange?.(nextPage, currentPageSize)}
                    />
                    <div className='resource-table-pagination__sizes'>
                        <Typography.Text className='resource-table-pagination__sizes-label' type='secondary'>
                            Per page
                        </Typography.Text>
                        <Segmented<number>
                            aria-label='Items per page'
                            size='small'
                            value={currentPageSize}
                            options={pageSizeOptions.map(value => ({label: String(value), value}))}
                            onChange={nextPageSize => props.onPageChange?.(1, nextPageSize)}
                        />
                    </div>
                </div>
            )}
            <Table<T>
                className='resource-table'
                size='small'
                sticky={sticky}
                rowKey={props.rowKey as any}
                columns={props.columns as any}
                dataSource={props.items}
                loading={props.loading}
                rowSelection={rowSelection}
                pagination={false}
                scroll={scroll}
                onRow={tableOnRow}
                rowClassName={props.rowClassName}
            />
        </div>
    );
};
