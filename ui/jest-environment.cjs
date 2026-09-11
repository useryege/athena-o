const {TestEnvironment} = require('jest-environment-jsdom');
// Keep Fetch's request and cancellation objects in the same Node realm.
const {Request, Response, Headers, AbortController, AbortSignal} = globalThis;

module.exports = class AthenaTestEnvironment extends TestEnvironment {
    async setup() {
        await super.setup();
        Object.assign(this.global, {Request, Response, Headers, AbortController, AbortSignal});
    }
};
