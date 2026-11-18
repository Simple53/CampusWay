import {applyAppearance} from './core/themes.js';
import {call} from './core/bridge.js';
import {setLanguage,t,message} from './core/i18n.js';
import {store,selected,active,primary} from './core/store.js';
import {esc,icon,field,button,download,toggleSwitch} from './core/ui.js';
import {home,updateHome} from './pages/home.js';
import {getTemplateSites,websiteURL} from './components/home/network-diag.js';
import {connections,routes} from './pages/connections.js';
import {editor,syncProtocol,readProfile} from './pages/editor.js';
import {settings} from './pages/settings.js';
const app=document.querySelector('#app'),modal=document.querySelector('#modal'),notice=document.querySelector('#notice');
let snapshotKey='',challengeID='',editorProfile=null,versionResult=null,noticeTimer;
let localIPCheckedAt=0;
let updateCheckAt=0,updateChecking=false,updating=false;
function route(){return location.hash.slice(1).split('/').filter(Boolean);}
function go(path){location.hash=path;}
function toast(text){notice.textContent=message(text);clearTimeout(noticeTimer);noticeTimer=setTimeout(()=>notice.textContent='',4500);}
function render(){const scroll=document.querySelector('#main')?.scrollTop||0;const [page='home',section]=route(),s=store.snapshot;setLanguage(s.preferences.language);applyAppearance(s.preferences);document.body.dataset.density=s.preferences.density;
 app.innerHTML=` <main id="main" tabindex="-1">${page==='editor'?editor(editorProfile||{}):page==='connections'?connections():page==='settings'?settings(section,versionResult):home()}</main>
 <nav class="bottom-nav" aria-label="${t('connections')}"><div class="nav-connection">${connectionControls()}</div>${[['home','activity'],['connections','route'],['settings','settings-2']].map(([id,i])=>`<a href="#${id}" ${page===id?'aria-current="page"':''}>${icon(i)}<span>${t(id)}</span></a>`).join('')}<div class="nav-updates">${updateControl()}</div></nav>`;
 syncProtocol();applyBusy();elapsed();document.querySelector('#main').scrollTop=scroll;}
