<script setup lang="ts">
import {ref,computed,watch,onMounted,onBeforeUnmount} from 'vue'
import {CalendarRange,ChevronDown,ChevronLeft,ChevronRight,Check} from 'lucide-vue-next'
import {type Range,addDays,monthRange,quarterRange,yearRange,customRange,rangeLabel,encodeRange} from './dates'

const props=defineProps<{modelValue:Range;today:string;all:Range;activeMonths:Set<string>}>()
const emit=defineEmits<{'update:modelValue':[Range]}>()
const open=ref(false),root=ref<HTMLElement>(),year=ref(+props.today.slice(0,4))
const from=ref(props.modelValue.from),to=ref(props.modelValue.to)
const label=computed(()=>rangeLabel(props.modelValue,props.today))
const ty=+props.today.slice(0,4),tm=+props.today.slice(5,7)
const presets=computed(()=>[
 {label:'本月',range:monthRange(ty,tm)},
 {label:'上个月',range:monthRange(ty,tm-1)},
 {label:'近 7 天',range:customRange(addDays(props.today,-6),props.today)},
 {label:'近 30 天',range:customRange(addDays(props.today,-29),props.today)},
 {label:'近 90 天',range:customRange(addDays(props.today,-89),props.today)},
 {label:'本季度',range:quarterRange(ty,Math.floor((tm-1)/3)+1)},
 {label:'今年',range:yearRange(ty)},
 {label:'去年',range:yearRange(ty-1)},
 {label:'全部时间',range:props.all},
])
const minYear=computed(()=>+props.all.from.slice(0,4))
const current=computed(()=>encodeRange(props.modelValue))
function pick(r:Range){emit('update:modelValue',r);open.value=false}
function applyCustom(){if(from.value&&to.value)pick(customRange(from.value,to.value))}
function toggle(){open.value=!open.value;if(open.value){year.value=+props.modelValue.to.slice(0,4);from.value=props.modelValue.from;to.value=props.modelValue.to}}
function outside(e:MouseEvent){if(open.value&&root.value&&!root.value.contains(e.target as Node))open.value=false}
watch(()=>props.modelValue,()=>{from.value=props.modelValue.from;to.value=props.modelValue.to})
onMounted(()=>document.addEventListener('mousedown',outside))
onBeforeUnmount(()=>document.removeEventListener('mousedown',outside))
defineExpose({close:()=>{const was=open.value;open.value=false;return was}})
</script>

<template>
 <div ref="root" class="range-picker">
  <button class="range-button" :class="{open}" @click="toggle" aria-label="选择时间范围"><CalendarRange :size="17"/><span>{{label}}</span><ChevronDown :size="15"/></button>
  <div v-if="open" class="range-popover" role="dialog" aria-label="时间范围">
   <div class="range-presets">
    <button v-for="p in presets" :key="p.label" :class="{active:encodeRange(p.range)===current}" @click="pick(p.range)">{{p.label}}<Check v-if="encodeRange(p.range)===current" :size="14"/></button>
   </div>
   <div class="range-calendar">
    <div class="range-year"><button @click="year--" :disabled="year<=minYear" aria-label="上一年"><ChevronLeft :size="16"/></button><button class="year-name" :class="{active:current===String(year)}" @click="pick(yearRange(year))">{{year}} 年全年</button><button @click="year++" :disabled="year>=ty" aria-label="下一年"><ChevronRight :size="16"/></button></div>
    <div class="range-quarters"><button v-for="q in 4" :key="q" :class="{active:current===`${year}-Q${q}`}" @click="pick(quarterRange(year,q))">Q{{q}}</button></div>
    <div class="range-months"><button v-for="m in 12" :key="m" :class="{active:current===monthRange(year,m).from.slice(0,7),has:activeMonths.has(monthRange(year,m).from.slice(0,7))}" @click="pick(monthRange(year,m))">{{m}} 月</button></div>
    <div class="range-custom"><span>自定义</span><input type="date" v-model="from" aria-label="开始日期"/><span>至</span><input type="date" v-model="to" aria-label="结束日期"/><button class="primary" :disabled="!from||!to" @click="applyCustom">应用</button></div>
   </div>
  </div>
 </div>
</template>
