<script setup lang="ts">
import {ref,computed,watch} from 'vue'
import {X,MessageCircle,Pencil,Check,EyeOff,ArrowLeft} from 'lucide-vue-next'
import {api} from './api'
import type {Activity,ChatTarget} from './types'
import {dayLabel,weekdayName} from './dates'
import {title,summary,topicTags,sourceName,sourceColor,timeLabel,countLabel,qualityCount,qualityTime,qualityTopic,number} from './present'

const props=defineProps<{activity:Activity;related:Activity[]}>()
const emit=defineEmits<{close:[];saved:[Activity];open:[Activity];chat:[ChatTarget]}>()
const editing=ref(false),busy=ref(false),error=ref('')
const editTitle=ref(''),editSummary=ref(''),editTags=ref(''),editHidden=ref(false)
const a=computed(()=>props.activity),r=computed(()=>props.activity.record)
const others=computed(()=>props.related.filter(x=>x.id!==a.value.id).sort((x,y)=>x.record.activity_date.localeCompare(y.record.activity_date)))
const modified=computed(()=>a.value.annotation.title!=null||a.value.annotation.summary!=null||a.value.annotation.tags!=null||a.value.annotation.hidden)
watch(()=>props.activity.id,()=>{editing.value=false;error.value=''})
function edit(){editTitle.value=title(a.value);editSummary.value=summary(a.value);editTags.value=topicTags(a.value).join('，');editHidden.value=a.value.annotation.hidden||false;error.value='';editing.value=true}
async function save(reset=false){
 busy.value=true;error.value=''
 try{
  const annotation=reset?{hidden:false}:{title:editTitle.value,summary:editSummary.value,tags:editTags.value.split(/[,，]/).map(x=>x.trim()).filter(Boolean),hidden:editHidden.value}
  const updated=await api(`activities/${a.value.id}/annotation`,{method:'PATCH',body:JSON.stringify({expected_version:a.value.annotation_version,annotation})})
  emit('saved',updated);editing.value=false
 }catch(e){error.value=(e as Error).message}finally{busy.value=false}
}
function close(){if(busy.value)return;if(editing.value)editing.value=false;else emit('close')}
defineExpose({close})
</script>

<template>
 <div class="modal-backdrop" @click.self="close">
  <section class="modal detail-modal" role="dialog" aria-modal="true" :aria-label="editing?'编辑活动记录':'活动记录详情'">
   <div class="modal-header">
    <div><span class="eyebrow-line"><i class="source-dot" :style="{background:sourceColor(r.source)}"></i>{{sourceName(r)}} · {{dayLabel(r.activity_date)}} {{weekdayName(r.activity_date)}}</span><h2>{{editing?'编辑展示内容':title(a)}}</h2></div>
    <button @click="close" :disabled="busy" aria-label="关闭"><X :size="21"/></button>
   </div>
   <div v-if="!editing" class="modal-body">
    <p class="detail-summary">{{summary(a)||'暂时没有摘要'}}</p>
    <div class="tags large"><span v-for="t in topicTags(a)" :key="t">{{t}}</span></div>
    <dl class="facts">
     <div><dt>交流</dt><dd>{{countLabel(r)}}<small>{{qualityCount[r.quality.count]||r.quality.count}}</small></dd></div>
     <div><dt>时间</dt><dd>{{timeLabel(r.first_activity_at,r.timezone)}}<template v-if="r.last_activity_at&&r.last_activity_at!==r.first_activity_at"> – {{timeLabel(r.last_activity_at,r.timezone)}}</template><small>{{qualityTime[r.quality.time]||r.quality.time}}</small></dd></div>
     <div><dt>整理</dt><dd>{{r.record_state==='partial'?'待补全':'已整理'}}<small>{{qualityTopic[r.quality.topic]||r.quality.topic}}{{r.provenance.producer?' · '+r.provenance.producer:''}}</small></dd></div>
    </dl>
    <p v-if="modified" class="field-hint"><EyeOff v-if="a.annotation.hidden" :size="13"/>{{a.annotation.hidden?'这条记录已隐藏。':''}}展示内容经过你的修改，重新导入不会覆盖。</p>
    <div v-if="others.length" class="related">
     <h3>同一会话的其他日子</h3>
     <button v-for="o in others" :key="o.id" @click="emit('open',o)"><span>{{o.record.activity_date}}</span><strong>{{title(o)}}</strong><small>{{o.record.user_message_count===null?'?':number(o.record.user_message_count)}} 次</small></button>
    </div>
   </div>
   <div v-else class="modal-body">
    <div v-if="error" class="modal-error" role="alert">{{error}}</div>
    <label class="field">主题<input v-model="editTitle" maxlength="160"/></label>
    <label class="field">摘要<textarea v-model="editSummary" rows="5" maxlength="4000"></textarea></label>
    <label class="field">标签<span class="field-hint">用逗号分隔</span><input v-model="editTags"/></label>
    <label class="check-label"><input type="checkbox" v-model="editHidden"/>隐藏这条记录</label>
    <p class="field-hint">这里的修改单独保存，后续导入不会覆盖来源内容。</p>
   </div>
   <div class="modal-footer">
    <template v-if="!editing">
     <button class="secondary" @click="edit"><Pencil :size="15"/>编辑</button>
     <button v-if="r.conversation_id" class="primary" @click="emit('chat',{source:r.source,conversationId:r.conversation_id,date:r.activity_date})"><MessageCircle :size="16"/>查看完整对话</button>
    </template>
    <template v-else>
     <span class="footer-left"><button class="text-button" @click="editing=false" :disabled="busy"><ArrowLeft :size="14"/>返回</button><button v-if="modified" class="text-button" @click="save(true)" :disabled="busy">恢复来源内容</button></span>
     <button class="primary" @click="save()" :disabled="busy||!editTitle.trim()"><Check :size="16"/>{{busy?'保存中…':'保存修改'}}</button>
    </template>
   </div>
  </section>
 </div>
</template>