function updateControl(){
 if(versionResult?.status!=='A different release is available')return '';
 const label=t(versionResult.install==='true'?'updateNow':'openRelease')+' '+versionResult.latest;
 return `<button type="button" class="nav-update" data-action="apply-update" title="${esc(label)}" aria-label="${esc(label)}" ${updating?'disabled':''}>${icon(updating?'refresh':'download-update')}</button>`;
}
function renderUpdate(){
 const control=document.querySelector('.nav-updates');
 if(control)control.innerHTML=updateControl();
}
async function checkUpdates(manual=false){
 if(updateChecking||updating||(!manual&&store.snapshot.preferences.autoUpdate===false))return;
 if(!manual&&Date.now()-updateCheckAt<4*60*60*1000)return;
 updateChecking=true;
 updateCheckAt=Date.now();
 try{
  versionResult=await call('CheckVersion','');
  renderUpdate();
  if(route().join('/')==='settings/about')render();
 }catch(err){
  if(manual)toast(err.message);
 }finally{
  updateChecking=false;
 }
}
function connectionControls(){
 const s=store.snapshot,p=active()||selected(),on=!!s.active&&s.state!=='disconnecting';
 return `${p?`<span class="current-route-selector"><span class="route-hint-text">${t('currentRoute')}:</span> <strong>${esc(p.name)}</strong></span>`:''}${p?toggleSwitch(on,'toggle-global',`aria-label="${t(on?'disconnect':'connect')}" ${s.state==='disconnecting'?'disabled':''}`):''}`;
}
function elapsed(){document.querySelectorAll('[data-elapsed]').forEach(e=>{const seconds=Math.max(0,Math.floor((Date.now()-Number(e.dataset.elapsed))/1000));e.textContent=`${Math.floor(seconds/3600).toString().padStart(2,'0')}:${Math.floor(seconds/60%60).toString().padStart(2,'0')}:${(seconds%60).toString().padStart(2,'0')}`;});}
function applyBusy(){document.querySelectorAll('button.primary, #editor button, [data-select], [data-action=edit-route], [data-action=connect-route], [data-action=toggle-global]').forEach(e=>e.disabled=store.busy||e.getAttribute('aria-checked')==='false'&&store.snapshot.state==='disconnecting');app.setAttribute('aria-busy',String(store.busy));}
async function refresh(force=false){const next=await call('GetSnapshot'),key=JSON.stringify(next),previous=store.snapshot;
 if(next.state==='error'&&previous.state!=='error')store.failedID=next.active||previous.active||next.selected;
 if(next.state!=='error')store.failedID='';
 // 收集连接期实时速率历史点供 SVG Sparkline 折线图使用（至多 25 个点，断开时重置）
 if(next.state==='connected'&&next.traffic){
  store.speedHistory.push({down:next.traffic.downloadRate||0,up:next.traffic.uploadRate||0});
  if(store.speedHistory.length>25)store.speedHistory.shift();
 }else if(next.state!=='connected'&&store.speedHistory.length>0){
  store.speedHistory=[];
 }
 store.snapshot=next;
 if(force||key!==snapshotKey){const first=!snapshotKey;snapshotKey=key;const page=route()[0]||'home';
 const samePreferences=JSON.stringify(previous.preferences)===JSON.stringify(next.preferences);
 if(!first&&samePreferences&&page==='home')updateHome();else if(!first&&samePreferences&&page==='connections'&&document.querySelector('#route-list')){if(JSON.stringify([previous.profiles,previous.selected,previous.active,previous.state])!==JSON.stringify([next.profiles,next.selected,next.active,next.state]))document.querySelector('#route-list').innerHTML=routes();applyBusy();}else if(!first&&samePreferences&&page==='settings'){
  if(route()[1]==='diagnostics'&&JSON.stringify(previous.logs)!==JSON.stringify(next.logs))render();
 }else if(first||page!=='editor')render();}
 const controls=document.querySelector('.nav-connection');const html=connectionControls();if(controls&&controls.dataset.rendered!==html){controls.innerHTML=html;controls.dataset.rendered=html;}applyBusy();
 if(next.challenge&&next.challenge.id!==challengeID){challengeID=next.challenge.id;verification(next.challenge);}else if(!next.challenge&&challengeID){challengeID='';if(modal.dataset.kind==='challenge')modal.close();}}

