<script setup lang="ts">
import {ref,computed,watch,onMounted,nextTick} from 'vue'
import {CalendarDays,ChevronLeft,ChevronRight,Search,ArrowUpRight,History,Layers,MessageCircle,Clock,Plus,X,Check,EyeOff,LoaderCircle,CalendarCheck} from 'lucide-vue-next'
import {api,allActivities,base} from './api'
import type {Activity,ImportResult,ChatTarget,DaySummary,Conversation} from './types'
import {type Range,monthRange,quarterRange,yearRange,customRange,shiftRange,encodeRange,decodeRange,rangeLabel,dayLabel,weekdayName,inRange,diffDays,addDays} from './dates'
import {title,summary,topicTags,sourceName,sourceColor,timeLabel,countLabel,dateTime,number,mergedMinutes} from './present'
import CalendarView from './CalendarView.vue'
import RangePicker from './RangePicker.vue'
import TagPicker from './TagPicker.vue'
import ActivityDetail from './ActivityDetail.vue'
import ChatReader from './ChatReader.vue'
import ArchivesPage from './ArchivesPage.vue'
import ImportsPage from './ImportsPage.vue'
import ImportDialog from './ImportDialog.vue'

type View='calendar'|'records'|'archives'|'imports'
const views:{id:View;label:string;icon:any}[]=[{id:'calendar',label:'日历',icon:CalendarDays},{id:'records',label:'记录',icon:Layers},{id:'archives',label:'聊天存档',icon:MessageCircle},{id:'imports',label:'导入与覆盖',icon:History}]
const headings:Record<View,string>={calendar:'留住每一次探索',records:'那些聊过的事',archives:'保留每一段对话',imports:'记录从这里汇集'}

const activities=ref<Activity[]>([]),imports=ref<ImportResult[]>([])
const loading=ref(true),error=ref(''),notice=ref(''),demo=ref(false)
const today=ref(new Intl.DateTimeFormat('en-CA',{timeZone:'Asia/Shanghai'}).format(new Date()))
const view=ref<View>('calendar'),range=ref<Range>(monthRange(+today.value.slice(0,4),+today.value.slice(5,7))),selected=ref(today.value)
const q=ref(''),sources=ref<string[]>([]),tags=ref<string[]>([]),showHidden=ref(false)
const detail=ref<Activity|null>(null),chat=ref<ChatTarget|null>(null),importOpen=ref(false),shownRecords=ref(150)
const rangePicker=ref<InstanceType<typeof RangePicker>>(),tagPicker=ref<InstanceType<typeof TagPicker>>()
const detailRef=ref<InstanceType<typeof ActivityDetail>>(),importRef=ref<InstanceType<typeof ImportDialog>>()
let noticeTimer:ReturnType<typeof setTimeout>|undefined
function toast(s:string){notice.value=s;clearTimeout(noticeTimer);noticeTimer=setTimeout(()=>notice.value='',4500)}

const allRange=computed<Range>(()=>({kind:'all',from:activities.value.reduce((m,a)=>a.record.activity_date<m?a.record.activity_date:m,today.value),to:today.value}))
const current=computed<Range>(()=>range.value.kind==='all'?allRange.value:range.value)
const label=computed(()=>rangeLabel(current.value,today.value))
const names=computed(()=>{const m=new Map<string,string>();for(const a of activities.value)if(a.record.source_label||!m.has(a.record.source))m.set(a.record.source,sourceName(a.record));return m})
const allSources=computed(()=>[...names.value.keys()].sort())

const textOf=(a:Activity)=>(title(a)+' '+summary(a)+' '+topicTags(a).join(' ')+' '+sourceName(a.record)).toLowerCase()
const visible=computed(()=>activities.value.filter(a=>showHidden.value||!a.annotation.hidden))
const bySourceText=computed(()=>{const query=q.value.trim().toLowerCase();return visible.value.filter(a=>(!sources.value.length||sources.value.includes(a.record.source))&&(!query||textOf(a).includes(query)))})
const filtered=computed(()=>tags.value.length?bySourceText.value.filter(a=>topicTags(a).some(t=>tags.value.includes(t))):bySourceText.value)
const ranged=computed(()=>filtered.value.filter(a=>inRange(a.record.activity_date,current.value)))
const narrowing=computed(()=>!!q.value.trim()||tags.value.length>0)

