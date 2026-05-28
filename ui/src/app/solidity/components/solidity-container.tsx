import * as React from 'react';
import {Redirect, Route, RouteComponentProps, Switch} from 'react-router';

import {BytecodeBlacklistContainer} from '../../bytecode-blacklist/components/bytecode-blacklist-container';
import {BytecodeDetail} from './bytecode-detail';
import {BytecodeList} from './bytecode-list';

require('./solidity.scss');

export const SolidityContainer = (props: RouteComponentProps<any>) => (
    <Switch>
        <Redirect exact={true} path={`${props.match.path}`} to={`${props.match.path}/bytecodes`} />
        <Route exact={true} path={`${props.match.path}/bytecodes`} component={BytecodeList} />
        <Route exact={true} path={`${props.match.path}/bytecodes/:codeHash`} component={BytecodeDetail} />
        <Route path={`${props.match.path}/bytecode-blacklist`} component={BytecodeBlacklistContainer} />
    </Switch>
);
