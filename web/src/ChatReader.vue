<script setup lang="ts">
import {ref,computed,watch,nextTick} from 'vue'
import {Download,LoaderCircle,Search,X,ChevronDown,ChevronRight,Wrench} from 'lucide-vue-next'
import {base} from './api'
import type {ChatTarget,Message,Snapshot} from './types'
import {sourceColor,number,dateTime} from './present'
import {weekdayName} from './dates'
import {renderMarkdown} from './markdown'

interface Merged {key:string;id:string;role:string;at:string|null;day:string|null;content:string;text:string;attributes?:Record<string,unknown>}
type Entry={type:'date';day:string}|{type:'message';message:Merged}|{type:'run';key:string;messages:Merged[]}

const props=defineProps<{target:ChatTarget;timezone?:string}>()
const root=base+'api/v1/archives'
const snapshots=ref<Snapshot[]>([]),snapshotId=ref(''),fragments=ref<Message[]>([])
const after=ref(-1),more=ref(false),busy=ref(false),error=ref(''),loaded=ref(false)
const mode=ref<'chat'|'all'>('chat'),query=ref(''),expanded=ref(new Set<string>())
const snapshot=computed(()=>snapshots.value.find(s=>s.id===snapshotId.value))
const dayFormat=new Intl.DateTimeFormat('en-CA',{timeZone:props.timezone||'Asia/Shanghai',year:'numeric',month:'2-digit',day:'2-digit'})
const timeFormat=new Intl.DateTimeFormat('zh-CN',{timeZone:props.timezone||'Asia/Shanghai',hour:'2-digit',minute:'2-digit',hour12:false})
const roles:Record<string,string>={user:'你',assistant:'AI',reasoning:'思考',tool:'工具',system:'系统',developer:'指令',unknown:'其他'}
const isChat=(m:Merged)=>m.role==='user'||m.role==='assistant'

function texts(x:unknown):string[]{
 if(typeof x==='string')return x.trim()?[x]:[]
 if(!Array.isArray(x))return []
 return x.flatMap(p=>typeof p==='string'?texts(p):typeof p?.text==='string'?texts(p.text):typeof p?.summary==='string'?texts(p.summary):p?.asset_pointer||p?.content_type==='image_asset_pointer'?['[图片]']:[])
}
const first=(...xs:unknown[])=>xs.map(x=>texts(x).join('\n\n')).find(Boolean)||''
const citation=/\ue200[^\ue201]*\ue201|[\ue200-\ue2ff]/g
function normalize(m:Merged){
 const raw=m.content.trimStart()
 if(raw.startsWith('{')){
  let v:any
  try{v=JSON.parse(raw)}catch{v=null}
  const kind=typeof v?.type==='string'?v.type:typeof v?.content_type==='string'?v.content_type:''
  if(v&&typeof v==='object'){
   if(kind==='reasoning'||kind==='thoughts'||kind==='reasoning_recap'||(!kind&&typeof v.signature==='string')){
    m.role='reasoning';m.text=first(v.summary,v.thoughts,v.content,v.text)||m.content;return
   }
   if(isChat(m)&&(kind===''||kind==='text'||kind==='multimodal_text'||kind.endsWith('Message'))){
    const s=first(v.parts,v.content,v.text)
    if(s)m.text=s
    else if(m.role==='assistant')m.role='tool'
   }else if(m.role==='assistant'&&kind)m.role='tool'
  }
 }
 if(m.role==='assistant')m.text=m.text.replace(citation,'')
}

async function request(url:string){const r=await fetch(url);if(r.status===401)throw Error('设备登录已失效，请从服务器门户重新登录。');const data=await r.json();if(!r.ok)throw Error(data.error||'存档读取失败');return data}