const days=computed(()=>{
 const m=new Map<string,Activity[]>()
 for(const a of filtered.value){const d=a.record.activity_date;const list=m.get(d);list?list.push(a):m.set(d,[a])}
 const out=new Map<string,DaySummary>()
 for(const [date,items] of m){
  items.sort((a,b)=>(a.record.first_activity_at||'z').localeCompare(b.record.first_activity_at||'z'))
  const top=items.reduce((best,a)=>(a.record.user_message_count??-1)>(best.record.user_message_count??-1)?a:best)
  out.set(date,{date,items,top,messages:items.reduce((n,a)=>n+(a.record.user_message_count||0),0),unknown:items.some(a=>a.record.user_message_count===null),sources:[...new Set(items.map(a=>a.record.source))]})
 }
 return out
})
const thresholds=computed(()=>{const v=[...days.value.values()].map(d=>d.messages).sort((a,b)=>a-b);const at=(p:number)=>v[Math.min(v.length-1,Math.floor(p*v.length))]||0;return [at(.4),at(.7),at(.9)]})
function level(d?:DaySummary){if(!d)return 0;const [a,b,c]=thresholds.value;return d.messages<=a?1:d.messages<=b?2:d.messages<=c?3:4}

function stats(items:Activity[]){
 const known=items.filter(a=>a.record.user_message_count!==null)
 return {messages:known.reduce((n,a)=>n+(a.record.user_message_count||0),0),unknown:items.length-known.length,estimated:known.some(a=>a.record.quality.count!=='observed'),days:new Set(items.filter(a=>a.record.user_message_count!==0).map(a=>a.record.activity_date)).size,conversations:new Set(items.filter(a=>a.record.conversation_id).map(a=>a.record.source+'\0'+a.record.conversation_id)).size,topics:items.length}
}
const now=computed(()=>stats(ranged.value))
const ongoing=computed(()=>current.value.from<=today.value&&current.value.to>today.value)
const previous=computed(()=>{
 if(range.value.kind==='all')return null
 const p=shiftRange(current.value,-1)
 if(ongoing.value){const to=addDays(p.from,diffDays(current.value.from,today.value));if(to<p.to)p.to=to}
 return stats(filtered.value.filter(a=>inRange(a.record.activity_date,p)))
})
function delta(a:number,b?:number){if(b===undefined)return '';const label=ongoing.value?'较上期同期':'较上期';if(!b)return a?`${label}没有记录`:'';const pct=Math.round((a-b)/b*100);return `${label} ${pct>0?'+':''}${pct}%`}
const spanDays=computed(()=>Math.min(diffDays(current.value.from,current.value.to),diffDays(current.value.from,today.value))+1)
const shares=computed(()=>{const m=new Map<string,number>();for(const a of ranged.value)m.set(a.record.source,(m.get(a.record.source)||0)+(a.record.user_message_count||0));const total=[...m.values()].reduce((x,y)=>x+y,0)||1;return [...m].sort((a,b)=>b[1]-a[1]).map(([s,n])=>({source:s,n,pct:n/total*100}))})
const sourceCounts=computed(()=>{const m=new Map<string,number>();for(const a of visible.value)if(inRange(a.record.activity_date,current.value))m.set(a.record.source,(m.get(a.record.source)||0)+1);return m})
const tagOptions=computed(()=>{const m=new Map<string,number>();for(const a of bySourceText.value)if(inRange(a.record.activity_date,current.value))for(const t of topicTags(a))m.set(t,(m.get(t)||0)+1);for(const t of tags.value)if(!m.has(t))m.set(t,0);return [...m].sort((a,b)=>b[1]-a[1]||a[0].localeCompare(b[0])).map(([tag,count])=>({tag,count}))})
const hiddenCount=computed(()=>activities.value.filter(a=>a.annotation.hidden).length)

