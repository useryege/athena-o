import {AthenaThemeProvider} from '../shared/athena-theme';
import * as React from 'react';
import {createRoot} from 'react-dom/client';
import {AdminApp} from '../admin/app';
import {readDeploymentBaseHRef} from '../shared/runtime-base';
import requests from '../shared/services/requests';

requests.setBaseHRef(readDeploymentBaseHRef());

createRoot(document.getElementById('app') as HTMLElement).render(
    <AthenaThemeProvider>
        <AdminApp />
    </AthenaThemeProvider>
);

(window as any).React = React;
