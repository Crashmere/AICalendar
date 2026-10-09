<script setup lang="ts">
import {ref,computed,onMounted,onBeforeUnmount} from 'vue'
import {Tag,ChevronDown,Check,Search} from 'lucide-vue-next'

const props=defineProps<{options:{tag:string;count:number}[];modelValue:string[]}>()
const emit=defineEmits<{'update:modelValue':[string[]]}>()
const open=ref(false),query=ref(''),root=ref<HTMLElement>()
const visible=computed(()=>{const q=query.value.trim().toLowerCase();return (q?props.options.filter(o=>o.tag.toLowerCase().includes(q)):props.options).slice(0,80)})
function toggle(tag:string){const s=new Set(props.modelValue);s.has(tag)?s.delete(tag):s.add(tag);emit('update:modelValue',[...s])}
function outside(e:MouseEvent){if(open.value&&root.value&&!root.value.contains(e.target as Node))open.value=false}
onMounted(()=>document.addEventListener('mousedown',outside))
onBeforeUnmount(()=>document.removeEventListener('mousedown',outside))
defineExpose({close:()=>{const was=open.value;open.value=false;return was}})
</script>

<template>
 <div ref="root" class="tag-picker">
  <button class="filter-button" :class="{on:modelValue.length}" @click="open=!open"><Tag :size="15"/>{{modelValue.length?`标签 · ${modelValue.length}`:'标签'}}<ChevronDown :size="14"/></button>
  <div v-if="open" class="tag-popover" role="dialog" aria-label="筛选标签">
   <div class="search small"><Search :size="15"/><input v-model="query" placeholder="搜索标签" aria-label="搜索标签"/></div>
   <p class="field-hint">显示包含任一所选标签的记录，数字为当前范围内的记录数。</p>
   <div class="tag-options">
    <button v-for="o in visible" :key="o.tag" :class="{active:modelValue.includes(o.tag)}" @click="toggle(o.tag)"><Check v-if="modelValue.includes(o.tag)" :size="13"/>{{o.tag}}<small>{{o.count}}</small></button>
    <p v-if="!visible.length" class="field-hint">没有匹配的标签</p>
   </div>
   <div class="popover-footer"><button class="text-button" :disabled="!modelValue.length" @click="emit('update:modelValue',[])">清除选择</button><button class="secondary" @click="open=false">完成</button></div>
  </div>
 </div>
</template>
