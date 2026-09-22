export namespace activity {
	
	export class CurrentStatus {
	    app: string;
	    title: string;
	    detail: string;
	    domain: string;
	    category: string;
	    elapsedSeconds: number;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CurrentStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app = source["app"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.domain = source["domain"];
	        this.category = source["category"];
	        this.elapsedSeconds = source["elapsedSeconds"];
	        this.enabled = source["enabled"];
	    }
	}

}

export namespace ai {
	
	export class AskRequest {
	    prompt: string;
	    sessionID: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new AskRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prompt = source["prompt"];
	        this.sessionID = source["sessionID"];
	        this.model = source["model"];
	    }
	}
	export class AskResponse {
	    reply: string;
	    sessionID: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new AskResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reply = source["reply"];
	        this.sessionID = source["sessionID"];
	        this.model = source["model"];
	    }
	}
	export class Status {
	    available: boolean;
	    binary: string;
	    version: string;
	    authenticated: boolean;
	    direct: boolean;
	    hint: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.binary = source["binary"];
	        this.version = source["version"];
	        this.authenticated = source["authenticated"];
	        this.direct = source["direct"];
	        this.hint = source["hint"];
	    }
	}

}

export namespace drive {
	
	export class BackupFile {
	    id: string;
	    name: string;
	    createdTime: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new BackupFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.createdTime = source["createdTime"];
	        this.size = source["size"];
	    }
	}
	export class DeviceAuth {
	    userCode: string;
	    verificationUrl: string;
	    expiresIn: number;
	    interval: number;
	
	    static createFrom(source: any = {}) {
	        return new DeviceAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userCode = source["userCode"];
	        this.verificationUrl = source["verificationUrl"];
	        this.expiresIn = source["expiresIn"];
	        this.interval = source["interval"];
	    }
	}
	export class DevicePoll {
	    status: string;
	    email: string;
	
	    static createFrom(source: any = {}) {
	        return new DevicePoll(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.email = source["email"];
	    }
	}
	export class Status {
	    hasClient: boolean;
	    embedded: boolean;
	    connected: boolean;
	    email: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasClient = source["hasClient"];
	        this.embedded = source["embedded"];
	        this.connected = source["connected"];
	        this.email = source["email"];
	    }
	}

}

export namespace main {
	
	export class PrayerDayTime {
	    key: string;
	    time: string;
	    dateTime: string;
	
	    static createFrom(source: any = {}) {
	        return new PrayerDayTime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.time = source["time"];
	        this.dateTime = source["dateTime"];
	    }
	}
	export class PrayerSettingsInput {
	    enabled: boolean;
	    method: string;
	    asrHanafi: boolean;
	    city: string;
	    lat: number;
	    lng: number;
	    tz: string;
	    clock12h: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PrayerSettingsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.method = source["method"];
	        this.asrHanafi = source["asrHanafi"];
	        this.city = source["city"];
	        this.lat = source["lat"];
	        this.lng = source["lng"];
	        this.tz = source["tz"];
	        this.clock12h = source["clock12h"];
	    }
	}

}

export namespace prayer {
	
	export class City {
	    id: string;
	    name: string;
	    nameAr: string;
	    country: string;
	    lat: number;
	    lng: number;
	    tz: string;
	
	    static createFrom(source: any = {}) {
	        return new City(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.nameAr = source["nameAr"];
	        this.country = source["country"];
	        this.lat = source["lat"];
	        this.lng = source["lng"];
	        this.tz = source["tz"];
	    }
	}
	export class PrayerStatus {
	    enabled: boolean;
	    city: string;
	    cityName: string;
	    // Go type: time
	    now: any;
	    today: Record<string, string>;
	    nextKey: string;
	    // Go type: time
	    nextTime?: any;
	    inSeconds: number;
	
	    static createFrom(source: any = {}) {
	        return new PrayerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.city = source["city"];
	        this.cityName = source["cityName"];
	        this.now = this.convertValues(source["now"], null);
	        this.today = source["today"];
	        this.nextKey = source["nextKey"];
	        this.nextTime = this.convertValues(source["nextTime"], null);
	        this.inSeconds = source["inSeconds"];
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
	export class Settings {
	    enabled: boolean;
	    method: string;
	    asrHanafi: boolean;
	    city: string;
	    lat: number;
	    lng: number;
	    tz: string;
	    clock12h: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.method = source["method"];
	        this.asrHanafi = source["asrHanafi"];
	        this.city = source["city"];
	        this.lat = source["lat"];
	        this.lng = source["lng"];
	        this.tz = source["tz"];
	        this.clock12h = source["clock12h"];
	    }
	}

}

export namespace store {
	
