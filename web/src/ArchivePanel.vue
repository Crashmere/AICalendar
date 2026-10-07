<script setup lang="ts">
import {ref,watch} from 'vue'
import {MessageCircle,Download,ChevronLeft,LoaderCircle,Archive,ChevronRight} from 'lucide-vue-next'
const props=defineProps<{source?:string;conversationId?:string}>()
interface Snapshot{id:string;manifest:{source:string;source_label?:string;conversation_id:string;title:string;captured_at:string;coverage:string;note:string;bytes:number;message_count:number;source_count:number};committed_at:string}
interface Message{id:string;role:string;at:string|null;content:string;part:number;parts:number;attributes?:Record<string,unknown>}
const base=import.meta.env.BASE_URL+'api/v1/archives'
const snapshots=ref<Snapshot[]>([]),selected=ref<Snapshot|null>(null),messages=ref<Message[]>([])
const label=ref('')
const busy=ref(false),error=ref(''),offset=ref(0),after=ref(-1),more=ref(false),moreSnapshots=ref(false)
async function request(url:string){const response=await fetch(url);const data=await response.json();if(!response.ok)throw Error(data.error||'存档读取失败');return data}
async function load(reset=true){busy.value=true;error.value='';try{if(reset){offset.value=0;snapshots.value=[];selected.value=null;messages.value=[]}const q=new URLSearchParams({source:props.source||'',conversation_id:props.conversationId||'',label:label.value,offset:String(offset.value),limit:'50'});const result=await request(base+'?'+q);snapshots.value.push(...result.items);offset.value=result.next_offset;moreSnapshots.value=result.items.length===50}catch(e){error.value=(e as Error).message}finally{busy.value=false}}
async function open(snapshot:Snapshot){selected.value=snapshot;messages.value=[];after.value=-1;await loadMessages()}
async function loadMessages(){if(!selected.value)return;const id=selected.value.id;busy.value=true;error.value='';try{const result=await request(`${base}/${id}/messages?after=${after.value}&limit=20`);if(selected.value?.id!==id)return;messages.value.push(...result.items);after.value=result.next_after;more.value=result.has_more}catch(e){error.value=(e as Error).message}finally{busy.value=false}}
function date(value:string|null){return value?new Date(value).toLocaleString('zh-CN'):'时间未知'}
function role(value:string){return({user:'你',assistant:'AI',tool:'工具',system:'系统',developer:'指令',unknown:'其他'} as Record<string,string>)[value]||value}
watch(()=>[props.source,props.conversationId],()=>load(),{immediate:true})
</script>
<template>
 <section class="archive-panel">
  <div v-if="error" class="modal-error" role="alert">{{error}}</div>
  <template v-if="selected">
   <div class="archive-heading"><button class="text-button" @click="selected=null" :disabled="busy"><ChevronLeft :size="16"/>返回存档</button><a class="secondary" :href="`${base}/${selected.id}/download`"><Download :size="16"/>下载完整存档</a></div>
   <h2>{{selected.manifest.title||'完整对话'}}</h2><p class="archive-description">{{selected.manifest.source_label||selected.manifest.source}} · {{selected.manifest.message_count}} 条消息 · 整理于 {{date(selected.manifest.captured_at)}}</p>
   <div v-if="selected.manifest.source_label" class="tags"><span>{{selected.manifest.source_label}}</span></div>
   <p v-if="selected.manifest.coverage!=='complete'" class="archive-note">来源完整性：{{selected.manifest.coverage==='partial'?'部分可恢复':'尚未确认'}}。{{selected.manifest.note}}</p>
   <div v-if="!messages.length&&!busy" class="empty-state compact"><p>此存档保留了原始资料，暂无可展示的统一消息。</p></div>
   <article v-for="message in messages" :key="message.id+':'+message.part" class="chat-message" :class="{'from-user':message.role==='user'}"><div class="chat-meta"><strong>{{role(message.role)}}</strong><span>{{date(message.at)}}</span><span v-if="message.parts>1">第 {{message.part+1}} / {{message.parts}} 段</span></div><pre>{{message.content||'（空消息，原始结构保存在存档中）'}}</pre><details v-if="message.attributes&&Object.keys(message.attributes).length"><summary>来源信息</summary><pre>{{JSON.stringify(message.attributes,null,2)}}</pre></details></article>
   <button v-if="more&&!busy" class="secondary archive-more" @click="loadMessages">继续加载消息<ChevronRight :size="15"/></button>
  </template>
  <template v-else>
   <h2>{{conversationId?'这段会话的完整存档':'完整聊天存档'}}</h2><p class="archive-description">逐条消息与原始资料一起保留。每次内容有变化时保存新的快照，旧版本仍可下载。</p>
   <form v-if="!conversationId" class="archive-filter" @submit.prevent="load()"><input v-model="label" placeholder="筛选来源标签…" aria-label="存档来源标签"/><button class="secondary" :disabled="busy">筛选</button></form>
   <div v-if="!snapshots.length&&!busy" class="empty-state"><Archive :size="34"/><h3>还没有聊天存档</h3><p>下次使用导入 skill 时，可以连同完整聊天一起保存。</p></div>
   <button v-for="snapshot in snapshots" :key="snapshot.id" class="archive-card" @click="open(snapshot)" :disabled="busy"><MessageCircle :size="21"/><span><strong>{{snapshot.manifest.title||'AI 会话'}}</strong><small>{{snapshot.manifest.source_label||snapshot.manifest.source}} · {{snapshot.manifest.message_count}} 条消息 · {{date(snapshot.manifest.captured_at)}}</small></span><ChevronRight :size="17"/></button>
   <button v-if="moreSnapshots&&!busy" class="secondary archive-more" @click="load(false)">加载更多存档</button>
  </template>
  <div v-if="busy" class="loading"><LoaderCircle class="spin" :size="20"/>正在读取存档…</div>
 </section>
