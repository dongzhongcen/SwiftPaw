export namespace core {
	
	export class AppConfig {
	    lastFolder: string;
	    lastSong: string;
	    playMode: string;
	    volume: number;
	    theme: string;
	    style: string;
	    reduceBlur: string;
	    background: BackgroundConfig;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lastFolder = source["lastFolder"];
	        this.lastSong = source["lastSong"];
	        this.playMode = source["playMode"];
	        this.volume = source["volume"];
	        this.theme = source["theme"];
	        this.style = source["style"];
	        this.reduceBlur = source["reduceBlur"];
	        this.background = this.convertValues(source["background"], BackgroundConfig);
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
		                a[key] = this.convertValues(a[key], classs);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackgroundConfig {
	    image: string;
	    blur: number;
	    dim: number;
	
	    static createFrom(source: any = {}) {
	        return new BackgroundConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.blur = source["blur"];
	        this.dim = source["dim"];
	    }
	}

}

export namespace lyrics {
	
	export class Line {
	    time: number;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new Line(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.text = source["text"];
	    }
	}
	export class Lyrics {
	    lines: Line[];
	    plain: boolean;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new Lyrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = this.convertValues(source["lines"], Line);
	        this.plain = source["plain"];
	        this.source = source["source"];
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
		                a[key] = this.convertValues(a[key], classs);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace music {
	
	export class Song {
	    name: string;
	    path: string;
	    title: string;
	    artist: string;
	    album: string;
	    format: string;
	    source: string;
	    id: string;
	    artwork: string;
	    duration: number;
	    extra?: any;
	
	    static createFrom(source: any = {}) {
	        return new Song(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.format = source["format"];
	        this.source = source["source"];
	        this.id = source["id"];
	        this.artwork = source["artwork"];
	        this.duration = source["duration"];
	        this.extra = source["extra"];
	    }
	}

}

export namespace plugin {
	
	export class Info {
	    id: string;
	    platform: string;
	    version: string;
	    author: string;
	    srcUrl: string;
	    supportedSearchType: string[];
	    enabled: boolean;
	    canSearch: boolean;
	    canPlay: boolean;
	    canLyric: boolean;
	    error: string;
	    missing: string[];
	    userVariables: UserVariable[];
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.srcUrl = source["srcUrl"];
	        this.supportedSearchType = source["supportedSearchType"];
	        this.enabled = source["enabled"];
	        this.canSearch = source["canSearch"];
	        this.canPlay = source["canPlay"];
	        this.canLyric = source["canLyric"];
	        this.error = source["error"];
	        this.missing = source["missing"];
	        this.userVariables = this.convertValues(source["userVariables"], UserVariable);
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
		                a[key] = this.convertValues(a[key], classs);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchResult {
	    isEnd: boolean;
	    data: music.Song[];
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnd = source["isEnd"];
	        this.data = this.convertValues(source["data"], music.Song);
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
		                a[key] = this.convertValues(a[key], classs);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UserVariable {
	    key: string;
	    name: string;
	    hint: string;
	
	    static createFrom(source: any = {}) {
	        return new UserVariable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	        this.hint = source["hint"];
	    }
	}

}

export namespace queue {
	
	export class State {
	    songs: music.Song[];
	    current: number;
	    mode: string;
	    upcoming: number[];
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.songs = this.convertValues(source["songs"], music.Song);
	        this.current = source["current"];
	        this.mode = source["mode"];
	        this.upcoming = source["upcoming"];
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
		                a[key] = this.convertValues(a[key], classs);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace store {
	
	export class Playlist {
	    id: number;
	    name: string;
	    count: number;
	    builtin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Playlist(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.count = source["count"];
	        this.builtin = source["builtin"];
	    }
	}

}