	export class ActivityAppRow {
	    name: string;
	    seconds: number;
	    sessions: number;
	    category: string;
	
	    static createFrom(source: any = {}) {
	        return new ActivityAppRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.seconds = source["seconds"];
	        this.sessions = source["sessions"];
	        this.category = source["category"];
	    }
	}
	export class ActivitySegment {
	    id: string;
	    app: string;
	    title: string;
	    detail: string;
	    domain: string;
	    category: string;
	    startedAt: string;
	    endedAt?: string;
	    seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new ActivitySegment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.app = source["app"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.domain = source["domain"];
	        this.category = source["category"];
	        this.startedAt = source["startedAt"];
	        this.endedAt = source["endedAt"];
	        this.seconds = source["seconds"];
	    }
	}
	export class ActivityStats {
	    segments: number;
	    oldest: string;
	    newest: string;
	    dbBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new ActivityStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.segments = source["segments"];
	        this.oldest = source["oldest"];
	        this.newest = source["newest"];
	        this.dbBytes = source["dbBytes"];
	    }
	}
	export class ActivitySummary {
	    range: string;
	    refDay: string;
	    totalSeconds: number;
	    workSeconds: number;
	    distractionSeconds: number;
	    otherSeconds: number;
	    idleSeconds: number;
	    sessions: number;
	    byApp: ActivityAppRow[];
	    byDomain: ActivityAppRow[];
	    topDistractions: ActivityAppRow[];
	    wastedPct: number;
	    liveSeconds: number;
	
