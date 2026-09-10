import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {AccountAccessEditor} from './admin-accounts';
import {ChoiceGroup} from '../../components';
import {parseAccountAccess, parseAccountIdentity, parseAccountProfile} from '../../shared/models';

test('real editor exposes only NONE and RW for Trader Sync and edits full matrix', () => {
    window.matchMedia = window.matchMedia || jest.fn().mockReturnValue({matches: false, addListener: () => {}, removeListener: () => {}});
    const access = parseAccountAccess({moduleAccess: [{module: 'trader_sync', dataAccess: 'read_write'}]});
    let edited: any;
    let tree: renderer.ReactTestRenderer;
    act(() => {tree = renderer.create(<AccountAccessEditor account={{username: 'member', identity: parseAccountIdentity({}), profile: parseAccountProfile({}, 'member')} as any} access={access} editable={true} dirty={false} updating={false} onChange={value => {edited = value;}} onSave={() => {}} onReset={() => {}} />);});
    const choice = tree!.root.findAllByType(ChoiceGroup).find(item => item.props.ariaLabel === 'Trader Sync data access for @member');
    expect(choice).toBeDefined();
    expect(choice!.props.options.map((item: any) => item.value)).toEqual([0, 2]);
    act(() => choice!.props.onChange(0));
    expect(edited.moduleAccess).toHaveLength(10);
    expect(edited.moduleAccess.find((item: any) => item.module === 12).dataAccess).toBe(0);
    act(() => tree!.unmount());
});