</template>
<style scoped>
.archive-filter{display:flex;gap:10px;margin:20px 0}.archive-filter input{min-width:0;flex:1;padding:10px;border:1px solid #dbe5df;border-radius:8px}.archive-panel{min-width:0;padding:24px;background:var(--surface,#fff);border:1px solid var(--border,#e0e6e3);border-radius:14px}.archive-panel h2{font-size:21px;margin:8px 0 12px;overflow-wrap:anywhere}.archive-description,.archive-note{font-size:13px;line-height:1.7;color:#6d7e75;overflow-wrap:anywhere}.archive-note{background:#f6f7ed;padding:12px;border-radius:8px}.archive-heading{display:flex;justify-content:space-between;gap:10px;align-items:center;flex-wrap:wrap;margin-bottom:20px}.archive-card{width:100%;display:flex;gap:14px;align-items:center;text-align:left;border:1px solid #e0e6e3;border-radius:10px;padding:18px;margin:12px 0;background:white;color:inherit}.archive-card>span{flex:1;min-width:0}.archive-card strong{display:block;font-size:14px;overflow-wrap:anywhere}.archive-card small{display:block;font-size:11px;color:#789084;margin-top:9px;line-height:1.6}.archive-card:hover{background:#f6faf7}.chat-message{padding:18px;border:1px solid #e2e9e5;border-radius:10px;margin:14px 0;background:#fafcfb}.chat-message.from-user{background:#edf5f0}.chat-meta{display:flex;gap:12px;flex-wrap:wrap;align-items:center;color:#7d8e85;font-size:11px;margin-bottom:12px}.chat-meta strong{font-size:13px;color:#315e4e}.chat-message pre{max-height:28rem;overflow:auto;white-space:pre-wrap;overflow-wrap:anywhere;word-break:break-word;font:13px/1.75 ui-monospace,SFMono-Regular,Consolas,monospace;margin:0;color:#293c32}.chat-message details{margin-top:14px;color:#73857b;font-size:11px}.chat-message details pre{margin-top:10px;font-size:11px}.archive-more{margin:18px auto;display:flex}@media(max-width:600px){.archive-panel{padding:16px}.archive-card{padding:14px;gap:10px}.chat-message{padding:12px}.chat-message pre{font-size:12px}.archive-panel h2{font-size:18px}}
</style>
