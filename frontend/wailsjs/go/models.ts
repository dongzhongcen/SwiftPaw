export namespace main {
	
	export class AppConfig {
	    lastFolder: string;
	    lastSong: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lastFolder = source["lastFolder"];
	        this.lastSong = source["lastSong"];
	    }
	}
	export class Song {
	    name: string;
	    path: string;
	    title: string;
	    artist: string;
	
	    static createFrom(source: any = {}) {
	        return new Song(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	    }
	}

}

