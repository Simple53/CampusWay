import {t} from './i18n.js';
export const esc=value=>String(value??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
export const icon=name=>`<svg class="icon" aria-hidden="true"><use href="/icons.svg#${name}"></use></svg>`;
export const button=(label,action,kind='',extra='')=>`<button type="button" class="${kind}" data-action="${action}" ${extra}>${t(label)}</button>`;
export const status=value=>`<span class="status ${value}"><i aria-hidden="true"></i>${t(value||'disconnected')}</span>`;
export function field(key,name,value='',type='text',extra=''){return `<label class="field"><span>${t(key)}</span><input name="${name}" type="${type}" value="${esc(value)}" ${extra}></label>`;}
export function select(key,name,value,options){return `<label class="field"><span>${t(key)}</span><select name="${name}">${options.map(([v,label])=>`<option value="${v}" ${v===value?'selected':''}>${t(label)}</option>`).join('')}</select></label>`;}
export function download(text,name){const url=URL.createObjectURL(new Blob([text],{type:'application/json'}));const a=document.createElement('a');a.href=url;a.download=name;a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);}
