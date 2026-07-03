import deepMerge from 'deepmerge';
import {BehaviorSubject, Observable} from 'rxjs';

export type ThemeMode = 'dark' | 'light';

export interface ViewPreferences {
    version: number;
    pageSizes: {[key: string]: number};
    sortOptions?: {[key: string]: string};
    hideBannerContent: string;
    hideSidebar: boolean;
    position: string;
    theme: ThemeMode;
}

const VIEW_PREFERENCES_KEY = 'view_preferences';

const minVer = 6;

const DEFAULT_PREFERENCES: ViewPreferences = {
    version: minVer,
    pageSizes: {},
    hideBannerContent: '',
    hideSidebar: false,
    position: '',
    theme: 'light'
};

export class ViewPreferencesService {
    private preferencesSubj: BehaviorSubject<ViewPreferences>;

    public init() {
        if (!this.preferencesSubj) {
            const preferences = this.loadPreferences();
            this.applyTheme(preferences.theme);
            this.preferencesSubj = new BehaviorSubject(preferences);
            window.addEventListener('storage', event => {
                if (event.key !== null && event.key !== VIEW_PREFERENCES_KEY) {
                    return;
                }
                const nextPreferences = this.loadPreferences();
                this.applyTheme(nextPreferences.theme);
                this.preferencesSubj.next(nextPreferences);
            });
        }
    }

    public getPreferences(): Observable<ViewPreferences> {
        return this.preferencesSubj;
    }

    public updatePreferences(change: Partial<ViewPreferences>) {
        const nextPref = Object.assign({}, this.preferencesSubj.getValue(), change, {version: minVer});
        window.localStorage.setItem(VIEW_PREFERENCES_KEY, JSON.stringify(nextPref));
        this.applyTheme(nextPref.theme);
        this.preferencesSubj.next(nextPref);
    }

    private applyTheme(theme: ThemeMode) {
        document.documentElement.dataset.theme = theme;
    }

    private loadPreferences(): ViewPreferences {
        let preferences: ViewPreferences;
        const preferencesStr = window.localStorage.getItem(VIEW_PREFERENCES_KEY);
        if (preferencesStr) {
            try {
                preferences = JSON.parse(preferencesStr);
            } catch (e) {
                preferences = DEFAULT_PREFERENCES;
            }
            if (!preferences.version || preferences.version < minVer) {
                preferences = DEFAULT_PREFERENCES;
            }
        } else {
            preferences = DEFAULT_PREFERENCES;
        }
        return deepMerge(DEFAULT_PREFERENCES, preferences);
    }
}
