import type {Activity,RecordData} from './types'

const known:[RegExp,string][]=[[/chatgpt|openai/i,'#10a37f'],[/gemini/i,'#4a7bec'],[/claude|anthropic/i,'#d4734f'],[/cursor/i,'#4b5563'],[/trae/i,'#8b5cf6']]
const fallback=['#0f8a8a','#c2417a','#b38a12','#3f6ad8','#6b8e23','#9a5b3c']
export function sourceColor(source:string){
 const hit=known.find(([pattern])=>pattern.test(source))
 if(hit)return hit[1]
 let h=0;for(const c of source)h=(h*31+c.charCodeAt(0))>>>0
 return fallback[h%fallback.length]
}

export const title=(a:Activity)=>a.annotation.title??a.record.title
export const summary=(a:Activity)=>a.annotation.summary??a.record.summary
export const topicTags=(a:Activity)=>a.annotation.tags??a.record.tags
export const sourceName=(r:RecordData)=>r.source_label||r.source
export const count=(a:Activity)=>a.record.user_message_count??0

export function timeLabel(s:string|null,timezone='Asia/Shanghai'){return s?new Intl.DateTimeFormat('zh-CN',{timeZone:timezone,hour:'2-digit',minute:'2-digit',hour12:false}).format(new Date(s)):'时间未记录'}
export function countLabel(r:RecordData){return r.user_message_count===null?'次数未知':`${r.quality.count==='estimated'?'约 ':''}${r.user_message_count} 次交流`}
export function dateTime(s?:string|null){return s?new Intl.DateTimeFormat('zh-CN',{timeZone:'Asia/Shanghai',year:'numeric',month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit',hour12:false}).format(new Date(s)):'尚未导入'}
export const number=(n:number)=>n.toLocaleString('zh-CN')

export const qualityCount:Record<string,string>={observed:'按发送记录直接计数',estimated:'估算次数',unknown:'次数未知'}
export const qualityTime:Record<string,string>={observed_interval:'记录了起止时间',observed_timestamps:'记录了发送时间点',estimated_interaction_span:'估算的互动区间',date_only:'只知道日期',unknown:'时间未知'}
export const qualityTopic:Record<string,string>={source_title:'来自平台标题',agent_summary:'由 agent 归纳',manual:'手动整理'}

export function mergedMinutes(rows:Activity[]){
 const intervals=rows.flatMap(a=>a.record.spans.map(s=>[Date.parse(s.start),Date.parse(s.end)] as [number,number])).sort((a,b)=>a[0]-b[0])
 if(!intervals.length)return null
 let total=0,[a,b]=intervals[0]
 for(const [c,d] of intervals.slice(1)){if(c<=b)b=Math.max(b,d);else{total+=b-a;a=c;b=d}}
 return Math.round((total+b-a)/60000)
}
