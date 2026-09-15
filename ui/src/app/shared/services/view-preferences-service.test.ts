import {ViewPreferencesService} from './view-preferences-service';

test('保存当前视图偏好时去除主题字段并保留分页与排序', () => {
    const key = 'athena.theme-refactor.test';
    localStorage.setItem(key, JSON.stringify({version: 6, pageSizes: {wallets: 50}, sortOptions: {wallets: 'name'}, theme: 'light'}));
    const service = new ViewPreferencesService(key);
    service.init();
    service.updatePreferences({hideSidebar: true});
    const saved = JSON.parse(localStorage.getItem(key)!);
    expect(saved.theme).toBeUndefined();
    expect(saved.pageSizes).toEqual({wallets: 50});
    expect(saved.sortOptions).toEqual({wallets: 'name'});
    expect(saved.hideSidebar).toBe(true);
    localStorage.removeItem(key);
});
