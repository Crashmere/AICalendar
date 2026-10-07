export interface Span {start:string;end:string}
export interface RecordData {source:string;external_id:string;conversation_id?:string;activity_date:string;timezone:string;first_activity_at:string|null;last_activity_at:string|null;user_message_count:number|null;title:string;summary:string;tags:string[];spans:Span[];quality:{count:string;time:string;topic:string};provenance:{method:string;producer?:string;source_ref?:string;produced_at?:string};record_state:string}
export interface Annotation {title?:string;summary?:string;tags?:string[];hidden:boolean}
export interface Activity {id:string;version:number;annotation_version:number;record:RecordData;annotation:Annotation;updated_at:string}
export interface Coverage {source:string;from:string;to:string;status:string;note?:string}
export interface ImportResult {id?:string;created_at?:string;committed:boolean;replay:boolean;inserted:number;updated:number;unchanged:number;conflicts:number;items?:{source:string;external_id:string;action:string;reason?:string}[];coverage:Coverage[]}
