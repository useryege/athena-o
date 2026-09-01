import {VersionMessage} from '../models';
import requests from './requests';

export class VersionService {
    public version(): Promise<VersionMessage> {
        return requests.rawGet('/api/version', {session: true}).then(res => res.body as VersionMessage);
    }
}
