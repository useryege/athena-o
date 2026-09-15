import {Empty, Pagination, Skeleton, Table, Typography} from 'antd';
import type {ColumnsType, TableProps} from 'antd/es/table';
import type {TableRowSelection} from 'antd/es/table/interface';
import * as React from 'react';
import {PAGE_SIZE_OPTIONS} from '../shared/pagination';
import {ChoiceGroup} from './choice-group';
import {useKeyboardPaintSelection} from './keyboard-paint-selection';

export const ResourceTable = <T,>(props: {
    rowKey: keyof T | ((record: T) => React.Key);
    items: T[];
    columns: ColumnsType<T>;
    loading?: boolean;
    /** Whether this source has completed a successful read (including an empty result). */
    hasData?: boolean;
    page?: number;
    pageSize?: number;
    pageSizeOptions?: number[];
    total?: number;
    onPageChange?: (page: number, pageSize: number) => void;
    onChange?: TableProps<T>['onChange'];
    onItemClick?: (record: T) => void;
    selectedRowKeys?: React.Key[];
    onSelectionChange?: (keys: React.Key[], records: T[]) => void;
    enableHoverKeyboardSelect?: boolean;
    label?: string;
    scrollX?: number | string;
    stickyHeader?: boolean | {offsetHeader?: number};
    rowClassName?: (record: T, index: number) => string;
    compactRender?: (record: T) => React.ReactNode;
    compactEmptyDescription?: React.ReactNode;
    expandable?: TableProps<T>['expandable'];
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
    const sticky = props.stickyHeader === true ? {offsetHeader: 64} : props.stickyHeader ? {offsetHeader: props.stickyHeader.offsetHeader ?? 64} : undefined;
    const hasPagination = props.total !== undefined && props.onPageChange;
    const regionClassName = [
        'resource-table-region',
        hasPagination ? 'resource-table-region--paginated' : undefined,
        props.compactRender ? 'resource-table-region--compact' : undefined,
        tableClassName
    ]
        .filter(Boolean)
        .join(' ');
    const pageSizeOptions = props.pageSizeOptions || PAGE_SIZE_OPTIONS;
    const currentPage = props.page || 1;
    const currentPageSize = props.pageSize || pageSizeOptions[0];

    if (props.hasData === false) {
        return props.loading ? (
            <div role='status' aria-label='Loading items'>
                <Skeleton active />
            </div>
        ) : null;
    }

    return (
        <div className={regionClassName} role='region' aria-label={props.label || 'Data table'} aria-busy={props.loading || undefined}>
            {hasPagination && (
                <div className='resource-table-pagination' role='navigation' aria-label='Table pagination'>
                    <div className='resource-table-pagination__controls'>
                        <Typography.Text className='resource-table-pagination__total athena-number' type='secondary'>
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
                        {pageSizeOptions.length > 1 && (
                            <div className='resource-table-pagination__sizes'>
                                <Typography.Text className='resource-table-pagination__sizes-label' type='secondary'>
                                    Per page
                                </Typography.Text>
                                <ChoiceGroup<number>
                                    ariaLabel='Items per page'
                                    size='small'
                                    value={currentPageSize}
                                    options={pageSizeOptions.map(value => ({label: String(value), value}))}
                                    onChange={nextPageSize => props.onPageChange?.(1, nextPageSize)}
                                />
                            </div>
                        )}
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
                onChange={props.onChange}
                onRow={tableOnRow}
                rowClassName={props.rowClassName}
                expandable={props.expandable}
            />
            {props.compactRender && (
                <div className='resource-table-compact' role='group' aria-label={`${props.label || 'Data table'} compact view`}>
                    {props.loading ? (
                        <ul className='resource-table-compact__items resource-table-compact__loading' aria-label='Loading items'>
                            {[0, 1, 2].map(index => (
                                <li className='resource-table-compact__item resource-table-compact__skeleton-card' key={index}>
                                    <Skeleton active={true} paragraph={{rows: 5}} />
                                </li>
                            ))}
                        </ul>
                    ) : props.items.length > 0 ? (
                        <ul className='resource-table-compact__items'>
                            {props.items.map(item => (
                                <li className='resource-table-compact__item' key={itemKey(item)}>
                                    {props.compactRender?.(item)}
                                </li>
                            ))}
                        </ul>
                    ) : (
                        <Empty className='resource-table-compact__empty' image={Empty.PRESENTED_IMAGE_SIMPLE} description={props.compactEmptyDescription || 'No items'} />
                    )}
                </div>
            )}
        </div>
    );
};
