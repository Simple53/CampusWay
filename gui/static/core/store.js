export const store={snapshot:{profiles:[],preferences:{accent:'graphite',language:'zh-CN',theme:'system',view:'grid',density:'compact',sort:'default'},state:'disconnected',logs:[]},query:'',busy:false};
export const selected=()=>store.snapshot.profiles.find(p=>p.id===store.snapshot.selected)||store.snapshot.profiles[0];
export const active=()=>store.snapshot.profiles.find(p=>p.id===store.snapshot.active);
export const running=()=>!!store.snapshot.active;
export function primary(profile){const s=store.snapshot;if(profile?.id===s.active)return s.state==='connected'?'disconnect':'stop';return s.active?'switch':'connect';}
