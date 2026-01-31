export namespace models {
	
	export class DeleteRowRequest {
	    connectionId: string;
	    database: string;
	    table: string;
	    primaryKey: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new DeleteRowRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connectionId = source["connectionId"];
	        this.database = source["database"];
	        this.table = source["table"];
	        this.primaryKey = source["primaryKey"];
	    }
	}
	export class InsertRowRequest {
	    connectionId: string;
	    database: string;
	    table: string;
	    values: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new InsertRowRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connectionId = source["connectionId"];
	        this.database = source["database"];
	        this.table = source["table"];
	        this.values = source["values"];
	    }
	}
	export class UpdateRowRequest {
	    connectionId: string;
	    database: string;
	    table: string;
	    primaryKey: Record<string, any>;
	    changes: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new UpdateRowRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connectionId = source["connectionId"];
	        this.database = source["database"];
	        this.table = source["table"];
	        this.primaryKey = source["primaryKey"];
	        this.changes = source["changes"];
	    }
	}
	export class BatchUpdateRequest {
	    connectionId: string;
	    database: string;
	    table: string;
	    updates: UpdateRowRequest[];
	    inserts: InsertRowRequest[];
	    deletes: DeleteRowRequest[];
	
	    static createFrom(source: any = {}) {
	        return new BatchUpdateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connectionId = source["connectionId"];
	        this.database = source["database"];
	        this.table = source["table"];
	        this.updates = this.convertValues(source["updates"], UpdateRowRequest);
	        this.inserts = this.convertValues(source["inserts"], InsertRowRequest);
	        this.deletes = this.convertValues(source["deletes"], DeleteRowRequest);
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
	export class ConnectionRequest {
	    name: string;
	    type: string;
	    host: string;
	    port: number;
	    username: string;
	    password: string;
	    database: string;
	    sslMode: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.database = source["database"];
	        this.sslMode = source["sslMode"];
	    }
	}
	
	
	export class QueryRequest {
	    connectionId: string;
	    database: string;
	    sql: string;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new QueryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connectionId = source["connectionId"];
	        this.database = source["database"];
	        this.sql = source["sql"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}

}

export namespace services {
	
	export class Response {
	    success: boolean;
	    data?: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = source["data"];
	        this.error = source["error"];
	    }
	}

}

