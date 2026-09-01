export const unicodeCharacterCount = (value: string) => Array.from(value).length;

export const hasControlCharacters = (value: string) => /\p{Cc}/u.test(value);
