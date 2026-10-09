export type RangeKind='month'|'quarter'|'year'|'custom'|'all'
export interface Range {kind:RangeKind;from:string;to:string}

const DAY=86400000
export const pad=(n:number)=>String(n).padStart(2,'0')
export const toDate=(s:string)=>{const [y,m,d]=s.split('-').map(Number);return new Date(Date.UTC(y,m-1,d))}
export const iso=(d:Date)=>d.toISOString().slice(0,10)
export const addDays=(s:string,n:number)=>iso(new Date(toDate(s).getTime()+n*DAY))
export const diffDays=(a:string,b:string)=>Math.round((toDate(b).getTime()-toDate(a).getTime())/DAY)
export const weekday=(s:string)=>(toDate(s).getUTCDay()+6)%7
export const validDate=(s:string)=>/^\d{4}-\d{2}-\d{2}$/.test(s)&&iso(toDate(s))===s

export function monthRange(y:number,m:number):Range{const d=new Date(Date.UTC(y,m-1,1));y=d.getUTCFullYear();m=d.getUTCMonth()+1;return{kind:'month',from:`${y}-${pad(m)}-01`,to:iso(new Date(Date.UTC(y,m,0)))}}
export function quarterRange(y:number,q:number):Range{const d=new Date(Date.UTC(y,(q-1)*3,1));y=d.getUTCFullYear();const m=d.getUTCMonth()+1;return{kind:'quarter',from:`${y}-${pad(m)}-01`,to:iso(new Date(Date.UTC(y,m+2,0)))}}
export const yearRange=(y:number):Range=>({kind:'year',from:`${y}-01-01`,to:`${y}-12-31`})
export const customRange=(a:string,b:string):Range=>({kind:'custom',from:a<b?a:b,to:a<b?b:a})

export function shiftRange(r:Range,n:number):Range{
 const [y,m]=r.from.split('-').map(Number)
 if(r.kind==='month')return monthRange(y,m+n)
 if(r.kind==='quarter')return quarterRange(y,Math.floor((m-1)/3)+1+n)
 if(r.kind==='year')return yearRange(y+n)
 if(r.kind==='custom'){const len=diffDays(r.from,r.to)+1;return customRange(addDays(r.from,n*len),addDays(r.to,n*len))}
 return r
}

export function encodeRange(r:Range){
 if(r.kind==='all')return 'all'
 if(r.kind==='year')return r.from.slice(0,4)
 if(r.kind==='month')return r.from.slice(0,7)
 if(r.kind==='quarter')return `${r.from.slice(0,4)}-Q${Math.floor((Number(r.from.slice(5,7))-1)/3)+1}`
 return `${r.from}..${r.to}`
}
export function decodeRange(s:string,all:Range):Range|null{
 let m
 if(s==='all')return all
 if((m=/^(\d{4})$/.exec(s)))return yearRange(+m[1])
 if((m=/^(\d{4})-(\d{2})$/.exec(s))&&+m[2]>=1&&+m[2]<=12)return monthRange(+m[1],+m[2])
 if((m=/^(\d{4})-Q([1-4])$/.exec(s)))return quarterRange(+m[1],+m[2])
 if((m=/^(.+)\.\.(.+)$/.exec(s))&&validDate(m[1])&&validDate(m[2]))return customRange(m[1],m[2])
 return null
}

const short=(s:string)=>`${+s.slice(5,7)}/${+s.slice(8,10)}`
export function rangeLabel(r:Range,today:string){
 if(r.kind==='all')return '全部时间'
 if(r.kind==='year')return `${r.from.slice(0,4)} 年`
 if(r.kind==='month')return `${r.from.slice(0,4)} 年 ${+r.from.slice(5,7)} 月`
 if(r.kind==='quarter')return `${r.from.slice(0,4)} 年第 ${Math.floor((+r.from.slice(5,7)-1)/3)+1} 季度`
 if(r.to===today)return `近 ${diffDays(r.from,r.to)+1} 天`
 const sameYear=r.from.slice(0,4)===r.to.slice(0,4)
 return sameYear?`${r.from.slice(0,4)} 年 ${short(r.from)} – ${short(r.to)}`:`${r.from.replaceAll('-','/')} – ${r.to.replaceAll('-','/')}`
}
export const dayLabel=(s:string)=>`${+s.slice(5,7)} 月 ${+s.slice(8,10)} 日`
export const weekdayName=(s:string)=>['星期一','星期二','星期三','星期四','星期五','星期六','星期日'][weekday(s)]
export const inRange=(d:string,r:Range)=>d>=r.from&&d<=r.to
