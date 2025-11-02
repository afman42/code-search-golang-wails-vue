export namespace main {
	
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

