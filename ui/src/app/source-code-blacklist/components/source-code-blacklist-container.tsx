import * as React from 'react';
import {Route, RouteComponentProps, Switch} from 'react-router';

import {SourceCodeBlacklistList} from './source-code-blacklist-list/source-code-blacklist-list';

export const SourceCodeBlacklistContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Route exact={true} path={`${props.match.path}`} component={SourceCodeBlacklistList} />
    </Switch>
);
