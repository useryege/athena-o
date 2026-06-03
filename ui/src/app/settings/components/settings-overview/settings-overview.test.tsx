import * as React from 'react';
import {act, create} from 'react-test-renderer';

jest.mock('../../../shared/components', () => ({
    Page: ({children}: {children: React.ReactNode}) => <div>{children}</div>
}));

import {SettingsOverview} from './settings-overview';

const nodeText = (node: any): string => {
    if (typeof node === 'string') {
        return node;
    }
    if (!node?.children) {
        return '';
    }
    return node.children.map(nodeText).join('');
};

describe('settings overview', () => {
    it('shows the application discovery entry', () => {
        const goto = jest.fn();
        const renderer = create(SettingsOverview({}, {apis: {navigation: {goto}}}) as React.ReactElement);

        expect(nodeText(renderer.root)).toContain('Application Discovery');
        expect(nodeText(renderer.root)).toContain('Control project discovery indexer');

        const panel = renderer.root
            .findAllByProps({className: 'settings-overview__redirect-panel'})
            .find(item => nodeText(item).includes('Application Discovery'));
        expect(panel).toBeTruthy();

        act(() => {
            panel!.props.onClick();
        });
        expect(goto).toHaveBeenCalledWith('./application-discovery');
    });
});
