export namespace domain {
	
	export class Coordinate {
	    lat: number;
	    lon: number;
	    alt: number;
	
	    static createFrom(source: any = {}) {
	        return new Coordinate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lat = source["lat"];
	        this.lon = source["lon"];
	        this.alt = source["alt"];
	    }
	}
	export class DeployedService {
	    instanceId: string;
	    serviceId: string;
	    folderName: string;
	    serviceTitle: string;
	    config: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new DeployedService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.instanceId = source["instanceId"];
	        this.serviceId = source["serviceId"];
	        this.folderName = source["folderName"];
	        this.serviceTitle = source["serviceTitle"];
	        this.config = source["config"];
	    }
	}
	export class GeneralConfig {
	    speedProfilePath: string;
	    loggingEnabled: boolean;
	    batteryRestricted: boolean;
	    batteryCapacity: number;
	    verboseLogging: boolean;
	    storeLocalData: boolean;
	    windEnabled: boolean;
	    windDirection: number;
	    windSpeed: number;
	    simulationName: string;
	    originalSimulationName: string;
	    groundFormation: string;
	    formationCenterLat: number;
	    formationCenterLon: number;
	    formationSpacing: number;
	    formationCenterMode: string;
	    swarmHost: string;
	    netsimInstances: number;
	
	    static createFrom(source: any = {}) {
	        return new GeneralConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.speedProfilePath = source["speedProfilePath"];
	        this.loggingEnabled = source["loggingEnabled"];
	        this.batteryRestricted = source["batteryRestricted"];
	        this.batteryCapacity = source["batteryCapacity"];
	        this.verboseLogging = source["verboseLogging"];
	        this.storeLocalData = source["storeLocalData"];
	        this.windEnabled = source["windEnabled"];
	        this.windDirection = source["windDirection"];
	        this.windSpeed = source["windSpeed"];
	        this.simulationName = source["simulationName"];
	        this.originalSimulationName = source["originalSimulationName"];
	        this.groundFormation = source["groundFormation"];
	        this.formationCenterLat = source["formationCenterLat"];
	        this.formationCenterLon = source["formationCenterLon"];
	        this.formationSpacing = source["formationSpacing"];
	        this.formationCenterMode = source["formationCenterMode"];
	        this.swarmHost = source["swarmHost"];
	        this.netsimInstances = source["netsimInstances"];
	    }
	}
	export class ServiceType {
	    id: string;
	    folderName: string;
	    title: string;
	    schemaRaw: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.folderName = source["folderName"];
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
	export class SimulationState {
	    uavs: UAV[];
	    generalConfig: GeneralConfig;
	    activeMode: string;
	
	    static createFrom(source: any = {}) {
	        return new SimulationState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uavs = this.convertValues(source["uavs"], UAV);
	        this.generalConfig = this.convertValues(source["generalConfig"], GeneralConfig);
	        this.activeMode = source["activeMode"];
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
	export class LogMessage {
	    InstanceID: string;
	    ServiceID: string;
	    Level: string;
	    Timestamp: string;
	    Message: string;
	    EventID: string;
	
	    static createFrom(source: any = {}) {
	        return new LogMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.InstanceID = source["InstanceID"];
	        this.ServiceID = source["ServiceID"];
	        this.Level = source["Level"];
	        this.Timestamp = source["Timestamp"];
	        this.Message = source["Message"];
	        this.EventID = source["EventID"];
	    }
	}
	export class LogFilter {
	    InstanceID: string;
	    ServiceID: string;
	    Level: string;
	    EventID: string;
	    SearchText: string;
	
	    static createFrom(source: any = {}) {
	        return new LogFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.InstanceID = source["InstanceID"] || "";
	        this.ServiceID = source["ServiceID"] || "";
	        this.Level = source["Level"] || "";
	        this.EventID = source["EventID"] || "";
	        this.SearchText = source["SearchText"] || "";
	    }
	}
}