const day=computed(()=>days.value.get(selected.value))
const dayItems=computed(()=>inRange(selected.value,current.value)?day.value?.items||[]:[])
const coverages=computed(()=>imports.value.flatMap(i=>i.coverage||[]))
const dayCoverage=computed(()=>coverages.value.filter(c=>c.from<=selected.value&&c.to>=selected.value&&(!sources.value.length||sources.value.includes(c.source))))
const recordDays=computed(()=>{
 const sorted=[...ranged.value].sort((a,b)=>b.record.activity_date.localeCompare(a.record.activity_date)||(a.record.first_activity_at||'').localeCompare(b.record.first_activity_at||''))
 const out:{date:string;items:Activity[];messages:number}[]=[]
 for(const a of sorted.slice(0,shownRecords.value)){const last=out[out.length-1];if(last?.date===a.record.activity_date){last.items.push(a);last.messages+=a.record.user_message_count||0}else out.push({date:a.record.activity_date,items:[a],messages:a.record.user_message_count||0})}
 return out
})
const conversations=computed(()=>{
 const m=new Map<string,Conversation>()
 for(const a of ranged.value){
  const r=a.record;if(!r.conversation_id)continue
  const key=r.source+'\0'+r.conversation_id;const c=m.get(key)
  if(!c){m.set(key,{key,source:r.source,sourceName:sourceName(r),conversationId:r.conversation_id,title:title(a),first:r.activity_date,last:r.activity_date,days:1,messages:r.user_message_count||0,unknown:r.user_message_count===null});continue}
  c.days++;c.messages+=r.user_message_count||0;c.unknown||=r.user_message_count===null
  if(r.activity_date<c.first)c.first=r.activity_date
  if(r.activity_date>c.last){c.last=r.activity_date;c.title=title(a)}
 }
 return [...m.values()].sort((a,b)=>b.last.localeCompare(a.last))
})
const related=computed(()=>{const r=detail.value?.record;return r?.conversation_id?activities.value.filter(a=>a.record.source===r.source&&a.record.conversation_id===r.conversation_id):[]})
const activeMonths=computed(()=>new Set(activities.value.map(a=>a.record.activity_date.slice(0,7))))

function defaultDay(r:Range){
 let best='';for(const d of days.value.keys())if(inRange(d,r)&&d<=today.value&&d>best)best=d
 return best||(inRange(today.value,r)?today.value:r.to)
}
function setRange(r:Range){range.value=r;shownRecords.value=150;if(!inRange(selected.value,r.kind==='all'?allRange.value:r))selected.value=defaultDay(r.kind==='all'?allRange.value:r)}
function move(n:number){setRange(shiftRange(current.value,n))}
function goToday(){
 const y=+today.value.slice(0,4),m=+today.value.slice(5,7),k=range.value.kind
 setRange(k==='month'?monthRange(y,m):k==='quarter'?quarterRange(y,Math.floor((m-1)/3)+1):k==='year'?yearRange(y):k==='custom'?customRange(addDays(today.value,-diffDays(current.value.from,current.value.to)),today.value):range.value)
 selected.value=today.value
}
function toggleSource(s:string){sources.value=sources.value.includes(s)?sources.value.filter(x=>x!==s):[...sources.value,s]}
function clearFilters(){q.value='';sources.value=[];tags.value=[]}
function openChat(t:ChatTarget){chat.value=t}
function saved(a:Activity){const i=activities.value.findIndex(x=>x.id===a.id);if(i>=0)activities.value.splice(i,1,a);detail.value=a;toast('修改已保存，重新导入也会保留')}
async function imported(r:ImportResult){importOpen.value=false;await load();toast(r.replay?'这批记录已经导入，无需重复写入':`已导入：新增 ${r.inserted} 条，更新 ${r.updated} 条`)}

async function load(){
 loading.value=true;error.value=''
 try{const [info,batches,items]=await Promise.all([api('info'),api('imports'),allActivities()]);demo.value=info.demo;today.value=info.today;imports.value=batches.items;activities.value=items}
 catch(e){error.value=(e as Error).message}finally{loading.value=false}
}