async function task(fn,rethrow=false){if(store.busy)return;store.busy=true;applyBusy();try{await fn();await refresh();}catch(e){if(rethrow)throw e;toast(e.message);}finally{store.busy=false;applyBusy();}}
function dialog(title,body,kind=''){if(modal.open)modal.close();modal.dataset.kind=kind;modal.innerHTML=`<div class="dialog-heading"><h2>${t(title)}</h2></div>${body}`;modal.showModal();}
function confirmAction(title,fn){dialog(title,`<div class="dialog-actions">${button('cancel','close')}${button('confirm','confirm','primary')}</div>`);modal.querySelector('[data-action=confirm]').onclick=()=>{modal.close();task(fn);};}
function edit(profile){
	// 仅在当前线路真正处于已连接状态时才阻止编辑；断开或出错状态均允许直接配置
	if(profile?.id===store.snapshot.active&&store.snapshot.state==='connected')return toast('noEditActive');
	editorProfile=profile?structuredClone(profile):{};
	go('editor');
}
async function start(p=selected(),automatic=false){if(!p)return edit();if(p.id===store.snapshot.active)return automatic?undefined:task(()=>call('Disconnect'));if(!p.username&&!store.snapshot.preview){toast('saveFirst');return edit(p);}const run=async()=>{if(p.hasPassword||p.remember||store.snapshot.preview||(p.protocol==='atrust'&&p.options?.authType==='auth/smsCheckCode')){try{await task(()=>call('Connect',p.id,'',false),true);return;}catch(err){if(!err?.message?.toLowerCase().includes('password'))return toast(err.message);}}dialog('auth',`<form id="password-form">${field('password','password','','password','required autocomplete="current-password" autofocus')}<label class="check"><input type="checkbox" name="remember" checked ${store.snapshot.canRemember?'':'disabled'}>${t('remember')}</label><div class="dialog-actions">${button('cancel','close')}<button class="primary" type="submit">${t('connect')}</button></div></form>`);const inp=document.querySelector('#password-form input[type=password]');if(inp)setTimeout(()=>inp.focus(),50);document.querySelector('#password-form').onsubmit=e=>{e.preventDefault();const value=e.target.elements.password.value;const remember=e.target.elements.remember?.checked??true;e.target.reset();modal.close();task(()=>call('Connect',p.id,value,remember));};};if(store.snapshot.active&&!automatic){dialog('switchConfirm',`<div class="dialog-actions">${button('cancel','close')}${button('confirm','confirm','primary')}</div>`);modal.querySelector('[data-action=confirm]').onclick=()=>{modal.close();run();};}else run();}
function verification(e){
	const imgHtml = e.image ? `
		<div class="captcha-box">
			<div class="captcha-preview-wrap" id="captcha-wrap" title="${t('captchaHint')}">
				<img id="captcha-img" class="challenge-image" src="${esc(e.image)}" alt="${t('verify')}" draggable="false">
				<div id="captcha-pins"></div>
			</div>
			<div class="captcha-actions">
				<span id="captcha-hint" class="hint">${t('captchaHint')}</span>
				<div style="display:flex;gap:6px;">
					<button type="button" class="quiet" id="reset-points-btn">${t('clearPoints')}</button>
					<button type="button" class="quiet" id="refresh-captcha-btn">${t('refreshCaptcha')}</button>
				</div>
			</div>
		</div>` : '';
	dialog('verify',`<p>${t('verifyHint')}</p>${imgHtml}<form id="verify-form">${field('response','response','','text','autocomplete="one-time-code" autofocus')}<div class="dialog-actions">${button('stop','cancel-challenge')}<button type="submit" id="submit-verify-btn" class="primary">${t('submit')}</button></div></form>`,'challenge');
	const form=document.querySelector('#verify-form');
	const input=form?.querySelector('input');
	const submitBtn=form?.querySelector('#submit-verify-btn');
	const wrap=document.querySelector('#captcha-wrap');
	const img=document.querySelector('#captcha-img');
	const pins=document.querySelector('#captcha-pins');
	const resetBtn=document.querySelector('#reset-points-btn');
	const refreshBtn=document.querySelector('#refresh-captcha-btn');
	let points=[];

	if(input)setTimeout(()=>input.focus(),50);

	if(wrap&&img&&pins){
		wrap.onclick=(ev)=>{
			const rect=img.getBoundingClientRect();
			if(ev.clientX<rect.left||ev.clientX>rect.right||ev.clientY<rect.top||ev.clientY>rect.bottom)return;
			const scaleX=(img.naturalWidth||rect.width)/rect.width;
			const scaleY=(img.naturalHeight||rect.height)/rect.height;
			const px=Math.round((ev.clientX-rect.left)*scaleX);
			const py=Math.round((ev.clientY-rect.top)*scaleY);
			points.push({x:px,y:py});

			const pin=document.createElement('span');
			pin.className='captcha-pin';
			pin.style.left=`${ev.clientX-rect.left}px`;
			pin.style.top=`${ev.clientY-rect.top}px`;
			pin.textContent=String(points.length);
			pins.appendChild(pin);

			if(input){
				const nw = img.naturalWidth || Math.round(rect.width);
				const nh = img.naturalHeight || Math.round(rect.height);
				input.value = JSON.stringify({
					coordinates: points.map(p => [p.x, p.y]),
					width: nw,
					height: nh
				});
			}
		};
	}

	if(resetBtn){
		resetBtn.onclick=()=>{
			points=[];
			if(pins)pins.innerHTML='';
			if(input)input.value='';
		};
	}

	if(refreshBtn){
		refreshBtn.onclick=async()=>{
			modal.close();
			challengeID='';
			toast('refreshingCaptcha');
			const profile=active()||selected();
			await task(()=>call('Disconnect'));
			await start(profile,true);
		};
	}

	const doSubmit=async()=>{
		const value=(input?.value||'').trim();
		if(!value)return;
		if(submitBtn)submitBtn.disabled=true;
		modal.close();
		challengeID='';
		try{
			await call('SubmitChallenge',e.id,value);
			await refresh();
		}catch(err){
			toast(err.message||String(err));
		}
	};

	if(form){
		form.onsubmit=event=>{
			event.preventDefault();
			doSubmit();
		};
	}
	if(submitBtn){
		submitBtn.onclick=event=>{
			event.preventDefault();
			doSubmit();
		};
	}
}

