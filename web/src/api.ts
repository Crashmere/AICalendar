import type {Activity} from './types'

export const base=import.meta.env.BASE_URL

export async function api(path:string,options:RequestInit={}){
 const r=await fetch(base+'api/v1/'+path,{...options,headers:{'Content-Type':'application/json',...options.headers}})
 if(r.status===401)throw Error('设备登录已失效，请从服务器门户重新登录。')
 const data=await r.json()
 if(!r.ok){const reasons=data.items?.filter((x:any)=>x.reason).map((x:any)=>x.reason).slice(0,3).join('；');throw Error(data.error||reasons||'请求失败，请稍后重试')}
 return data
}

export async function allActivities(){
 const all:Activity[]=[]
 for(let offset=0;;){
  const page=await api(`activities?include_hidden=true&limit=5000&offset=${offset}`)
  all.push(...page.items);offset+=page.items.length
  if(offset>=page.total||!page.items.length)return all
 }
}
