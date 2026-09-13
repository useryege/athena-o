import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {dirname, resolve} from 'node:path';
import test from 'node:test';
import {ESLint} from 'eslint';

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const uiRoot = resolve(scriptDirectory, '..');
const eslint = new ESLint({cwd: uiRoot, overrideConfigFile: resolve(uiRoot, 'eslint.config.mjs')});

const restrictedCases = [
    {
        name: 'member TypeScript files cannot import admin modules one directory up',
        filePath: 'src/app/member/screen.ts',
        specifier: '../admin/feature',
        message: 'The member application must not import administrator modules.'
    },
    {
        name: 'nested member TSX files cannot import admin modules three directories up',
        filePath: 'src/app/member/nested/deep/screen.tsx',
        specifier: '../../../admin/feature',
        message: 'The member application must not import administrator modules.'
    },
    {
        name: 'admin TSX files cannot import member modules one directory up',
        filePath: 'src/app/admin/screen.tsx',
        specifier: '../member/feature',
        message: 'The administrator application must not import member modules.'
    },
    {
        name: 'nested admin TypeScript files cannot import member modules two directories up',
        filePath: 'src/app/admin/nested/screen.ts',
        specifier: '../../member/feature',
        message: 'The administrator application must not import member modules.'
    },
    {
        name: 'shared TypeScript files cannot import member modules two directories up',
        filePath: 'src/app/shared/nested/screen.ts',
        specifier: '../../member/feature',
        message: 'Shared and session modules must not import either application realm.'
    },
    {
        name: 'nested shared TSX files cannot import admin modules three directories up',
        filePath: 'src/app/shared/nested/deep/screen.tsx',
        specifier: '../../../admin/feature',
        message: 'Shared and session modules must not import either application realm.'
    },
    {
        name: 'session TSX files cannot import member modules one directory up',
        filePath: 'src/app/session/screen.tsx',
        specifier: '../member/feature',
        message: 'Shared and session modules must not import either application realm.'
    },
    {
        name: 'nested session TypeScript files cannot import admin modules two directories up',
        filePath: 'src/app/session/nested/screen.ts',
        specifier: '../../admin/feature',
        message: 'Shared and session modules must not import either application realm.'
    },
    {
        name: 'components TypeScript files cannot import member modules one directory up',
        filePath: 'src/app/components/screen.ts',
        specifier: '../member/feature',
        message: 'Shared and session modules must not import either application realm.'
    },
    {
        name: 'nested component TSX files cannot import admin modules three directories up',
        filePath: 'src/app/components/nested/deep/screen.tsx',
        specifier: '../../../admin/feature',
        message: 'Shared and session modules must not import either application realm.'
    }
];

async function lint(relativeFilePath, source) {
    const [result] = await eslint.lintText(source, {filePath: resolve(uiRoot, relativeFilePath)});
    assert.equal(result.fatalErrorCount, 0, `${relativeFilePath} must parse without fatal errors`);
    return result;
}

for (const restrictedCase of restrictedCases) {
    test(restrictedCase.name, async () => {
        const result = await lint(
            restrictedCase.filePath,
            `import importedModule from '${restrictedCase.specifier}';\nvoid importedModule;\n`
        );
        const restrictedMessages = result.messages
            .filter(({ruleId}) => ruleId === 'no-restricted-imports')
            .map(({ruleId, message, messageId, nodeType, line, column, endLine, endColumn, severity}) => ({
                ruleId,
                message,
                messageId,
                nodeType,
                line,
                column,
                endLine,
                endColumn,
                severity
            }));

        assert.deepEqual(restrictedMessages, [
            {
                ruleId: 'no-restricted-imports',
                message: `'${restrictedCase.specifier}' import is restricted from being used by a pattern. ${restrictedCase.message}`,
                messageId: 'patternWithCustomMessage',
                nodeType: 'ImportDeclaration',
                line: 1,
                column: 1,
                endLine: 1,
                endColumn: restrictedCase.specifier.length + 31,
                severity: 2
            }
        ]);
    });
}

test('same-realm, shared-module, and third-party imports remain allowed', async () => {
    const allowedCases = [
        ['src/app/member/nested/screen.ts', '../feature'],
        ['src/app/admin/nested/screen.tsx', '../feature'],
        ['src/app/member/nested/screen.ts', '../../shared/format'],
        ['src/app/components/screen.tsx', 'react']
    ];

    for (const [filePath, specifier] of allowedCases) {
        const result = await lint(filePath, `import importedModule from '${specifier}';\nvoid importedModule;\n`);
        assert.deepEqual(
            result.messages.filter(({ruleId}) => ruleId === 'no-restricted-imports'),
            [],
            `${filePath} should allow ${specifier}`
        );
    }
});

test('existing TypeScript test files remain ignored', async () => {
    assert.equal(await eslint.isPathIgnored(resolve(uiRoot, 'src/app/member/screen.test.ts')), true);
});