	    static createFrom(source: any = {}) {
	        return new ActivitySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.range = source["range"];
	        this.refDay = source["refDay"];
	        this.totalSeconds = source["totalSeconds"];
	        this.workSeconds = source["workSeconds"];
	        this.distractionSeconds = source["distractionSeconds"];
	        this.otherSeconds = source["otherSeconds"];
	        this.idleSeconds = source["idleSeconds"];
	        this.sessions = source["sessions"];
	        this.byApp = this.convertValues(source["byApp"], ActivityAppRow);
	        this.byDomain = this.convertValues(source["byDomain"], ActivityAppRow);
	        this.topDistractions = this.convertValues(source["topDistractions"], ActivityAppRow);
	        this.wastedPct = source["wastedPct"];
	        this.liveSeconds = source["liveSeconds"];
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
	export class Context {
	    id: string;
	    name: string;
	    color: string;
	    createdAt: string;
	    description: string;
	    descriptionAr: string;
	    dailyTargetSeconds: number;
	    maxSeconds: number;
	    elapsedSeconds: number;
	    timerStartedAt?: string;
	    totalSeconds: number;
	    todaySeconds: number;
	    tasksTodaySeconds: number;
	    tasksTotalSeconds: number;
	    recurrence: string;
	    weekSeconds: number;
	    weekTasksSeconds: number;
	
	    static createFrom(source: any = {}) {
	        return new Context(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.createdAt = source["createdAt"];
	        this.description = source["description"];
	        this.descriptionAr = source["descriptionAr"];
	        this.dailyTargetSeconds = source["dailyTargetSeconds"];
	        this.maxSeconds = source["maxSeconds"];
	        this.elapsedSeconds = source["elapsedSeconds"];
	        this.timerStartedAt = source["timerStartedAt"];
	        this.totalSeconds = source["totalSeconds"];
	        this.todaySeconds = source["todaySeconds"];
	        this.tasksTodaySeconds = source["tasksTodaySeconds"];
	        this.tasksTotalSeconds = source["tasksTotalSeconds"];
	        this.recurrence = source["recurrence"];
	        this.weekSeconds = source["weekSeconds"];
	        this.weekTasksSeconds = source["weekTasksSeconds"];
	    }
	}
	export class ContextTimeEntry {
	    id: string;
	    contextId: string;
	    startedAt: string;
	    endedAt?: string;
	    seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new ContextTimeEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.contextId = source["contextId"];
	        this.startedAt = source["startedAt"];
	        this.endedAt = source["endedAt"];
	        this.seconds = source["seconds"];
	    }
	}
	export class Horizon {
	    key: string;
	    label: string;
	    labelAr: string;
	    defaultDays: number;
	    description: string;
	    descriptionAr: string;
	    position: number;
	
	    static createFrom(source: any = {}) {
	        return new Horizon(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.labelAr = source["labelAr"];
	        this.defaultDays = source["defaultDays"];
	        this.description = source["description"];
	        this.descriptionAr = source["descriptionAr"];
	        this.position = source["position"];
	    }
	}
	export class Stats {
	    byHorizon: Record<string, any>;
	    total: number;
	    focused: number;
	    done: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.byHorizon = source["byHorizon"];
	        this.total = source["total"];
	        this.focused = source["focused"];
	        this.done = source["done"];
	    }
	}
	export class TaskDetail {
	    id: string;
	    title: string;
	    description: string;
	    horizon: string;
	    status: string;
	    contextId?: string;
	    parentId?: string;
	    priority: string;
	    startDate?: string;
	    dueDate?: string;
	    focus: boolean;
	    sortOrder: number;
	    elapsedSeconds: number;
	    timerStartedAt?: string;
	    maxSeconds: number;
	    createdAt: string;
	    updatedAt: string;
	    completedAt?: string;
	    contextName?: string;
	    contextColor?: string;
	    childCount: number;
	    todaySeconds: number;
	    totalSeconds: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.horizon = source["horizon"];
	        this.status = source["status"];
	        this.contextId = source["contextId"];
	        this.parentId = source["parentId"];
	        this.priority = source["priority"];
	        this.startDate = source["startDate"];
	        this.dueDate = source["dueDate"];
	        this.focus = source["focus"];
	        this.sortOrder = source["sortOrder"];
	        this.elapsedSeconds = source["elapsedSeconds"];
	        this.timerStartedAt = source["timerStartedAt"];
	        this.maxSeconds = source["maxSeconds"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.completedAt = source["completedAt"];
	        this.contextName = source["contextName"];
	        this.contextColor = source["contextColor"];
	        this.childCount = source["childCount"];
	        this.todaySeconds = source["todaySeconds"];
	        this.totalSeconds = source["totalSeconds"];
	    }
	}
	export class TaskFilter {
	    horizon: string;
	    status: string;
	    contextId: string;
	    focusOnly: boolean;
	    search: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.horizon = source["horizon"];
	        this.status = source["status"];
	        this.contextId = source["contextId"];
	        this.focusOnly = source["focusOnly"];
	        this.search = source["search"];
	    }
	}
	export class TaskInput {
	    title: string;
	    description: string;
	    horizon: string;
	    status: string;
	    contextId?: string;
	    parentId?: string;
	    priority: string;
	    startDate?: string;
	    dueDate?: string;
	    focus: boolean;
	    maxSeconds: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.description = source["description"];
	        this.horizon = source["horizon"];
	        this.status = source["status"];
	        this.contextId = source["contextId"];
	        this.parentId = source["parentId"];
	        this.priority = source["priority"];
	        this.startDate = source["startDate"];
	        this.dueDate = source["dueDate"];
	        this.focus = source["focus"];
	        this.maxSeconds = source["maxSeconds"];
	    }
	}
	export class TimeEntry {
	    id: string;
	    taskId: string;
	    startedAt: string;
	    endedAt?: string;
	    seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new TimeEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.taskId = source["taskId"];
	        this.startedAt = source["startedAt"];
	        this.endedAt = source["endedAt"];
	        this.seconds = source["seconds"];
	    }
	}

}

export namespace update {
	
	export class Download {
	    latest: string;
	    path: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new Download(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.latest = source["latest"];
	        this.path = source["path"];
	        this.size = source["size"];
	    }
	}
	export class Status {
	    current: string;
	    latest: string;
	    available: boolean;
	    pageUrl: string;
	    notes: string;
	    publishedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.available = source["available"];
	        this.pageUrl = source["pageUrl"];
	        this.notes = source["notes"];
	        this.publishedAt = source["publishedAt"];
	    }
	}

}

