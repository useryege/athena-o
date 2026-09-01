export type ApplicationRealm = 'member' | 'admin';

let configuredRealm: ApplicationRealm | undefined;
let configuredServices: Record<PropertyKey, unknown> = {};

export const configureServices = <T extends object>(realm: ApplicationRealm, next: Partial<T>) => {
    if (configuredRealm && configuredRealm !== realm) {
        throw new Error(`Athena services are already configured for the ${configuredRealm} realm`);
    }
    configuredRealm = realm;
    configuredServices = {...configuredServices, ...next};
};

const serviceRegistry = new Proxy({} as Record<PropertyKey, unknown>, {
    get(_target, property: PropertyKey) {
        const service = configuredServices[property];
        if (!service) {
            throw new Error(`Service ${String(property)} is not available in the ${configuredRealm || 'unconfigured'} realm`);
        }
        return service;
    }
});

export const serviceProjection = <T extends object>() => serviceRegistry as T;
