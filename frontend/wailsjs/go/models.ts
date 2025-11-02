export namespace main {
	
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
	export class ExportRequest {
	    results: SearchResult[];
	    format: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], SearchResult);
	        this.format = source["format"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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

}

