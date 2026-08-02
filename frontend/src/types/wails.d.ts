declare module '../../wailsjs/go/main/App' {
  export function CancelSearch(): Promise<void>;
  export function ClearIndex(): Promise<void>;
  export function FindDefinitions(symbolName: string, maxResults: number): Promise<any[]>;
  export function FindUsages(symbolName: string, maxResults: number): Promise<any[]>;
  export function GetAllSymbols(maxResults: number): Promise<any[]>;
  export function GetAvailableEditors(): Promise<any>;
  export function GetDirectoryContents(directory: string): Promise<string[]>;
  export function GetEditorDetectionStatus(): Promise<any>;
  export function GetIndexStats(): Promise<{totalFiles:number,indexedFiles:number,totalSymbols:number,errors:number}>;
  export function GetInitialLogs(): Promise<any[]>;
  export function GetKnownTextExtensions(): Promise<string[]>;
  export function GetNewLogs(): Promise<any[]>;
  export function IndexDirectory(directory: string): Promise<{totalFiles:number,indexedFiles:number,totalSymbols:number,errors:number}>;
  export function IsAppReady(): Promise<boolean>;
  export function OpenInAndroidStudio(filePath: string): Promise<void>;
  export function OpenInAtom(filePath: string): Promise<void>;
  export function OpenInCLion(filePath: string): Promise<void>;
  export function OpenInCodeBlocks(filePath: string): Promise<void>;
  export function OpenInDefaultEditor(filePath: string): Promise<void>;
  export function OpenInDevCpp(filePath: string): Promise<void>;
  export function OpenInEclipse(filePath: string): Promise<void>;
  export function OpenInEmacs(filePath: string): Promise<void>;
  export function OpenInGeany(filePath: string): Promise<void>;
  export function OpenInGoland(filePath: string): Promise<void>;
  export function OpenInIntelliJ(filePath: string): Promise<void>;
  export function OpenInJetBrains(filePath: string): Promise<void>;
  export function OpenInNeovide(filePath: string): Promise<void>;
  export function OpenInNeovim(filePath: string): Promise<void>;
  export function OpenInNetBeans(filePath: string): Promise<void>;
  export function OpenInNotepadPlusPlus(filePath: string): Promise<void>;
  export function OpenInPhpStorm(filePath: string): Promise<void>;
  export function OpenInPyCharm(filePath: string): Promise<void>;
  export function OpenInRider(filePath: string): Promise<void>;
  export function OpenInSublime(filePath: string): Promise<void>;
  export function OpenInVSCode(filePath: string): Promise<void>;
  export function OpenInVSCodium(filePath: string): Promise<void>;
  export function OpenInVim(filePath: string): Promise<void>;
  export function OpenInVisualStudio(filePath: string): Promise<void>;
  export function OpenInWebStorm(filePath: string): Promise<void>;
  export function ReadFile(filePath: string): Promise<string>;
  export function ReadFileLog(filePath: string): Promise<string>;
  export function SearchSymbols(name: string, maxResults: number): Promise<any[]>;
  export function SearchWithProgress(request: any): Promise<any[]>;
  export function SelectDirectory(title: string): Promise<string>;
  export function ShowInFolder(filePath: string): Promise<void>;
  export function ValidateDirectory(directory: string): Promise<boolean>;
}

export interface SymbolInfo {
  name: string;
  type: string;
  line: number;
  endLine?: number;
  signature?: string;
  file: string;
}

export interface IndexStats {
  totalFiles: number;
  indexedFiles: number;
  totalSymbols: number;
  errors: number;
}

export interface EditorAvailability {
  vscode: boolean;
  vscodium: boolean;
  sublime: boolean;
  atom: boolean;
  jetbrains: boolean;
  geany: boolean;
  neovim: boolean;
  vim: boolean;
  goland: boolean;
  pycharm: boolean;
  intellij: boolean;
  webstorm: boolean;
  phpstorm: boolean;
  clion: boolean;
  rider: boolean;
  androidstudio: boolean;
  systemdefault: boolean;
  emacs: boolean;
  neovide: boolean;
  codeblocks: boolean;
  devcpp: boolean;
  notepadplusplus: boolean;
  visualstudio: boolean;
  eclipse: boolean;
  netbeans: boolean;
}

export interface LogMessage {
  timestamp: string;
  level: string;
  message: string;
}
