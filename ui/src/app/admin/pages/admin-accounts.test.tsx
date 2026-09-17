import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {AccountAccessEditor} from './admin-accounts';
import {Select} from 'antd';
import {parseAccountAccess, parseAccountIdentity, parseAccountProfile} from '../../shared/models';

test('real editor exposes only NONE and RW for Trader Sync and edits full matrix', () => {
    window.matchMedia = window.matchMedia || jest.fn().mockReturnValue({matches: false, addListener: () => {}, removeListener: () => {}});
    const access = parseAccountAccess({
        moduleAccess: [
            {module: 'trader_sync', dataAccess: 'read_write'},
            {module: 'solana', dataAccess: 'read'},
            {module: 'token', dataAccess: 'read'}
        ]
    });
    let edited: any;
    let tree: renderer.ReactTestRenderer;
    act(() => {
        tree = renderer.create(
            <AccountAccessEditor
                account={{username: 'member', identity: parseAccountIdentity({}), profile: parseAccountProfile({}, 'member')} as any}
                access={access}
                editable={true}
                dirty={false}
                updating={false}
                onChange={value => {
                    edited = value;
                }}
                onSave={() => {}}
                onReset={() => {}}
            />
        );
    });
    const choice = tree!.root.findAllByType(Select).find(item => item.props['aria-label'] === 'Trader Sync data access for @member');
    expect(choice).toBeDefined();
    expect(choice!.props.options.map((item: any) => item.value)).toEqual([0, 2]);
    const solanaChoice = tree!.root.findAllByType(Select).find(item => item.props['aria-label'] === 'Solana data access for @member');
    expect(solanaChoice!.props.options.map((item: any) => item.value)).toEqual([0, 1]);
    act(() => choice!.props.onChange(0));
    expect(edited.moduleAccess.map((item: any) => item.module).sort((a: number, b: number) => a - b)).toEqual([1, 4, 8, 9, 11, 12, 13]);
    expect(edited.moduleAccess.find((item: any) => item.module === 12).dataAccess).toBe(0);
    expect(edited.moduleAccess.find((item: any) => item.module === 13).dataAccess).toBe(1);
    expect(edited.moduleAccess.find((item: any) => item.module === 8).dataAccess).toBe(1);
    act(() => tree!.unmount());
});
