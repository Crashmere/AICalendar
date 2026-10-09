<script setup lang="ts">
import {computed} from 'vue'
import {type Range,addDays,diffDays,weekday,monthRange,inRange} from './dates'
import {sourceColor,number,title} from './present'
import type {DaySummary} from './types'

const props=defineProps<{range:Range;days:Map<string,DaySummary>;selected:string;today:string;level:(d?:DaySummary)=>number}>()
const emit=defineEmits<{select:[string];range:[Range]}>()

const compact=computed(()=>diffDays(props.range.from,props.range.to)<=62)
const cells=computed(()=>{
 const start=addDays(props.range.from,-weekday(props.range.from)),end=addDays(props.range.to,6-weekday(props.range.to))
 const out:string[]=[];for(let d=start;d<=end;d=addDays(d,1))out.push(d)
 return out
})
const years=computed(()=>{
 const out=[]
 for(let y=+props.range.to.slice(0,4);y>=+props.range.from.slice(0,4);y--){
  const months=[]
  for(let m=1;m<=12;m++){
   const r=monthRange(y,m)
   if(r.to<props.range.from||r.from>props.range.to)continue
   const days:string[]=[];for(let d=r.from;d<=r.to;d=addDays(d,1))days.push(d)
   months.push({key:r.from.slice(0,7),label:`${m} 月`,range:r,lead:weekday(r.from),days,messages:days.reduce((n,d)=>n+(inRange(d,props.range)?props.days.get(d)?.messages||0:0),0)})
  }
  let messages=0,active=0
  for(const [d,s] of props.days)if(d.startsWith(String(y))&&inRange(d,props.range)){messages+=s.messages;active++}
  out.push({year:y,months,messages,active})
 }
 return out
})
function dayTitle(d:string){const s=props.days.get(d);return s?`${d} · ${s.unknown&&!s.messages?'次数未知':number(s.messages)+' 次'} · ${s.items.length} 条记录`:`${d} · 没有记录`}
</script>

<template>
 <section class="calendar">
  <template v-if="compact">
   <div class="weekdays"><span v-for="d in ['一','二','三','四','五','六','日']" :key="d">{{d}}</span></div>
   <div class="calendar-grid">
    <button v-for="date in cells" :key="date" class="day" :class="[`heat-${level(days.get(date))}`,{selected:date===selected,today:date===today,out:!inRange(date,range),has:days.has(date)}]" :disabled="!inRange(date,range)" @click="emit('select',date)" :aria-label="dayTitle(date)" :title="dayTitle(date)">
     <span class="day-top"><span class="day-number">{{+date.slice(8)}}</span><small v-if="date.endsWith('-01')||date===cells[0]" class="day-month">{{+date.slice(5,7)}} 月</small></span>
     <template v-if="days.has(date)&&inRange(date,range)">
      <span class="day-count">{{days.get(date)!.unknown&&!days.get(date)!.messages?'?':number(days.get(date)!.messages)}}<small>次</small></span>
      <span class="day-title">{{title(days.get(date)!.top)}}</span>
      <span class="day-sources"><i v-for="s in days.get(date)!.sources.slice(0,5)" :key="s" :style="{background:sourceColor(s)}"></i><small v-if="days.get(date)!.items.length>1">{{days.get(date)!.items.length}} 条</small></span>
     </template>
    </button>
   </div>
  </template>
  <div v-else class="year-list">
   <section v-for="y in years" :key="y.year" class="year-block">
    <header><h3>{{y.year}} 年</h3><span>{{number(y.messages)}} 次交流 · {{y.active}} 个活跃日</span></header>
    <div class="month-cards">
     <div v-for="m in y.months" :key="m.key" class="mini-month">
      <button class="mini-title" @click="emit('range',m.range)" :title="`查看 ${y.year} 年 ${m.label}`"><strong>{{m.label}}</strong><span v-if="m.messages">{{number(m.messages)}}</span></button>
      <div class="mini-grid"><i v-for="n in m.lead" :key="'b'+n" class="blank"></i><button v-for="d in m.days" :key="d" :class="[`heat-${inRange(d,range)?level(days.get(d)):0}`,{selected:d===selected,today:d===today,out:!inRange(d,range)}]" :disabled="!inRange(d,range)" :title="dayTitle(d)" :aria-label="dayTitle(d)" @click="emit('select',d)" @dblclick="emit('range',m.range)"></button></div>
     </div>
    </div>
   </section>
  </div>
  <div class="calendar-footer"><span class="legend">少<i v-for="n in 5" :key="n" :class="`heat-${n-1}`"></i>多</span><span>{{compact?'点击日期查看当天的探索':'点击日期查看当天，点月份标题进入月历'}}</span></div>
 </section>
</template>