/**
 * handleSaveProfile 统一处理线路配置保存：
 * 1. 无论是点击保存按钮、快捷回车还是表单 submit 均统一调用，保证即点即响应；
 * 2. 在前端先行校验必填项并聚焦错误字段，提示明确且绝不静默吞掉操作；
 * 3. 密码与账户持久化后返回线路列表并提示 savedNotice。
 */
async function handleSaveProfile(form){
	const formEl=form||document.querySelector('#editor');
	if(!formEl)return;
	const username=(formEl.elements.username?.value||'').trim();
	if(!username&&!store.snapshot.preview){
		const inp=formEl.elements.username;
		if(inp)inp.focus();
		return toast('saveFirst');
	}
	const name=(formEl.elements.name?.value||'').trim();
	if(!name){
		const inp=formEl.elements.name;
		if(inp)inp.focus();
		return toast('required');
	}
	const server=(formEl.elements.server?.value||'').trim();
	if(!server){
		const inp=formEl.elements.server;
		if(inp)inp.focus();
		return toast('required');
	}
	const profileData=readProfile(formEl);
	const passwordVal=formEl.elements.password?.value||'';
	await task(async()=>{
		await call('SaveProfile',profileData,passwordVal);
		formEl.reset();
		editorProfile=null;
		go('connections');
		toast('savedNotice');
	});
}

