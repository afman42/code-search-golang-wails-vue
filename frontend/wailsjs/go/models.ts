export namespace main {
	
	export class EditorAvailability {
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
	
	    static createFrom(source: any = {}) {
	        return new EditorAvailability(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vscode = source["vscode"];
	        this.vscodium = source["vscodium"];
	        this.sublime = source["sublime"];
	        this.atom = source["atom"];
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

}

