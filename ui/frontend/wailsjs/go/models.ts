export namespace simulation {
	
	export class DeployedService {
	    instanceId: string;
	    serviceId: string;
	    serviceTitle: string;
	    config: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new DeployedService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.instanceId = source["instanceId"];
	        this.serviceId = source["serviceId"];
	        this.serviceTitle = source["serviceTitle"];
	        this.config = source["config"];
	    }
	}
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
	export class UAV {
	    id: string;
	    services: DeployedService[];
	
	    static createFrom(source: any = {}) {
	        return new UAV(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.services = this.convertValues(source["services"], DeployedService);
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

