<script setup lang="ts">
import {ref,watch} from 'vue'
import {MessageCircle,ChevronRight,Archive,Search,X,LoaderCircle,Package} from 'lucide-vue-next'
import {base} from './api'
import type {ChatTarget,Snapshot,Conversation} from './types'
import {sourceColor,number,dateTime} from './present'

const props=defineProps<{conversations:Conversation[];rangeLabel:string}>()
const emit=defineEmits<{chat:[ChatTarget]}>()
const tab=ref<'conversations'|'snapshots'>('conversations'),shown=ref(100)
const snapshots=ref<Snapshot[]>([]),query=ref(''),offset=ref(0),more=ref(false),busy=ref(false),error=ref(''),started=ref(false)
watch(()=>props.conversations,()=>shown.value=100)
async function load(reset=true){
 busy.value=true;error.value=''
 try{
  if(reset){offset.value=0;snapshots.value=[]}
  const r=await fetch(`${base}api/v1/archives?`+new URLSearchParams({q:query.value.trim(),offset:String(offset.value),limit:'100'}))
  const data=await r.json();if(!r.ok)throw Error(data.error||'存档读取失败')
  snapshots.value.push(...data.items);offset.value=data.next_offset;more.value=data.items.length===100
 }catch(e){error.value=(e as Error).message}finally{busy.value=false;started.value=true}
}
function showSnapshots(){tab.value='snapshots';if(!started.value)load()}
let timer:ReturnType<typeof setTimeout>|undefined
watch(query,()=>{clearTimeout(timer);timer=setTimeout(()=>load(),300)})
const span=(c:Conversation)=>c.first===c.last?c.first:`${c.first} – ${c.last}`
</script>

<template>
 <section class="archives-page">
  <div class="tabs"><button :class="{active:tab==='conversations'}" @click="tab='conversations'">会话<small>{{number(conversations.length)}}</small></button><button :class="{active:tab==='snapshots'}" @click="showSnapshots">全部存档快照</button></div>
  <template v-if="tab==='conversations'">
   <p class="section-note">{{rangeLabel}}内有记录的会话，按最近交流排序；搜索和来源、标签筛选同样生效。</p>
   <div v-if="!conversations.length" class="empty-state"><Archive :size="34"/><h3>这个范围内没有会话</h3><p>换个时间范围或调整筛选条件。</p></div>
   <button v-for="c in conversations.slice(0,shown)" :key="c.key" class="list-card" @click="emit('chat',{source:c.source,conversationId:c.conversationId})">
    <i class="source-bar" :style="{background:sourceColor(c.source)}"></i>
    <span class="list-main"><strong>{{c.title}}</strong><small>{{c.sourceName}} · {{span(c)}}{{c.days>1?` · ${c.days} 天`:''}}</small></span>
    <span class="list-count">{{c.unknown&&!c.messages?'?':number(c.messages)}}<small>次</small></span><ChevronRight :size="17"/>
   </button>
   <button v-if="conversations.length>shown" class="secondary list-more" @click="shown+=100">再显示 100 个（共 {{number(conversations.length)}} 个）</button>
  </template>
  <template v-else>
   <p class="section-note">按整理时间列出服务里保存的全部快照，包括原始导出包；可按标题或来源标签搜索。</p>
   <div class="search wide"><Search :size="16"/><input v-model="query" placeholder="搜索存档标题或来源标签" aria-label="搜索存档"/><button v-if="query" @click="query=''" aria-label="清除搜索"><X :size="15"/></button></div>
   <div v-if="error" class="modal-error" role="alert">{{error}}</div>
   <div v-if="started&&!snapshots.length&&!busy" class="empty-state"><Archive :size="34"/><h3>没有匹配的存档</h3></div>
   <button v-for="s in snapshots" :key="s.id" class="list-card" @click="emit('chat',{source:s.manifest.source,conversationId:s.manifest.conversation_id,snapshotId:s.id})">
    <component :is="s.manifest.message_count?MessageCircle:Package" :size="19" class="list-icon" :style="{color:sourceColor(s.manifest.source)}"/>
    <span class="list-main"><strong>{{s.manifest.title||'AI 会话'}}</strong><small>{{s.manifest.source_label||s.manifest.source}} · {{s.manifest.message_count?number(s.manifest.message_count)+' 条消息':'原始导出包'}} · {{(s.manifest.bytes/1048576).toFixed(1)}} MB · 整理于 {{dateTime(s.manifest.captured_at)}}</small></span><ChevronRight :size="17"/>
   </button>
   <button v-if="more&&!busy" class="secondary list-more" @click="load(false)">加载更多存档</button>
   <div v-if="busy" class="loading small"><LoaderCircle class="spin" :size="20"/>正在读取存档…</div>
  </template>
 </section>
</template>