let restoring=false
function readURL(){
 restoring=true
 const p=new URLSearchParams(location.search)
 const v=p.get('view') as View;view.value=views.some(x=>x.id===v)?v:'calendar'
 const r=p.get('range');range.value=(r&&decodeRange(r,allRange.value))||monthRange(+today.value.slice(0,4),+today.value.slice(5,7))
 const d=p.get('day');selected.value=d&&/^\d{4}-\d{2}-\d{2}$/.test(d)?d:defaultDay(current.value)
 q.value=p.get('q')||'';sources.value=p.get('source')?.split(',').filter(Boolean)||[];tags.value=p.getAll('tag');showHidden.value=p.get('hidden')==='1'
 nextTick(()=>restoring=false)
}
function writeURL(push:boolean){
 if(restoring)return
 const p=new URLSearchParams()
 if(view.value!=='calendar')p.set('view',view.value)
 p.set('range',encodeRange(range.value));if(view.value==='calendar')p.set('day',selected.value)
 if(q.value.trim())p.set('q',q.value.trim());if(sources.value.length)p.set('source',sources.value.join(','));for(const t of tags.value)p.append('tag',t);if(showHidden.value)p.set('hidden','1')
 const url=location.pathname+'?'+p.toString()
 if(url===location.pathname+location.search)return
 push?history.pushState(null,'',url):history.replaceState(null,'',url)
}
watch([view,()=>encodeRange(range.value)],()=>writeURL(true))
watch([selected,q,sources,tags,showHidden],()=>writeURL(false))
watch(()=>[q.value,sources.value.join(),tags.value.join()],()=>shownRecords.value=150)

function onKey(e:KeyboardEvent){
 if(e.key!=='Escape')return
 if(rangePicker.value?.close()||tagPicker.value?.close())return
 if(chat.value){chat.value=null;return}
 if(detail.value){detailRef.value?.close();return}
 if(importOpen.value)importRef.value?.close()
}
onMounted(async()=>{readURL();window.addEventListener('popstate',readURL);document.addEventListener('keydown',onKey);await load();if(!new URLSearchParams(location.search).get('day'))selected.value=defaultDay(current.value)})
</script>

