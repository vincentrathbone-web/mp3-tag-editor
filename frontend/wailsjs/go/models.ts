export namespace main {
	
	export class CoverSource {
	    format: string;
	    width: number;
	    height: number;
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new CoverSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.bytes = source["bytes"];
	    }
	}
	export class Cover {
	    dataUrl?: string;
	    thumb: string;
	    mime: string;
	    width: number;
	    height: number;
	    bytes: number;
	    source?: CoverSource;
	
	    static createFrom(source: any = {}) {
	        return new Cover(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataUrl = source["dataUrl"];
	        this.thumb = source["thumb"];
	        this.mime = source["mime"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.bytes = source["bytes"];
	        this.source = this.convertValues(source["source"], CoverSource);
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
	
	export class SaveRequest {
	    path: string;
	    tags: Record<string, string>;
	    cover: string;
	    coverData: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.tags = source["tags"];
	        this.cover = source["cover"];
	        this.coverData = source["coverData"];
	    }
	}
	export class Track {
	    path: string;
	    name: string;
	    ext: string;
	    folder: string;
	    size: number;
	    modified: string;
	    tagVersion: string;
	    duration: number;
	    bitrate: number;
	    vbr: boolean;
	    sampleRate: number;
	    channels: string;
	    codec: string;
	    tags: Record<string, string>;
	    cover?: Cover;
	    problem: string;
	
	    static createFrom(source: any = {}) {
	        return new Track(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.ext = source["ext"];
	        this.folder = source["folder"];
	        this.size = source["size"];
	        this.modified = source["modified"];
	        this.tagVersion = source["tagVersion"];
	        this.duration = source["duration"];
	        this.bitrate = source["bitrate"];
	        this.vbr = source["vbr"];
	        this.sampleRate = source["sampleRate"];
	        this.channels = source["channels"];
	        this.codec = source["codec"];
	        this.tags = source["tags"];
	        this.cover = this.convertValues(source["cover"], Cover);
	        this.problem = source["problem"];
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

}

