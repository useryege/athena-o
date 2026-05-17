import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';

import {BytecodeBlacklistList} from './bytecode-blacklist-list/bytecode-blacklist-list';

export const BytecodeBlacklistContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={BytecodeBlacklistList} />
    </Switch>
);
