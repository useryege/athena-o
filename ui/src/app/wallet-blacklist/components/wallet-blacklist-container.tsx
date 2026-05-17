import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';

import {WalletBlacklistList} from './wallet-blacklist-list/wallet-blacklist-list';

export const WalletBlacklistContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={WalletBlacklistList} />
    </Switch>
);
