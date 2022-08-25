//~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// This file is autogenerated. DO NOT EDIT.
//
// To regenerate, run "make gen-cue" from the repository root.
//~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~


export const PanelModelVersion = Object.freeze([0, 0]);


export enum TextMode {
  Code = 'code',
  HTML = 'html',
  Markdown = 'markdown',
}

export enum CodeLanguage {
  Go = 'go',
  Html = 'html',
  Json = 'json',
  Markdown = 'markdown',
  Plaintext = 'plaintext',
  Sql = 'sql',
  Typescript = 'typescript',
  Xml = 'xml',
  Yaml = 'yaml',
}

export const defaultCodeLanguage: CodeLanguage = CodeLanguage.Plaintext;

export interface CodeOptions {
  language: CodeLanguage;
  showLineNumbers: boolean;
  showMiniMap: boolean;
}

export const defaultCodeOptions: Partial<CodeOptions> = {
  language: CodeLanguage.Plaintext,
  showLineNumbers: false,
  showMiniMap: false,
};

export interface PanelOptions {
  code?: CodeOptions;
  content: string;
  mode: TextMode;
}

export const defaultPanelOptions: Partial<PanelOptions> = {
  content: `# Title

For markdown syntax help: [commonmark.org/help](https://commonmark.org/help/)`,
  mode: TextMode.Markdown,
};
