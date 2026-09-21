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

export namespace store {
	
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
	    todaySeconds: number;
	    tasksTodaySeconds: number;
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
	        this.todaySeconds = source["todaySeconds"];
	        this.tasksTodaySeconds = source["tasksTodaySeconds"];
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

