module.exports = {
  preset: 'ts-jest',
  testEnvironment: '<rootDir>/jest-environment.cjs',
  reporters: ['default', 'jest-junit'],
  collectCoverage: true,
  testMatch: [
    '<rootDir>/src/app/**/*.test.tsx',
    '<rootDir>/src/app/mobile/**/*.test.tsx',
    '<rootDir>/src/app/shared/**/*.test.ts'
  ],
  transform: {
    '^.+\\.tsx?$': ['ts-jest', {
      isolatedModules: true,
      tsconfig: {moduleResolution: 'node'},
    }]
  },
  globals: {
    'self': {}
  },
  moduleNameMapper: {
    // https://github.com/facebook/jest/issues/3094
    '\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$': '<rootDir>/__mocks__/fileMock.js',
    '.+\\.(css|styl|less|sass|scss)$': '<rootDir>/__mocks__/fileMock.js',
  },
};
