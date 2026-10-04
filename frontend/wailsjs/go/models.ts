export namespace main {
	
	export class AppConfig {
	    lastFolder: string;
	    lastSong: string;
	    playMode: string;
	    volume: number;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lastFolder = source["lastFolder"];
	        this.lastSong = source["lastSong"];
	        this.playMode = source["playMode"];
	        this.volume = source["volume"];
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

