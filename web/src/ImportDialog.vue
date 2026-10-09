<script setup lang="ts">
import {ref} from 'vue'
import {X,Sparkles,Upload,ArrowUpRight,Check} from 'lucide-vue-next'
import {api} from './api'
import type {ImportResult} from './types'

const emit=defineEmits<{close:[];imported:[ImportResult]}>()
const body=ref<any>(null),preview=ref<ImportResult|null>(null),fileName=ref(''),busy=ref(false),error=ref('')
async function selectFile(event:Event){
 const file=(event.target as HTMLInputElement).files?.[0];preview.value=null;body.value=null
 if(!file)return
 fileName.value=file.name
 try{if(file.size>1048576)throw Error('单批文件最多 1 MiB，请用 skill 分批导入');body.value=JSON.parse(await file.text());error.value=''}catch(e){error.value=(e as Error).message}
}
async function run(commit=false){
 if(!body.value)return
 busy.value=true;error.value=''
 try{const result=await api(commit?'imports':'imports/preview',{method:'POST',body:JSON.stringify(body.value)});preview.value=result;if(commit)emit('imported',result)}
 catch(e){error.value=(e as Error).message}finally{busy.value=false}
}
function close(){if(!busy.value)emit('close')}
defineExpose({close})
</script>

<template>
 <div class="modal-backdrop" @click.self="close">
  <section class="modal" role="dialog" aria-modal="true" aria-label="导入记录">
   <div class="modal-header"><div><span class="eyebrow-line">BRING YOUR JOURNEY HERE</span><h2>把最近的探索带进来</h2></div><button @click="close" :disabled="busy" aria-label="关闭"><X :size="21"/></button></div>
   <div class="modal-body">
    <div v-if="error" class="modal-error" role="alert">{{error}}</div>
    <div class="skill-tip"><Sparkles :size="18"/><div><strong>也可以直接让 agent 整理</strong><p>“用 ai-calendar-import，导入最近一周的 AI 使用记录。”</p></div></div>
    <label class="file-picker"><Upload :size="26"/><strong>{{fileName||'选择整理好的 JSON 文件'}}</strong><span>规范化记录 · 单批最多 500 条 / 1 MiB</span><input type="file" accept=".json,application/json" @change="selectFile"/></label>
    <div v-if="preview" class="preview-result">
     <h3>{{preview.replay?'这批记录已经导入':'导入预览'}}</h3>
     <div class="preview-counts"><span><strong>{{preview.inserted}}</strong>新增</span><span><strong>{{preview.updated}}</strong>更新</span><span><strong>{{preview.unchanged}}</strong>未变化</span><span :class="{danger:preview.conflicts}"><strong>{{preview.conflicts}}</strong>冲突</span></div>
     <p v-for="(item,i) in preview.items?.filter(x=>x.reason).slice(0,5)" :key="i" class="conflict-line">{{item.source}}：{{item.reason}}</p>
     <p class="field-hint">有冲突时整批不会写入。用户修改过的标题与标签会保留。</p>
    </div>
   </div>
   <div class="modal-footer"><span class="field-hint">先预览，再保存</span>
    <button v-if="!preview" class="primary" :disabled="!body||busy" @click="run()">{{busy?'检查中…':'预览导入'}}<ArrowUpRight :size="16"/></button>
    <button v-else class="primary" :disabled="busy||preview.conflicts>0||preview.replay" @click="run(true)">{{busy?'导入中…':preview.replay?'已导入':'确认导入'}}<Check :size="16"/></button>
   </div>
  </section>
 </div>
</template>