app.addEventListener('submit',e=>{
	if(e.target.id==='dns-settings'){
		e.preventDefault();
		const form=new FormData(e.target);
		return task(async()=>{
			const dns={enabled:form.has('enabled'),remote:form.get('remote').trim(),secondary:form.get('secondary').trim(),ttl:Number(form.get('ttl'))||0,records:form.get('records').trim()};
			await call('SavePreferences',{...store.snapshot.preferences,dns});
			toast('dnsSaved');
		});
	}
	if(e.target.id!=='editor')return;
	e.preventDefault();
	handleSaveProfile(e.target);
});
document.addEventListener('click',e=>{const target=e.target.closest('[data-action],[data-route-connect],[data-select],[data-accent]');if(!target)return;if(target.dataset.accent)return task(()=>call('SavePreferences',{...store.snapshot.preferences,accent:target.dataset.accent}));const a=target.dataset.action;
if(a==='theme-mode')return task(()=>call('SavePreferences',{...store.snapshot.preferences,theme:target.dataset.themeMode}));
if(a==='save-profile'){
	e.preventDefault();
	return handleSaveProfile(document.querySelector('#editor'));
}
if(a==='toggle-connect'||a==='toggle-global'){
	const s=store.snapshot;
	if(s.active){
		return task(()=>call('Disconnect'));
	}
	return start(active()||selected());
}
if(a==='minimize-window')return call('MinimizeWindow');
if(a==='connect-route'||target.dataset.routeConnect){
 const p=store.snapshot.profiles.find(p=>p.id===(target.dataset.routeConnect||target.dataset.id));
 if(!p||store.busy||store.snapshot.state==='disconnecting')return;
 return p.id===store.snapshot.active?task(()=>call('Disconnect')):start(p,true);
}
if(a==='refresh-ip'){
	if(store.ipRefreshing)return;
	store.ipRefreshing=true;updateHome();
	return task(async()=>{
		try{await refreshLocalIPs(true);store.ipInfo=await call('RefreshIPInfo');}
		finally{store.ipRefreshing=false;updateHome();}
	});
}
if(a==='website-link'){
	const url=websiteURL(target.dataset.url);
	if(!url)return;
	if(e.ctrlKey||e.metaKey){
		if(window.runtime?.BrowserOpenURL)window.runtime.BrowserOpenURL(url);
		else window.open(url,'_blank','noopener,noreferrer');
		return;
	}
	return task(async()=>{await navigator.clipboard.writeText(url);toast('copied');});
}
if(a==='edit-links'){
	const sites=store.snapshot.preferences.websiteLinks?.length?store.snapshot.preferences.websiteLinks:getTemplateSites(active()||selected());
	dialog('editLinks',`<form id="website-form"><label class="field"><span>${t('websiteInput')}</span><textarea rows="6" maxlength="24576">${esc(sites.map(s=>[s.name,s.url].filter(Boolean).join(' | ')).join('\n'))}</textarea></label><div class="dialog-actions">${button('resetLinks','reset-links','quiet')}${button('cancel','close')}<button class="primary" type="submit">${t('save')}</button></div></form>`);
	const form=modal.querySelector('#website-form');
	const save=websiteLinks=>task(async()=>{
		await call('SavePreferences',{...store.snapshot.preferences,websiteLinks});
		store.diagReport=null;
		modal.close();
	});
	form.querySelector('[data-action=reset-links]').onclick=()=>save([]);
	form.onsubmit=event=>{
		event.preventDefault();
		const lines=form.querySelector('textarea').value.split(/\r?\n/).map(s=>s.trim()).filter(Boolean);
		const links=lines.map(line=>{
			const delimiter=line.indexOf('|');
			const name=delimiter<0?'':line.slice(0,delimiter).trim();
			return {name,url:websiteURL(line.slice(delimiter+1).trim())};
		});
		if(links.length>12||links.some(l=>!l.url||l.url.length>2048||l.name.length>80))return toast('invalidWebsite');
		save(links);
	};
	return;
}
if(a==='run-diag'){store.diagTesting=true;updateHome();return task(async()=>{try{store.diagReport=await call('RunDiagnostics');}finally{store.diagTesting=false;updateHome();}});}
// 首页组件自定义编辑交互
if(a==='toggle-home-edit'){store.homeEdit=!store.homeEdit;render();return;}
if(a==='toggle-widget-size'){
	const w=target.dataset.widget||target.closest('[data-widget]')?.dataset.widget;
	if(!w)return;
	const sizes={...(store.snapshot.preferences.widgetSizes||{})};
	sizes[w]=Number(target.closest('.widget-shell')?.dataset.columns)===12?'half':'full';
	const widgetDimensions={...store.snapshot.preferences.widgetDimensions};
	delete widgetDimensions[w];
	return task(()=>call('SavePreferences',{...store.snapshot.preferences,widgetSizes:sizes,widgetDimensions}));
}
if(a==='move-widget-up'){
	const w=target.dataset.widget;const cur=[...(store.snapshot.preferences.widgets||['status','speed','diag','logs'])];
	const idx=cur.indexOf(w);
	if(idx>0){const tmp=cur[idx-1];cur[idx-1]=cur[idx];cur[idx]=tmp;return task(()=>call('SavePreferences',{...store.snapshot.preferences,widgets:cur}));}
	return;
}
if(a==='move-widget-down'){
	const w=target.dataset.widget;const cur=[...(store.snapshot.preferences.widgets||['status','speed','diag','logs'])];
	const idx=cur.indexOf(w);
	if(idx>=0&&idx<cur.length-1){const tmp=cur[idx+1];cur[idx+1]=cur[idx];cur[idx]=tmp;return task(()=>call('SavePreferences',{...store.snapshot.preferences,widgets:cur}));}
	return;
}
if(a==='hide-widget'){
	const w=target.dataset.widget;const cur=[...(store.snapshot.preferences.widgets||['status','speed','diag','logs'])];
	const next=cur.filter(x=>x!==w);
	return task(()=>call('SavePreferences',{...store.snapshot.preferences,widgets:next}));
}
if(a==='add-widget'){
	const w=target.dataset.widget;const cur=[...(store.snapshot.preferences.widgets||['status','speed','diag','logs'])];
	if(!cur.includes(w)){cur.push(w);return task(()=>call('SavePreferences',{...store.snapshot.preferences,widgets:cur}));}
	return;
}
if(target.dataset.select)return task(()=>call('SelectProfile',target.dataset.select));if(a==='edit-route')return edit(store.snapshot.profiles.find(p=>p.id===target.dataset.id));if(a==='add')return edit();if(a==='edit')return edit(selected());if(a==='edit-account')return edit(store.snapshot.profiles.find(p=>p.id===target.dataset.id));if(a==='goto-connections')return go('connections');if(a==='connect')return start(active()||selected());if(a==='close')return modal.close();if(a==='cancel-challenge')return task(async()=>{await call('Disconnect');modal.close();});if(a?.startsWith('view-'))return task(()=>call('SavePreferences',{...store.snapshot.preferences,view:a.slice(5)}));if(a==='delete')return confirmAction('removeConfirm',async()=>{await call('DeleteProfile',editorProfile.id);editorProfile=null;go('connections');});if(a==='forget')return confirmAction('forget',()=>call('ForgetPassword',target.dataset.id));if(a==='export')return task(async()=>download(await call('ExportData'),'RouteDesk-routes.json'));
if(a==='export-daily-logs')return task(async()=>{const today=new Date().toISOString().slice(0,10);download(await call('ExportDailyLogs'),`RouteDesk-${today}.log`);toast('exported');});
if(a==='open-logs-folder')return task(async()=>call('OpenLogsFolder'));
if(a==='apply-update'){
 if(updating||store.busy)return;
 if(versionResult?.install!=='true'){
  const url='https://github.com/Simple53/CampusWay/releases/latest';
  if(window.runtime?.BrowserOpenURL)window.runtime.BrowserOpenURL(url);
  else window.open(url,'_blank','noopener,noreferrer');
  return;
 }
 updating=true;renderUpdate();toast('updating');
 return task(async()=>{
  try{
   const result=await call('ApplyUpdate','');
   if(result.status==='restarting')toast('updateRestarting');
  }finally{updating=false;renderUpdate();}
 });
}
if(a==='toggle-ip-mask'){store.ipMasked=!store.ipMasked;updateHome();return;}
if(a==='check')return checkUpdates(true);if(a==='export-logs')download(JSON.stringify(store.snapshot.logs,null,2),'RouteDesk-events.json');if(a==='copy-logs')task(async()=>{await navigator.clipboard.writeText(store.snapshot.logs.map(l=>`${l.time} [${(l.level||'info').toUpperCase()}] ${message(l.message)}`).join('\n'));toast('copied');});});
app.addEventListener('input',e=>{if(e.target.id==='search'){store.query=e.target.value;document.querySelector('#route-list').innerHTML=routes();}});
app.addEventListener('change',e=>{if(e.target.closest('#editor')){syncProtocol();return;}
if(e.target.name==='autoUpdate'){
 const autoUpdate=e.target.checked;
 return task(async()=>{
  await call('SavePreferences',{...store.snapshot.preferences,autoUpdate});
  if(autoUpdate){updateCheckAt=0;await refresh();void checkUpdates();}
 });
}
if(e.target.dataset.widget){
	const w=e.target.dataset.widget;
	const current=[...(store.snapshot.preferences.widgets||['status','speed','diag','logs'])];
	const next=e.target.checked?Array.from(new Set([...current,w])):current.filter(x=>x!==w);
	return task(()=>call('SavePreferences',{...store.snapshot.preferences,widgets:next}));
}
if(['theme','view','density','sort','language','closeAction'].includes(e.target.name))return task(()=>call('SavePreferences',{...store.snapshot.preferences,[e.target.name]:e.target.value}));if(e.target.id==='import-file'){const file=e.target.files[0];if(file)task(async()=>{if(file.size>1024*1024)throw new Error('Import exceeds 1 MB');await call('ImportData',await file.text());toast('imported');render();});}});

