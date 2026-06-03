import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {useAsyncData} from './components';

const Probe = (props: {load: () => Promise<string> & {abort?: () => void}}) => {
    const state = useAsyncData(props.load, []);
    return <span>{state.loading ? 'loading' : state.error ? state.error.message : state.data}</span>;
};

test('useAsyncData renders loaded data', async () => {
    let tree: renderer.ReactTestRenderer;
    await act(async () => {
        tree = renderer.create(<Probe load={() => Promise.resolve('ready') as any} />);
    });
    expect(tree.toJSON()).toMatchObject({children: ['ready']});
});

test('useAsyncData renders errors', async () => {
    let tree: renderer.ReactTestRenderer;
    await act(async () => {
        tree = renderer.create(<Probe load={() => Promise.reject(new Error('nope')) as any} />);
    });
    expect(tree.toJSON()).toMatchObject({children: ['nope']});
});