const messages=computed(()=>{
 const out:Merged[]=[]
 for(const f of fragments.value){
  const last=out[out.length-1]
  if(last&&last.id===f.id&&f.part>0){last.content+=f.content;continue}
  out.push({key:f.id+':'+out.length,id:f.id,role:f.role,at:f.at,day:f.at?dayFormat.format(new Date(f.at)):null,content:f.content,text:'',attributes:f.attributes})
 }
 for(const m of out){m.text=m.content;normalize(m)}
 return out
})
const matches=computed(()=>{const q=query.value.trim().toLowerCase();return q?messages.value.filter(m=>m.text.toLowerCase().includes(q)):null})
const entries=computed(()=>{
 const out:Entry[]=[];let day:string|null=null,run:Merged[]|null=null
 const collapse=mode.value==='chat'&&!matches.value&&messages.value.some(isChat)
 for(const m of matches.value??messages.value){
  if(m.day&&m.day!==day){day=m.day;run=null;out.push({type:'date',day})}
  if(collapse&&!isChat(m)){
   if(!run){run=[];out.push({type:'run',key:m.key,messages:run})}
   run.push(m)
  }else{run=null;out.push({type:'message',message:m})}
 }
 return out
})
const counts=computed(()=>({user:messages.value.filter(m=>m.role==='user').length,chat:messages.value.filter(isChat).length,other:messages.value.filter(m=>!isChat(m)).length}))

