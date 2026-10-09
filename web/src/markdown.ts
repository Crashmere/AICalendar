import MarkdownIt from 'markdown-it'

// Raw HTML stays escaped and images are shown as text, so archived chats never run markup or fetch remote resources.
const md=new MarkdownIt({html:false,linkify:false,typographer:false,breaks:true})
md.renderer.rules.image=(tokens,i)=>{const t=tokens[i];return `<span class="md-image">[图片] ${md.utils.escapeHtml(t.content||t.attrGet('src')||'')}</span>`}
const defaultLink=md.renderer.rules.link_open||((tokens,i,options,_env,self)=>self.renderToken(tokens,i,options))
md.renderer.rules.link_open=(tokens,i,options,env,self)=>{tokens[i].attrSet('target','_blank');tokens[i].attrSet('rel','noopener noreferrer nofollow');return defaultLink(tokens,i,options,env,self)}

const cache=new Map<string,{text:string;html:string}>()
export function renderMarkdown(key:string,text:string){
 const hit=cache.get(key)
 if(hit?.text===text)return hit.html
 const html=md.render(text)
 if(cache.size>2000)cache.clear()
 cache.set(key,{text,html})
 return html
}
