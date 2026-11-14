// This adapter exists only in the local preview server, outside embedded assets.
// Passwords and verification responses are discarded; timers never contact a gateway.
export function createPreviewBridge({delay=900,onChange=()=>{}}={}){
 let serial=0,generation=0,outcome='success';
 const defaults=()=>({profiles:[],preferences:{accent:'graphite',language:'zh-CN',theme:'system',view:'grid',density:'compact',sort:'default'},selected:'',active:'',state:'disconnected',message:'ready',connectedAt:0,challenge:null,logs:[],traffic:null,canRemember:false,preview:true});
 let state=defaults();
 const clone=x=>structuredClone(x),emit=(status,message)=>{state.state=status;state.message=message;state.logs.push({seq:++serial,time:new Date().toISOString(),message});state.logs=state.logs.slice(-100);onChange();};
 const after=(fn)=>{const ticket=generation;setTimeout(()=>{if(ticket===generation)fn();},delay);};
 const profile=id=>{const p=state.profiles.find(p=>p.id===id);if(!p)throw new Error('请选择一条线路');return p;};
 function connected(){state.challenge=null;state.connectedAt=Date.now();emit('connected','connectedEvent');}
 function reset(scenario){generation++;state=defaults();serial=0;outcome=scenario==='failure'?'failure':scenario==='verification'?'verification':'success';if(scenario!=='empty'){
 state.profiles=[{id:'demo-atrust',name:'校园主线路',server:'gateway.demo.invalid',port:443,protocol:'atrust',username:'',remember:false,hasPassword:false,customDomains:'',options:{authType:'auth/psw',socksPort:1080,httpPort:1081}},{id:'demo-easy',name:'备用线路',server:'backup.demo.invalid',port:443,protocol:'easyconnect',username:'',remember:false,hasPassword:false,customDomains:'',options:{socksPort:2080,httpPort:2081}}];state.selected=state.profiles[0].id;
 if(scenario==='connected'){state.active=state.selected;connected();}
 }onChange();}
 const api={
 GetSnapshot:async()=>{const snapshot=clone(state);if(state.state==='connected'){
 // Deterministic synthetic samples are labelled in the UI and never used by production.
 const seconds=Math.max(0,(Date.now()-state.connectedAt)/1000),wave=(Math.floor(seconds)%8)/8;
 snapshot.traffic={uploaded:Math.floor(seconds*24576),downloaded:Math.floor(seconds*393216),uploadRate:24576*(1+wave),downloadRate:393216*(1+wave),sampledAt:Date.now()};
 }return snapshot;},SetLayout:async()=>{},
 SavePreferences:async prefs=>{if(!['zh-CN','en'].includes(prefs.language))throw new Error('Unsupported language');if(!['graphite','ocean','jade','violet','rose','amber'].includes(prefs.accent||'graphite'))throw new Error('Invalid color theme');state.preferences=clone({...prefs,accent:prefs.accent||'graphite'});},
 SelectProfile:async id=>{profile(id);state.selected=id;},
 SaveProfile:async(p,_discardedPassword)=>{if(!p.name?.trim()||!p.server?.trim())throw new Error('请填写连接名称和网关');if(p.id&&p.id===state.active)throw new Error('noEditActive');if(!Number.isInteger(p.port)||p.port<1||p.port>65535)throw new Error('端口范围为 1–65535');if(p.options.socksPort===p.options.httpPort)throw new Error('代理端口不能相同');const saved={...clone(p),id:p.id||crypto.randomUUID(),remember:false,hasPassword:false};const i=state.profiles.findIndex(q=>q.id===saved.id);if(i<0)state.profiles.push(saved);else state.profiles[i]=saved;state.selected=saved.id;return clone(saved);},
 DeleteProfile:async id=>{profile(id);if(id===state.active)throw new Error('请先断开正在运行的线路');state.profiles=state.profiles.filter(p=>p.id!==id);if(state.selected===id)state.selected=state.profiles[0]?.id||'';},
 Connect:async(id,_discardedPassword,remember=false)=>{profile(id);generation++;state.active=id;state.selected=id;state.challenge=null;state.connectedAt=0;state.traffic=null;if(remember||state.profiles.find(p=>p.id===id)?.remember){const p=state.profiles.find(p=>p.id===id);if(p){p.remember=true;p.hasPassword=true;}}emit('connecting','connectingEvent');after(()=>{if(outcome==='failure'){state.active='';emit('error','failedDemo');}else if(outcome==='verification'){state.challenge={id:crypto.randomUUID(),kind:'challenge',message:'verifyHint'};emit('awaiting_auth','verifyEvent');}else connected();});},
 Disconnect:async()=>{generation++;state.challenge=null;emit('disconnecting','disconnecting');after(()=>{state.active='';state.connectedAt=0;emit('disconnected','disconnectedEvent');});},
 SubmitChallenge:async(id,_discardedResponse)=>{if(state.challenge?.id!==id)throw new Error('验证码已过期');state.challenge=null;emit('connecting','connectingEvent');after(connected);},
 ForgetPassword:async id=>{const p=profile(id);p.hasPassword=false;p.remember=false;},
 CheckVersion:async()=>({current:'0.1.0-preview',latest:'v0.1.0',status:'You are on the latest release'}),
 ExportData:async()=>JSON.stringify({version:1,profiles:state.profiles.map(p=>({...clone(p),username:'',remember:false,hasPassword:false})),preferences:state.preferences},null,2),
 ImportData:async data=>{if(data.length>1024*1024)throw new Error('文件超过 1 MB');const input=JSON.parse(data);if(input.version!==1||!Array.isArray(input.profiles)||input.profiles.length>100)throw new Error('配置文件格式不正确');const imported=input.profiles.map(p=>{if(!p.name||!p.server||!['atrust','easyconnect'].includes(p.protocol)||!Number.isInteger(p.port)||p.port<1||p.port>65535)throw new Error('线路配置无效');return {id:crypto.randomUUID(),name:String(p.name).slice(0,80),server:String(p.server).slice(0,253),port:p.port,protocol:p.protocol,username:'',remember:false,hasPassword:false,customDomains:typeof p.customDomains==='string'?p.customDomains.slice(0,8192):'',options:p.options&&typeof p.options==='object'?clone(p.options):{}};});if(state.profiles.length+imported.length>100)throw new Error('最多导入 100 条线路');state.profiles.push(...imported);if(!state.selected)state.selected=imported[0]?.id||'';}
 };
 reset('routes');return {api,reset};
}
