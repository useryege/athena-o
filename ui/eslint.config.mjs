import globals from 'globals';
import pluginJs from '@eslint/js';
import tseslint from 'typescript-eslint';
import pluginReactConfig from 'eslint-plugin-react/configs/recommended.js';
import eslintPluginPrettierRecommended from 'eslint-plugin-prettier/recommended';

export default [
    {languageOptions: {globals: globals.browser}},
    pluginJs.configs.recommended,
    ...tseslint.configs.recommended,
    {
        rules: {
            '@typescript-eslint/no-explicit-any': 'off',
            '@typescript-eslint/ban-types': 'off',
            '@typescript-eslint/no-var-requires': 'off'
        }
    },
    {
        settings: {
            react: {
                version: 'detect'
            }
        },
        ...pluginReactConfig,
        rules: {
            'react/display-name': 'off',
            'react/no-string-refs': 'off',
            'react/jsx-no-useless-fragment': ['error', {allowExpressions: true}]
        }
    },
    eslintPluginPrettierRecommended,
    {
        files: ['src/**/*.{ts,tsx}']
    },
    {
        files: ['src/app/member/**/*.{ts,tsx}'],
        rules: {
            'no-restricted-imports': [
                'error',
                {
                    patterns: [
                        {
                            group: ['../admin/**', '../../admin/**', '../../../admin/**'],
                            message: 'The member application must not import administrator modules.'
                        }
                    ]
                }
            ]
        }
    },
    {
        files: ['src/app/admin/**/*.{ts,tsx}'],
        rules: {
            'no-restricted-imports': [
                'error',
                {
                    patterns: [
                        {
                            group: ['../member/**', '../../member/**', '../../../member/**'],
                            message: 'The administrator application must not import member modules.'
                        }
                    ]
                }
            ]
        }
    },
    {
        files: ['src/app/shared/**/*.{ts,tsx}', 'src/app/session/**/*.{ts,tsx}', 'src/app/components/**/*.{ts,tsx}'],
        rules: {
            'no-restricted-imports': [
                'error',
                {
                    patterns: [
                        {
                            group: ['../member/**', '../../member/**', '../../../member/**', '../admin/**', '../../admin/**', '../../../admin/**'],
                            message: 'Shared and session modules must not import either application realm.'
                        }
                    ]
                }
            ]
        }
    },
    {
        ignores: ['dist', 'assets', '**/*.config.js', '__mocks__', 'coverage', 'playwright-report', 'test-results', '**/*.test.{ts,tsx}']
    }
];