async function loadPage(){
 const id=snapshotId.value
 const result=await request(`${root}/${id}/messages?after=${after.value}&limit=200`)
 if(snapshotId.value!==id)return false
 fragments.value.push(...result.items);after.value=result.next_after;more.value=result.has_more
 return true
}
async function load(until:(()=>boolean)|null){
 busy.value=true;error.value=''
 try{while(await loadPage()&&more.value&&until&&!until());}catch(e){error.value=(e as Error).message}finally{busy.value=false}
}
async function openSnapshot(id:string){
 snapshotId.value=id;fragments.value=[];after.value=-1;more.value=false;expanded.value=new Set()
 const date=props.target.date
 await load(()=>date?messages.value.some(m=>m.day!==null&&m.day>date):messages.value.length>=300)
 if(date){await nextTick();document.querySelector(`.chat-reader [data-day="${date}"]`)?.scrollIntoView({block:'start'})}
}
async function init(){
 busy.value=true;error.value='';loaded.value=false;snapshots.value=[]
 try{
  const q=new URLSearchParams({source:props.target.source,conversation_id:props.target.conversationId,limit:'100'})
  snapshots.value=(await request(`${root}?${q}`)).items
 }catch(e){error.value=(e as Error).message}finally{busy.value=false;loaded.value=true}
 const first=props.target.snapshotId&&snapshots.value.some(s=>s.id===props.target.snapshotId)?props.target.snapshotId:snapshots.value[0]?.id
 if(first)await openSnapshot(first)
}
function toggleRun(key:string){const s=new Set(expanded.value);s.has(key)?s.delete(key):s.add(key);expanded.value=s}
function toolSummary(m:Merged){
 if(m.role==='reasoning')return m.text.replace(/[*#]/g,'').replace(/\s+/g,' ').slice(0,140)
 try{const v=JSON.parse(m.content);const parts=[v.type,v.name,v.command,v.tool,v.title].filter(x=>typeof x==='string');if(parts.length)return parts.join(' · ').slice(0,140)}catch{}
 return m.content.replace(/\s+/g,' ').slice(0,140)||'（空消息）'
}
const dayTitle=(d:string)=>`${d.slice(0,4)} 年 ${+d.slice(5,7)} 月 ${+d.slice(8,10)} 日 ${weekdayName(d)}`
watch(()=>[props.target.source,props.target.conversationId,props.target.snapshotId],init,{immediate:true})
</script>

<template>
 <section class="chat-reader">
  <div v-if="error" class="modal-error" role="alert">{{error}}</div>
  <div v-if="loaded&&!snapshots.length" class="empty-state compact"><h3>没有找到这段会话的存档</h3><p>下次使用导入 skill 时，可以连同完整聊天一起保存。</p></div>
  <template v-if="snapshot">
   <header class="chat-head">
    <h2>{{snapshot.manifest.title||'完整对话'}}</h2>
    <p><i class="source-dot" :style="{background:sourceColor(snapshot.manifest.source)}"></i>{{snapshot.manifest.source_label||snapshot.manifest.source}} · {{number(snapshot.manifest.message_count)}} 条消息 · 整理于 {{dateTime(snapshot.manifest.captured_at)}}</p>
    <details v-if="snapshot.manifest.coverage!=='complete'&&snapshot.manifest.note" class="chat-note"><summary>来源完整性：{{snapshot.manifest.coverage==='partial'?'部分可恢复':'尚未确认'}}</summary><p>{{snapshot.manifest.note}}</p></details>
    <div class="chat-tools">
     <div class="segmented"><button :class="{active:mode==='chat'}" @click="mode='chat'">只看对话</button><button :class="{active:mode==='all'}" @click="mode='all'">全部消息</button></div>
     <div class="search small"><Search :size="15"/><input v-model="query" placeholder="在会话中搜索" aria-label="在会话中搜索"/><button v-if="query" @click="query=''" aria-label="清除搜索"><X :size="14"/></button></div>
     <select v-if="snapshots.length>1" :value="snapshotId" @change="openSnapshot(($event.target as HTMLSelectElement).value)" aria-label="选择快照"><option v-for="(s,i) in snapshots" :key="s.id" :value="s.id">{{i===0?'最新快照':'旧快照'}} · {{dateTime(s.manifest.captured_at)}} · {{s.manifest.message_count}} 条</option></select>
     <a class="secondary" :href="`${root}/${snapshot.id}/download`"><Download :size="15"/>下载存档</a>
    </div>
    <p v-if="matches" class="chat-status">找到 {{matches.length}} 条匹配消息{{more?`（已读取 ${messages.length} / ${snapshot.manifest.message_count} 条）`:''}}<button v-if="more" class="text-button" :disabled="busy" @click="load(()=>false)">读取全部后搜索</button></p>
    <p v-else-if="messages.length" class="chat-status">{{more?`已读取 ${number(messages.length)} / ${number(snapshot.manifest.message_count)} 条 · `:''}}你发送 {{counts.user}} 条<template v-if="counts.other"> · 工具、思考与系统消息 {{counts.other}} 条{{mode==='chat'&&counts.chat?'（已折叠）':''}}</template></p>
   </header>
   <div v-if="!messages.length&&!busy&&!more" class="empty-state compact"><p>此存档保留了原始资料，暂无可展示的统一消息。</p></div>
   <template v-for="e in entries" :key="e.type==='date'?'d'+e.day:e.type==='run'?'r'+e.key:e.message.key">
    <div v-if="e.type==='date'" class="chat-day" :data-day="e.day">{{dayTitle(e.day)}}</div>
    <div v-else-if="e.type==='run'" class="chat-run">
     <button class="run-toggle" @click="toggleRun(e.key)"><component :is="expanded.has(e.key)?ChevronDown:ChevronRight" :size="15"/><Wrench :size="14"/>{{e.messages.length}} 条工具、思考与系统消息<small>{{toolSummary(e.messages[0])}}</small></button>
     <template v-if="expanded.has(e.key)"><details v-for="m in e.messages" :key="m.key" class="tool-message"><summary><strong>{{roles[m.role]||m.role}}</strong><span>{{m.at?timeFormat.format(new Date(m.at)):''}}</span><small>{{toolSummary(m)}}</small></summary><pre>{{m.text}}</pre></details></template>
    </div>
    <article v-else-if="isChat(e.message)" class="chat-message" :class="e.message.role">
     <div class="chat-meta"><strong>{{roles[e.message.role]}}</strong><span>{{e.message.at?timeFormat.format(new Date(e.message.at)):'时间未知'}}</span></div>
     <div v-if="e.message.role==='assistant'&&e.message.text.length<150000" class="markdown" v-html="renderMarkdown(e.message.key+snapshotId,e.message.text)"></div>
     <pre v-else>{{e.message.text||'（空消息，原始结构保存在存档中）'}}</pre>
     <details v-if="mode==='all'&&e.message.attributes&&Object.keys(e.message.attributes).length" class="attributes"><summary>来源信息</summary><pre>{{JSON.stringify(e.message.attributes,null,2)}}</pre></details>
    </article>
    <details v-else class="tool-message standalone"><summary><strong>{{roles[e.message.role]||e.message.role}}</strong><span>{{e.message.at?timeFormat.format(new Date(e.message.at)):''}}</span><small>{{toolSummary(e.message)}}</small></summary><pre>{{e.message.text}}</pre></details>
   </template>
   <div v-if="more&&!busy" class="chat-more"><button class="secondary" @click="load(null)">继续加载</button><button class="text-button" @click="load(()=>false)">读取全部 {{number(snapshot.manifest.message_count)}} 条</button></div>
  </template>
  <div v-if="busy" class="loading small"><LoaderCircle class="spin" :size="20"/>正在读取存档…</div>
 </section>
</template>