<template>
 <div class="app-shell">
  <aside class="sidebar">
   <a class="brand" :href="base"><span class="brand-mark"><CalendarDays :size="22"/></span><span>AICalendar<small>AI 活动日历</small></span></a>
   <nav><button v-for="v in views" :key="v.id" :class="{active:view===v.id}" @click="view=v.id" :aria-label="v.label"><component :is="v.icon" :size="18"/><span>{{v.label}}</span></button></nav>
   <div class="sidebar-bottom"><p>最近整理<strong>{{dateTime(imports[0]?.created_at)}}</strong></p><a class="portal-link" href="/portal/">服务器门户<ArrowUpRight :size="15"/></a></div>
  </aside>
  <main>
   <header class="page-header"><div><h1>{{headings[view]}}</h1><p>{{view==='imports'?'每个来源整理到了哪里，每次导入带来了什么。':'回顾与 AI 一起思考、创造和解决问题的日子。'}}</p></div><button class="primary" @click="importOpen=true"><Plus :size="17"/>导入记录</button></header>
   <div v-if="demo" class="demo-label">本地演示环境 · 使用合成数据</div>
   <div v-if="error" class="error-banner" role="alert">{{error}}<button @click="error=''" aria-label="关闭错误"><X :size="17"/></button></div>
   <div v-if="notice" class="toast" role="status"><Check :size="17"/>{{notice}}</div>
   <div v-if="loading&&!activities.length" class="loading"><LoaderCircle class="spin" :size="22"/>正在整理页面…</div>
   <ImportsPage v-else-if="view==='imports'" :imports="imports" :activities="activities" :names="names" :today="today" @import="importOpen=true"/>
   <template v-else>
    <section class="toolbar">
     <div class="range-nav"><button class="icon-button" @click="move(-1)" :disabled="range.kind==='all'" aria-label="上一段时间"><ChevronLeft :size="18"/></button><RangePicker ref="rangePicker" :model-value="current" @update:model-value="setRange" :today="today" :all="allRange" :active-months="activeMonths"/><button class="icon-button" @click="move(1)" :disabled="range.kind==='all'" aria-label="下一段时间"><ChevronRight :size="18"/></button><button class="today-button" @click="goToday"><CalendarCheck :size="15"/>今天</button></div>
     <div class="search"><Search :size="17"/><input v-model="q" placeholder="搜索主题、摘要、标签…" aria-label="搜索记录"/><button v-if="q" @click="q=''" aria-label="清除搜索"><X :size="15"/></button></div>
    </section>
    <section class="filters">
     <button v-for="s in allSources" :key="s" class="source-chip" :class="{on:sources.includes(s),off:sources.length&&!sources.includes(s)}" @click="toggleSource(s)"><i :style="{background:sourceColor(s)}"></i>{{names.get(s)}}<small>{{number(sourceCounts.get(s)||0)}}</small></button>
     <TagPicker ref="tagPicker" v-model="tags" :options="tagOptions"/>
     <label v-if="hiddenCount" class="hidden-toggle"><input type="checkbox" v-model="showHidden"/>显示 {{hiddenCount}} 条隐藏记录</label>
     <span v-for="t in tags" :key="t" class="active-tag">{{t}}<button @click="tags=tags.filter(x=>x!==t)" :aria-label="`移除标签 ${t}`"><X :size="12"/></button></span>
    </section>
    <p v-if="narrowing||sources.length" class="filter-status">{{label}}内有 {{number(ranged.length)}} 条匹配记录<template v-if="narrowing&&filtered.length>ranged.length">，其他时间还有 {{number(filtered.length-ranged.length)}} 条<button class="text-button" @click="setRange({kind:'all',from:'',to:''})">查看全部时间</button></template><button class="text-button" @click="clearFilters">清除筛选</button></p>
    <section class="stats">
     <div class="stat"><span>交流次数<MessageCircle :size="16"/></span><strong>{{now.estimated?'约 ':''}}{{now.unknown===now.topics&&now.unknown?'—':number(now.messages)}}<small>次</small></strong><p>{{delta(now.messages,previous?.messages)||(now.unknown?`另有 ${now.unknown} 条次数未知`:'按你发送的消息计数')}}</p><div v-if="shares.length>1" class="share-bar" :title="shares.map(s=>`${names.get(s.source)} ${Math.round(s.pct)}%`).join(' · ')"><i v-for="s in shares" :key="s.source" :style="{width:s.pct+'%',background:sourceColor(s.source)}"></i></div></div>
     <div class="stat"><span>活跃天数<CalendarDays :size="16"/></span><strong>{{number(now.days)}}<small>/ {{number(spanDays)}} 天</small></strong><p>{{delta(now.days,previous?.days)||(now.days?`活跃日平均 ${Math.round(now.messages/now.days)} 次`:'基于已导入的记录')}}</p></div>
     <div class="stat"><span>会话<Layers :size="16"/></span><strong>{{number(now.conversations)}}<small>个</small></strong><p>{{delta(now.conversations,previous?.conversations)||'同一会话跨天只计一次'}}</p></div>
     <div class="stat"><span>主题记录<Clock :size="16"/></span><strong>{{number(now.topics)}}<small>条</small></strong><p>{{delta(now.topics,previous?.topics)||'每个会话每天一条'}}</p></div>
    </section>
    <section v-if="view==='calendar'" class="workspace calendar-layout">
     <CalendarView :range="current" :days="days" :selected="selected" :today="today" :level="level" @select="selected=$event" @range="setRange"/>
     <aside class="day-panel">
      <div class="day-panel-heading"><div><h2>{{dayLabel(selected)}}</h2><p>{{selected.slice(0,4)}} 年 · {{weekdayName(selected)}}<span v-if="selected===today"> · 今天</span></p></div><span v-if="dayItems.length" class="day-badge">{{day?.unknown&&!day?.messages?'次数未知':number(day?.messages||0)+' 次'}} · {{dayItems.length}} 条</span></div>
      <div v-if="mergedMinutes(dayItems)!==null" class="duration-note"><Clock :size="13"/>互动区间合计约 {{mergedMinutes(dayItems)}} 分钟<span title="按可恢复区间去重合并，不代表本人专注时间">ⓘ</span></div>
      <div v-if="!dayItems.length" class="empty-state compact"><CalendarDays :size="32"/><h3>{{narrowing||sources.length?'这天没有匹配的记录':dayCoverage.length?'这天没有记录':'这一天，还待整理'}}</h3><p>{{narrowing||sources.length?'试试调整筛选条件。':dayCoverage.length?'已有来源的导入范围覆盖这一天。':'未导入的日期不代表没有使用。'}}</p></div>
      <div v-else class="timeline">
       <button v-for="a in dayItems" :key="a.id" class="activity-card" :class="{muted:a.annotation.hidden}" @click="detail=a">
        <div class="card-meta"><i class="source-dot" :style="{background:sourceColor(a.record.source)}"></i><span>{{sourceName(a.record)}}</span><EyeOff v-if="a.annotation.hidden" :size="13"/><span class="card-time">{{timeLabel(a.record.first_activity_at,a.record.timezone)}}</span></div>
        <h3>{{title(a)}}</h3><p>{{summary(a)||'暂时没有摘要'}}</p>
        <div class="card-footer"><span class="tags"><span v-for="t in topicTags(a).slice(0,3)" :key="t">{{t}}</span></span><span class="card-count">{{countLabel(a.record)}}</span><span v-if="a.record.record_state==='partial'" class="partial">待补全</span></div>
       </button>
      </div>
      <p v-if="dayCoverage.length" class="coverage-note">已有 {{new Set(dayCoverage.map(c=>c.source)).size}} 个来源的导入范围覆盖此日{{dayCoverage.some(c=>c.status!=='complete')?'，其中部分资料不完整':''}}。</p>
     </aside>
    </section>
    <section v-else-if="view==='records'" class="workspace records-list">
     <div v-if="!recordDays.length" class="empty-state"><Search :size="34"/><h3>{{label}}内没有匹配的记录</h3><p>换个时间范围或调整筛选条件。</p></div>
     <template v-for="g in recordDays" :key="g.date">
      <div class="record-day"><strong>{{g.date}}</strong><span>{{weekdayName(g.date)}}</span><small>{{number(g.messages)}} 次 · {{g.items.length}} 条</small></div>
      <button v-for="a in g.items" :key="a.id" class="record-row" :class="{muted:a.annotation.hidden}" @click="detail=a">
       <span class="record-time">{{timeLabel(a.record.first_activity_at,a.record.timezone)}}</span>
       <span class="record-main"><strong>{{title(a)}}<EyeOff v-if="a.annotation.hidden" :size="13"/></strong><small>{{summary(a)}}</small><span class="tags"><span v-for="t in topicTags(a)" :key="t">{{t}}</span></span></span>
       <span class="record-meta"><span><i class="source-dot" :style="{background:sourceColor(a.record.source)}"></i>{{sourceName(a.record)}}</span><small>{{countLabel(a.record)}}</small></span><ChevronRight :size="16"/>
      </button>
     </template>
     <button v-if="ranged.length>shownRecords" class="secondary list-more" @click="shownRecords+=150">再显示 150 条（共 {{number(ranged.length)}} 条）</button>
    </section>
    <ArchivesPage v-else class="workspace" :conversations="conversations" :range-label="label" @chat="openChat"/>
   </template>
   <footer class="page-footer"><span>AICalendar</span><span>记录来自你的整理 · {{dateTime(imports[0]?.created_at)}}</span></footer>
  </main>
  <ActivityDetail v-if="detail" ref="detailRef" :activity="detail" :related="related" @close="detail=null" @saved="saved" @open="detail=$event" @chat="openChat"/>
  <div v-if="chat" class="modal-backdrop chat-backdrop" @click.self="chat=null"><section class="modal chat-modal" role="dialog" aria-modal="true" aria-label="完整聊天存档"><div class="modal-header slim"><h2>完整对话</h2><button @click="chat=null" aria-label="关闭聊天存档"><X :size="21"/></button></div><div class="modal-body"><ChatReader :target="chat"/></div></section></div>
  <ImportDialog v-if="importOpen" ref="importRef" @close="importOpen=false" @imported="imported"/>
 </div>
</template>
