// Wails generated types declaration
declare module '../../wailsjs/go/main/App' {
  export function OpenInVSCode(filePath: string): Promise<void>;
  export function OpenInVSCodium(filePath: string): Promise<void>;
  export function OpenInSublime(filePath: string): Promise<void>;
  export function OpenInAtom(filePath: string): Promise<void>;
  export function OpenInJetBrains(filePath: string): Promise<void>;
  export function OpenInGeany(filePath: string): Promise<void>;
  export function OpenInGoland(filePath: string): Promise<void>;
  export function OpenInPyCharm(filePath: string): Promise<void>;
  export function OpenInIntelliJ(filePath: string): Promise<void>;
  export function OpenInWebStorm(filePath: string): Promise<void>;
  export function OpenInPhpStorm(filePath: string): Promise<void>;
  export function OpenInCLion(filePath: string): Promise<void>;
  export function OpenInRider(filePath: string): Promise<void>;
  export function OpenInAndroidStudio(filePath: string): Promise<void>;
  export function OpenInEmacs(filePath: string): Promise<void>;
  export function OpenInNeovide(filePath: string): Promise<void>;
  export function OpenInCodeBlocks(filePath: string): Promise<void>;
  export function OpenInDevCpp(filePath: string): Promise<void>;
  export function OpenInNotepadPlusPlus(filePath: string): Promise<void>;
  export function OpenInVisualStudio(filePath: string): Promise<void>;
  export function OpenInEclipse(filePath: string): Promise<void>;
  export function OpenInNetBeans(filePath: string): Promise<void>;
  export function OpenInDefaultEditor(filePath: string): Promise<void>;
  export function ShowInFolder(filePath: string): Promise<void>;
  export function ReadFile(filePath: string): Promise<string>;
  export function SearchWithProgress(searchRequest: any): Promise<any[]>;
  export function SelectDirectory(title: string): Promise<string>;
  export function ValidateDirectory(directory: string): Promise<boolean>;
  export function GetAvailableEditors(): Promise<any>;
  export function GetEditorDetectionStatus(): Promise<any>;
  export function CancelSearch(): Promise<void>;
}