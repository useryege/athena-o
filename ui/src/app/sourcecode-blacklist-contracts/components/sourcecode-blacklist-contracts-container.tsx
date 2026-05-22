import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';

import {SourcecodeBlacklistContractsList} from './sourcecode-blacklist-contracts-list/sourcecode-blacklist-contracts-list';

export const SourcecodeBlacklistContractsContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={SourcecodeBlacklistContractsList} />
    </Switch>
);
