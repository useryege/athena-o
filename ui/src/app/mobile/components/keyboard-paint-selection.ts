import * as React from 'react';

type PaintMode = 'select' | 'deselect';

const isTypingTarget = (target: EventTarget | null): boolean => {
    if (!(target instanceof HTMLElement)) {
        return false;
    }
    const tag = target.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') {
        return true;
    }
    if (target.isContentEditable || target.closest('.ant-select')) {
        return true;
    }
    return false;
};

export const useKeyboardPaintSelection = <T>(options: {
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
        if (!onSelectionChange || paintedKeysRef.current.has(key)) {
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
            if (event.key === 'Escape') {
                if (!isTypingTarget(event.target) && selectedKeysRef.current.length > 0) {
                    selectedKeysRef.current = [];
                    onSelectionChangeRef.current?.([], []);
                }
                return;
            }
            if ((event.key !== 'x' && event.key !== 'X') || event.repeat || isTypingTarget(event.target)) {
                return;
            }

            xHeldRef.current = true;
            setXHeld(true);
            if (hoveredKeyRef.current != null) {
                applyPaint(hoveredKeyRef.current);
            }
        };

        const onKeyUp = (event: KeyboardEvent) => {
            if (event.key === 'x' || event.key === 'X') {
                resetGesture();
            }
        };

        window.addEventListener('keydown', onKeyDown);
        window.addEventListener('keyup', onKeyUp);
        window.addEventListener('blur', resetGesture);
        return () => {
            window.removeEventListener('keydown', onKeyDown);
            window.removeEventListener('keyup', onKeyUp);
            window.removeEventListener('blur', resetGesture);
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
