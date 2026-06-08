import {Card, Checkbox, Empty, Pagination, Skeleton, Table} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import type {TableRowSelection} from 'antd/es/table/interface';
import * as React from 'react';
import {useBreakpoint} from './data';

type PaintMode = 'select' | 'deselect';

const isTypingTarget = (target: EventTarget | null): boolean => {
    if (!(target instanceof HTMLElement)) {
        return false;
    }
    const tag = target.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') {
        return true;
    }
    if (target.isContentEditable) {
        return true;
    }
    if (target.closest('.ant-select')) {
        return true;
    }
    return false;
};

const useKeyboardPaintSelection = <T,>(options: {
    enabled: boolean;
    items: T[];
    itemKey: (item: T) => React.Key;
    selectedKeys: React.Key[];
    onSelectionChange?: (keys: React.Key[], records: T[]) => void;
}) => {
    const [xHeld, setXHeld] = React.useState(false);
    const paintModeRef = React.useRef<PaintMode | null>(null);
    const paintedKeysRef = React.useRef(new Set<React.Key>());
    const selectedKeysRef = React.useRef(options.selectedKeys);
    const itemsRef = React.useRef(options.items);
    const hoveredKeyRef = React.useRef<React.Key | null>(null);
    const xHeldRef = React.useRef(false);
    const onSelectionChangeRef = React.useRef(options.onSelectionChange);
    const itemKeyRef = React.useRef(options.itemKey);

    selectedKeysRef.current = options.selectedKeys;
    itemsRef.current = options.items;
    onSelectionChangeRef.current = options.onSelectionChange;
    itemKeyRef.current = options.itemKey;
    xHeldRef.current = xHeld;

    const applyPaint = React.useCallback((key: React.Key) => {
        const onSelectionChange = onSelectionChangeRef.current;
        if (!onSelectionChange) {
            return;
        }
        if (paintedKeysRef.current.has(key)) {
            return;
        }

        const item = itemsRef.current.find(value => itemKeyRef.current(value) === key);
        if (!item) {
            return;
        }

        const selected = selectedKeysRef.current.includes(key);
        if (paintModeRef.current === null) {
            paintModeRef.current = selected ? 'deselect' : 'select';
        }

        const shouldSelect = paintModeRef.current === 'select';
        if (selected === shouldSelect) {
            paintedKeysRef.current.add(key);
            return;
        }

        const nextKeys = shouldSelect ? [...selectedKeysRef.current, key] : selectedKeysRef.current.filter(value => value !== key);
        const nextKeySet = new Set(nextKeys);
        selectedKeysRef.current = nextKeys;
        paintedKeysRef.current.add(key);
        onSelectionChange(
            nextKeys,
            itemsRef.current.filter(value => nextKeySet.has(itemKeyRef.current(value)))
        );
    }, []);

    React.useEffect(() => {
        if (!options.enabled) {
            return;
        }

        const resetGesture = () => {
            paintModeRef.current = null;
            paintedKeysRef.current = new Set();
            xHeldRef.current = false;
            setXHeld(false);
        };

        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key !== 'x' && event.key !== 'X') {
                return;
            }
            if (event.repeat) {
                return;
            }
            if (isTypingTarget(event.target)) {
                return;
            }

            xHeldRef.current = true;
            setXHeld(true);
            if (hoveredKeyRef.current != null) {
                applyPaint(hoveredKeyRef.current);
            }
        };

        const onKeyUp = (event: KeyboardEvent) => {
            if (event.key !== 'x' && event.key !== 'X') {
                return;
            }
            resetGesture();
        };

        const onBlur = () => resetGesture();

        window.addEventListener('keydown', onKeyDown);
        window.addEventListener('keyup', onKeyUp);
        window.addEventListener('blur', onBlur);
        return () => {
            window.removeEventListener('keydown', onKeyDown);
            window.removeEventListener('keyup', onKeyUp);
            window.removeEventListener('blur', onBlur);
            resetGesture();
        };
    }, [applyPaint, options.enabled]);

    const getRowMouseHandlers = React.useCallback(
        (key: React.Key) => ({
            onMouseEnter: () => {
                hoveredKeyRef.current = key;
                if (xHeldRef.current) {
                    applyPaint(key);
                }
            },
            onMouseLeave: () => {
                if (hoveredKeyRef.current === key) {
                    hoveredKeyRef.current = null;
                }
            }
        }),
        [applyPaint]
    );

    return {
        listClassName: xHeld ? 'resource-list--paint-select' : undefined,
        getRowMouseHandlers
    };
};

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