// 首页组件原生拖拽排序交互
let draggedWidget = null;
const resizeBounds=(columns,height)=>({
	columns:Math.max(4,Math.min(12,columns)),
	height:Math.max(144,Math.min(720,height))
});
function saveWidgetDimensions(shell,dimensions){
	const widgetDimensions={...store.snapshot.preferences.widgetDimensions,[shell.dataset.widget]:dimensions};
	return task(()=>call('SavePreferences',{...store.snapshot.preferences,widgetDimensions}));
}
app.addEventListener('pointerdown',e=>{
	const handle=e.target.closest('.widget-resize-handle');
	if(!handle||e.button!==0||store.busy)return;
	e.preventDefault();
	const shell=handle.closest('.widget-shell'),grid=shell.parentElement;
	const step=grid.clientWidth/12, narrow=grid.clientWidth<600;
	const columns=Number(shell.dataset.columns),height=shell.offsetHeight;
	let dimensions;
	handle.setPointerCapture(e.pointerId);
	shell.classList.add('resizing');
	const move=event=>{
		dimensions=resizeBounds(narrow?columns:columns+Math.round((event.clientX-e.clientX)/step),Math.round((height+event.clientY-e.clientY)/24)*24);
		shell.style.setProperty('--widget-columns',dimensions.columns);
		shell.style.setProperty('--widget-height',dimensions.height+'px');
	};
	const finish=event=>{
		handle.removeEventListener('pointermove',move);
		handle.removeEventListener('lostpointercapture',finish);
		shell.classList.remove('resizing');
		if(dimensions)saveWidgetDimensions(shell,dimensions);
	};
	handle.addEventListener('pointermove',move);
	handle.addEventListener('lostpointercapture',finish);
});
app.addEventListener('keydown',e=>{
	const handle=e.target.closest('.widget-resize-handle');
	if(!handle||!['ArrowLeft','ArrowRight','ArrowUp','ArrowDown'].includes(e.key)||store.busy)return;
	e.preventDefault();
	const shell=handle.closest('.widget-shell');
	const columns=Number(shell.dataset.columns)+Number(e.key==='ArrowRight')-Number(e.key==='ArrowLeft');
	const height=shell.offsetHeight+24*(Number(e.key==='ArrowDown')-Number(e.key==='ArrowUp'));
	saveWidgetDimensions(shell,resizeBounds(columns,height));
});
app.addEventListener('dragstart', e => {
	const shell = e.target.closest('.widget-shell');
	if (!shell) return;
	draggedWidget = shell.dataset.widget;
	shell.classList.add('dragging');
	if (e.dataTransfer) {
		e.dataTransfer.effectAllowed = 'move';
		e.dataTransfer.setData('text/plain', draggedWidget);
	}
});
app.addEventListener('dragover', e => {
	const shell = e.target.closest('.widget-shell');
	if (!shell || !draggedWidget || shell.dataset.widget === draggedWidget) return;
	e.preventDefault();
	if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
	shell.classList.add('drag-over');
});
app.addEventListener('dragleave', e => {
	const shell = e.target.closest('.widget-shell');
	if (shell) shell.classList.remove('drag-over');
});
app.addEventListener('drop', e => {
	const shell = e.target.closest('.widget-shell');
	if (!shell || !draggedWidget) return;
	e.preventDefault();
	shell.classList.remove('drag-over');
	const targetWidget = shell.dataset.widget;
	if (targetWidget && targetWidget !== draggedWidget) {
		const cur = [...(store.snapshot.preferences.widgets || ['status', 'speed', 'diag', 'logs'])];
		const fromIdx = cur.indexOf(draggedWidget);
		const toIdx = cur.indexOf(targetWidget);
		if (fromIdx >= 0 && toIdx >= 0) {
			cur.splice(fromIdx, 1);
			cur.splice(toIdx, 0, draggedWidget);
			task(() => call('SavePreferences', { ...store.snapshot.preferences, widgets: cur }));
		}
	}
	draggedWidget = null;
});
app.addEventListener('dragend', () => {
	document.querySelectorAll('.widget-shell').forEach(el => el.classList.remove('dragging', 'drag-over'));
	draggedWidget = null;
});

modal.addEventListener('cancel',e=>{if(modal.dataset.kind==='challenge'){e.preventDefault();task(async()=>{await call('Disconnect');modal.close();});}});
window.addEventListener('hashchange',()=>{render();document.querySelector('#main').focus({preventScroll:true});});
// Only snapshots update on polling; an editor draft is never replaced by background events.
async function refreshLocalIPs(force=false){
	if(!force&&Date.now()-localIPCheckedAt<30000)return;
	localIPCheckedAt=Date.now();
	try{
		store.localIPs=await call('GetLocalIPs');
		if((route()[0]||'home')==='home')updateHome();
	}catch{}
}
async function poll(){try{if(!store.busy){await refresh();void checkUpdates();void refreshLocalIPs();}}catch(e){if(!snapshotKey){render();toast(e.message);}}finally{setTimeout(poll,700);}}
poll();setInterval(()=>{elapsed();if((route()[0]||'home')==='home')updateHome();},1000);

matchMedia('(prefers-color-scheme: dark)').addEventListener('change',()=>{if(store.snapshot.preferences.theme==='system')applyAppearance(store.snapshot.preferences);});
