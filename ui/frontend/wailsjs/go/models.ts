export namespace main {
	
	export class ServiceType {
	    id: string;
	    title: string;
	    schemaRaw: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.schemaRaw = source["schemaRaw"];
	    }
	}

}

