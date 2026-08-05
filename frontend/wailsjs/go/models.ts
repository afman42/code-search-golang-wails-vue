export namespace main {
	
	export class EditorAvailability {
	    vscode: boolean;
	    vscodium: boolean;
	    sublime: boolean;
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
	
	    static createFrom(source: any = {}) {
	        return new EditorAvailability(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vscode = source["vscode"];
	        this.vscodium = source["vscodium"];
	        this.sublime = source["sublime"];
	        this.jetbrains = source["jetbrains"];
	        this.geany = source["geany"];
	        this.neovim = source["neovim"];
	        this.vim = source["vim"];
	        this.goland = source["goland"];
	        this.pycharm = source["pycharm"];
	        this.intellij = source["intellij"];
	        this.webstorm = source["webstorm"];
	        this.phpstorm = source["phpstorm"];
	        this.clion = source["clion"];
	        this.rider = source["rider"];
	        this.androidstudio = source["androidstudio"];
	        this.systemdefault = source["systemdefault"];
	        this.emacs = source["emacs"];
	        this.neovide = source["neovide"];
	        this.codeblocks = source["codeblocks"];
	        this.devcpp = source["devcpp"];
	        this.notepadplusplus = source["notepadplusplus"];
	        this.visualstudio = source["visualstudio"];
	        this.eclipse = source["eclipse"];
	        this.netbeans = source["netbeans"];
	    }
	}
	export class LogMessage {
	    type: string;
	    content: any;
	
	    static createFrom(source: any = {}) {
	        return new LogMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.content = source["content"];
	    }
	}
	export class SearchRequest {
	    directory: string;
	    query: string;
	    extension: string;
	    caseSensitive: boolean;
	    includeBinary: boolean;
	    maxFileSize: number;
	    minFileSize: number;
	    maxResults: number;
	    searchSubdirs: boolean;
	    useRegex?: boolean;
	    excludePatterns: string[];
	    allowedFileTypes: string[];
	    contextLines: number;
	    directories: string[];
	    fuzzySearch: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.directory = source["directory"];
	        this.query = source["query"];
	        this.extension = source["extension"];
	        this.caseSensitive = source["caseSensitive"];
	        this.includeBinary = source["includeBinary"];
	        this.maxFileSize = source["maxFileSize"];
	        this.minFileSize = source["minFileSize"];
	        this.maxResults = source["maxResults"];
	        this.searchSubdirs = source["searchSubdirs"];
	        this.useRegex = source["useRegex"];
	        this.excludePatterns = source["excludePatterns"];
	        this.allowedFileTypes = source["allowedFileTypes"];
	        this.contextLines = source["contextLines"];
	        this.directories = source["directories"];
	        this.fuzzySearch = source["fuzzySearch"];
	    }
	}
	export class SearchResult {
	    filePath: string;
	    lineNum: number;
	    content: string;
	    matchedText: string;
	    contextBefore: string[];
	    contextAfter: string[];
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.lineNum = source["lineNum"];
	        this.content = source["content"];
	        this.matchedText = source["matchedText"];
	        this.contextBefore = source["contextBefore"];
	        this.contextAfter = source["contextAfter"];
	    }
	}
	export class SymbolInfo {
	    name: string;
	    type: string;
	    line: number;
	    file: string;
	    signature: string;
	
	    static createFrom(source: any = {}) {
	        return new SymbolInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.line = source["line"];
	        this.file = source["file"];
	        this.signature = source["signature"];
	    }
	}

}

