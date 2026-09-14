import renderer, {act} from 'react-test-renderer';
import {Button, Input} from 'antd';
import {MemberApp} from '../app';
import {parseUserInfo, AppBootstrapSessionStatus} from '../../shared/models';
import {ensureMemberBusinessServices, memberServices as services} from '../services';
import cases from '../../../../e2e/theme-refactor/fixtures/worm-assets-combinations.json';

const fixture = cases.find(item => item.id === 'worm-combinations-edit')!;
const replies = fixture.replies as any[];
const combination = replies.find(item => item.path.includes('/combinations/')).json;
const event = replies.find(item => item.path.includes('/events/')).json;
const bootstrap = replies[0].json;
let tree: renderer.ReactTestRenderer;
const identity = (iss: string) => parseUserInfo({...bootstrap.session.userInfo, iss});
beforeEach(() => {
    ensureMemberBusinessServices();
    localStorage.clear();
    sessionStorage.clear();
    window.matchMedia = jest
        .fn()
        .mockImplementation(query => ({
            matches: false,
            media: query,
            addListener: jest.fn(),
            removeListener: jest.fn(),
            addEventListener: jest.fn(),
            removeEventListener: jest.fn()
        }));
    jest.spyOn(services.authService, 'bootstrap').mockResolvedValue({
        settings: bootstrap.settings,
        session: {status: AppBootstrapSessionStatus.Authenticated, userInfo: identity('old-issuer')}
    });
    jest.spyOn(services.users, 'get').mockResolvedValue(identity('old-issuer'));
    jest.spyOn(services.wormTrading, 'getMarketCombination').mockResolvedValue(combination);
    jest.spyOn(services.wormTrading, 'getEvent').mockResolvedValue(event);
});
afterEach(() => {
    if (tree) act(() => tree.unmount());
    jest.restoreAllMocks();
});
const mount = async () => {
    window.history.replaceState(null, '', fixture.route);
    await act(async () => {
        tree = renderer.create(<MemberApp />);
    });
};
const nameInput = () => tree.root.findAllByType(Input).find(item => item.props.value === 'September market basket' || item.props.value === 'Private previous draft')!;
test('Worm editor discards the previous issuer draft even when account and access revision are unchanged', async () => {
    await mount();
    await act(async () => nameInput().props.onChange({target: {value: 'Private previous draft'}}));
    jest.mocked(services.users.get).mockResolvedValue(identity('new-issuer'));
    await act(async () => window.dispatchEvent(new Event('focus')));
    expect(nameInput().props.value).toBe('September market basket');
    expect(JSON.stringify(tree.toJSON())).not.toContain('Private previous draft');
});
test('late Worm save cannot navigate or notify after an issuer switch', async () => {
    await mount();
    let resolve!: (value: any) => void;
    jest.spyOn(services.wormTrading, 'updateMarketCombination').mockReturnValue(new Promise<any>(done => (resolve = done)));
    await act(async () => nameInput().props.onChange({target: {value: 'Private previous draft'}}));
    await act(async () => {
        tree.root
            .findAllByType(Button)
            .find(item => item.props['aria-label'] === 'Save changes')!
            .props.onClick();
    });
    jest.mocked(services.users.get).mockResolvedValue(identity('new-issuer'));
    await act(async () => window.dispatchEvent(new Event('focus')));
    await act(async () => resolve({...combination, name: 'Late previous saved result', revision: 8}));
    expect(window.location.pathname).toBe(fixture.route);
    expect(JSON.stringify(tree.toJSON())).not.toContain('Late previous saved result');
});
