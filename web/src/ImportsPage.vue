<script setup lang="ts">
import {computed} from 'vue'
import {Download,History,Upload,ChevronRight} from 'lucide-vue-next'
import {base} from './api'
import type {Activity,ImportResult,Coverage} from './types'
import {sourceColor,number,dateTime} from './present'
import {diffDays} from './dates'

const props=defineProps<{imports:ImportResult[];activities:Activity[];names:Map<string,string>;today:string}>()
const emit=defineEmits<{import:[]}>()
const statusLabel:Record<string,string>={complete:'范围已整理',partial:'部分资料',unknown:'完整性未知'}
const day=new Intl.DateTimeFormat('en-CA',{timeZone:'Asia/Shanghai'})

const sources=computed(()=>{
 const map=new Map<string,{source:string;records:number;messages:number;first:string;last:string;coverage:Coverage[]}>()
 const get=(s:string)=>{let v=map.get(s);if(!v){v={source:s,records:0,messages:0,first:'9999',last:'0000',coverage:[]};map.set(s,v)}return v}
 for(const a of props.activities){const v=get(a.record.source);v.records++;v.messages+=a.record.user_message_count||0;if(a.record.activity_date<v.first)v.first=a.record.activity_date;if(a.record.activity_date>v.last)v.last=a.record.activity_date}
 const seen=new Set<string>()
 for(const b of props.imports)for(const c of b.coverage||[]){const k=[c.source,c.from,c.to,c.status].join('|');if(!seen.has(k)){seen.add(k);get(c.source).coverage.push(c)}}
 return [...map.values()].sort((a,b)=>b.records-a.records)
})
const bounds=computed(()=>{
 let from=props.today
 for(const s of sources.value){if(s.records&&s.first<from)from=s.first;for(const c of s.coverage)if(c.from<from)from=c.from}
 return {from,len:Math.max(1,diffDays(from,props.today)+1)}
})
const pos=(a:string,b:string)=>{const l=Math.max(0,diffDays(bounds.value.from,a)),w=Math.max(.6,(diffDays(a,b)+1)/bounds.value.len*100);return {left:l/bounds.value.len*100+'%',width:Math.min(w,100)+'%'}}
const years=computed(()=>{const out=[];for(let y=+bounds.value.from.slice(0,4)+1;y<=+props.today.slice(0,4);y++)out.push({y,left:diffDays(bounds.value.from,`${y}-01-01`)/bounds.value.len*100+'%'});return out})
const groups=computed(()=>{
 const map=new Map<string,ImportResult[]>()
 for(const b of props.imports){const d=b.created_at?day.format(new Date(b.created_at)):'未知日期';map.set(d,[...(map.get(d)||[]),b])}
 return [...map].map(([date,items])=>({date,items,inserted:items.reduce((n,b)=>n+b.inserted,0),updated:items.reduce((n,b)=>n+b.updated,0),unchanged:items.reduce((n,b)=>n+b.unchanged,0)}))
})
</script>

<template>
 <section class="imports-page">
  <div class="section-header"><h2>来源覆盖</h2><a class="secondary" :href="base+'api/v1/export'"><Download :size="15"/>导出日历摘要</a></div>
  <div v-if="!imports.length" class="empty-state"><History :size="38"/><h3>从第一次整理开始</h3><p>使用导入 skill，或上传已经整理好的 JSON 文件。</p><button class="primary" @click="emit('import')"><Upload :size="16"/>导入记录</button></div>
  <div v-else class="coverage-timeline">
   <div class="timeline-axis"><span>{{bounds.from.slice(0,7)}}</span><i v-for="y in years" :key="y.y" :style="{left:y.left}">{{y.y}}</i><span>今天</span></div>
   <div v-for="s in sources" :key="s.source" class="timeline-row">
    <div class="timeline-name"><i class="source-dot" :style="{background:sourceColor(s.source)}"></i><strong>{{names.get(s.source)||s.source}}</strong><small>{{number(s.records)}} 条记录 · {{number(s.messages)}} 次交流<template v-if="s.records"> · {{s.first}} – {{s.last}}</template></small></div>
    <div class="timeline-track">
     <span v-for="(c,i) in s.coverage" :key="i" class="coverage-bar" :class="c.status" :style="pos(c.from,c.to)" :title="`${c.from} – ${c.to} · ${statusLabel[c.status]}`"></span>
     <span v-if="s.records" class="record-span" :style="{...pos(s.first,s.last),background:sourceColor(s.source)}"></span>
    </div>
    <details v-if="s.coverage.length" class="coverage-notes"><summary>{{s.coverage.length}} 个覆盖声明</summary><p v-for="(c,i) in s.coverage" :key="i"><strong>{{c.from}} – {{c.to}}</strong><span :class="c.status">{{statusLabel[c.status]}}</span>{{c.note}}</p></details>
    <p v-else class="field-hint">尚未声明覆盖范围：没有记录的日期不代表没有使用。</p>
   </div>
   <p class="legend-line"><span><i class="coverage-bar complete"></i>范围已整理</span><span><i class="coverage-bar partial"></i>部分资料</span><span><i class="coverage-bar unknown"></i>完整性未知</span><span><i class="record-span sample"></i>实际有记录的日期跨度</span></p>
  </div>
  <template v-if="imports.length">
   <div class="section-header spaced"><h2>导入批次</h2><span class="section-note">共 {{imports.length}} 批</span></div>
   <details v-for="g in groups" :key="g.date" class="import-group">
    <summary><ChevronRight :size="15" class="chevron"/><strong>{{g.date}}</strong><span>{{g.items.length}} 批 · 新增 {{number(g.inserted)}} · 更新 {{number(g.updated)}} · 未变化 {{number(g.unchanged)}}</span></summary>
    <div v-for="b in g.items" :key="b.id" class="import-row"><span>{{dateTime(b.created_at).split(' ').pop()}}</span><span>新增 {{b.inserted}} · 更新 {{b.updated}} · 未变化 {{b.unchanged}}</span><small>{{b.coverage.length?b.coverage.map(c=>names.get(c.source)||c.source).join('、')+' 覆盖声明':''}}</small></div>
   </details>
  </template>
 </section>
</template>
