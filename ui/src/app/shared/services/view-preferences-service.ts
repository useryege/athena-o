import deepMerge from 'deepmerge';
import {BehaviorSubject, Observable} from 'rxjs';

export interface ViewPreferences {
    version: number;
    pageSizes: {[key: string]: number};
    sortOptions?: {[key: string]: string};
    hideBannerContent: string;
    hideSidebar: boolean;
    position: string;
}

const minVer = 6;

const DEFAULT_PREFERENCES: ViewPreferences = {
    version: minVer,
    pageSizes: {},
    hideBannerContent: '',
    hideSidebar: false,
    position: ''
};

export class ViewPreferencesService {
    private preferencesSubj: BehaviorSubject<ViewPreferences>;

    constructor(private readonly storageKey = 'athena.member.preferences') {}

    public init() {
        if (!this.preferencesSubj) {
            const preferences = this.loadPreferences();
            this.preferencesSubj = new BehaviorSubject(preferences);
            window.addEventListener('storage', event => {
                if (event.key !== null && event.key !== this.storageKey) {
                    return;
                }
                const nextPreferences = this.loadPreferences();
                this.preferencesSubj.next(nextPreferences);
            });
        }
    }

    public getPreferences(): Observable<ViewPreferences> {
        return this.preferencesSubj;
    }

    public updatePreferences(change: Partial<ViewPreferences>) {
        const nextPref = this.supportedPreferences({...this.preferencesSubj.getValue(), ...change, version: minVer});
        window.localStorage.setItem(this.storageKey, JSON.stringify(nextPref));
        this.preferencesSubj.next(nextPref);
    }

    private supportedPreferences(value: Partial<ViewPreferences>): ViewPreferences {
        const supported: Partial<ViewPreferences> = {};
        for (const key of ['version', 'pageSizes', 'sortOptions', 'hideBannerContent', 'hideSidebar', 'position'] as const) {
            if (value[key] !== undefined) {
                Object.assign(supported, {[key]: value[key]});
            }
        }
        return deepMerge(DEFAULT_PREFERENCES, supported);
    }

    private loadPreferences(): ViewPreferences {
        let preferences: ViewPreferences;
        const preferencesStr = window.localStorage.getItem(this.storageKey);
        if (preferencesStr) {
            try {
                const parsed = JSON.parse(preferencesStr);
                preferences = parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : DEFAULT_PREFERENCES;
            } catch (e) {
                preferences = DEFAULT_PREFERENCES;
            }
            if (!preferences.version || preferences.version < minVer) {
                preferences = DEFAULT_PREFERENCES;
            }
        } else {
            preferences = DEFAULT_PREFERENCES;
        }
        return this.supportedPreferences(preferences);
    }
}
