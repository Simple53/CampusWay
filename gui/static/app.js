import {applyAppearance} from './core/themes.js';
import {call} from './core/bridge.js';
import {setLanguage,t,message} from './core/i18n.js';
import {store,selected,active,primary} from './core/store.js';
import {esc,icon,field,button,download,toggleSwitch} from './core/ui.js';
import {home,updateHome} from './pages/home.js';
import {connections,routes} from './pages/connections.js';
import {editor,syncProtocol,readProfile} from './pages/editor.js';
import {settings} from './pages/settings.js';
const app=document.querySelector('#app'),modal=document.querySelector('#modal'),notice=document.querySelector('#notice');
let snapshotKey='',challengeID='',editorProfile=null,versionResult=null,noticeTimer;
function route(){return location.hash.slice(1).split('/').filter(Boolean);}
function go(path){location.hash=path;}
function toast(text){notice.textContent=message(text);clearTimeout(noticeTimer);noticeTimer=setTimeout(()=>notice.textContent='',4500);}
function render(){const scroll=document.querySelector('#main')?.scrollTop||0;const [page='home',section]=route(),s=store.snapshot;setLanguage(s.preferences.language);applyAppearance(s.preferences);document.body.dataset.density=s.preferences.density;
 const activeRoute = active() || selected();
 const isRunning = s.state === 'connected';
 app.innerHTML=`<header class="topbar">
   <div style="display:flex;align-items:center;gap:12px;">
     <a class="brand" href="#home">${icon('route')}CampusWay</a>
     ${activeRoute ? `<span class="header-route-name">${esc(activeRoute.name)}</span>` : ''}
   </div>
   <div style="display:flex;align-items:center;gap:12px;">
     ${activeRoute ? toggleSwitch(isRunning, 'toggle-global', `title="${t(isRunning?'disconnect':'connect')}"`) : ''}
   </div>
 </header>
 <main id="main" tabindex="-1">${page==='editor'?editor(editorProfile||{}):page==='connections'?connections():page==='settings'?settings(section,versionResult):home()}</main>
 <nav class="bottom-nav" aria-label="${t('connections')}">${[['home','activity'],['connections','route'],['settings','settings-2']].map(([id,i])=>`<a href="#${id}" ${page===id?'aria-current="page"':''}>${icon(i)}<span>${t(id)}</span></a>`).join('')}</nav>`;
 syncProtocol();applyBusy();elapsed();document.querySelector('#main').scrollTop=scroll;}
function elapsed(){document.querySelectorAll('[data-elapsed]').forEach(e=>{const seconds=Math.max(0,Math.floor((Date.now()-Number(e.dataset.elapsed))/1000));e.textContent=`${Math.floor(seconds/3600).toString().padStart(2,'0')}:${Math.floor(seconds/60%60).toString().padStart(2,'0')}:${(seconds%60).toString().padStart(2,'0')}`;});}
function applyBusy(){document.querySelectorAll('button.primary, #editor button, [data-select], [data-action=edit-route]').forEach(e=>e.disabled=store.busy);app.setAttribute('aria-busy',String(store.busy));}
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
 if(!first&&samePreferences&&page==='home')updateHome();else if(!first&&samePreferences&&page==='connections'&&document.querySelector('#route-list')){if(JSON.stringify([previous.profiles,previous.selected,previous.active,previous.state])!==JSON.stringify([next.profiles,next.selected,next.active,next.state]))document.querySelector('#route-list').innerHTML=routes();applyBusy();}else if(first||page!=='editor')render();}
 if(next.challenge&&next.challenge.id!==challengeID){challengeID=next.challenge.id;verification(next.challenge);}else if(!next.challenge&&challengeID){challengeID='';if(modal.dataset.kind==='challenge')modal.close();}}

async function task(fn){if(store.busy)return;store.busy=true;applyBusy();try{await fn();await refresh();}catch(e){toast(e.message);}finally{store.busy=false;applyBusy();}}
function dialog(title,body,kind=''){if(modal.open)modal.close();modal.dataset.kind=kind;modal.innerHTML=`<div class="dialog-heading"><h2>${t(title)}</h2></div>${body}`;modal.showModal();}
function confirmAction(title,fn){dialog(title,`<div class="dialog-actions">${button('cancel','close')}${button('confirm','confirm','primary')}</div>`);modal.querySelector('[data-action=confirm]').onclick=()=>{modal.close();task(fn);};}
function edit(profile){
	// 仅在当前线路真正处于已连接状态时才阻止编辑；断开或出错状态均允许直接配置
	if(profile?.id===store.snapshot.active&&store.snapshot.state==='connected')return toast('noEditActive');
	editorProfile=profile?structuredClone(profile):{};
	go('editor');
}
async function start(p=selected(),automatic=false){if(!p)return edit();if(p.id===store.snapshot.active)return automatic?undefined:task(()=>call('Disconnect'));if(!p.username&&!store.snapshot.preview){toast('saveFirst');return edit(p);}const run=async()=>{if(p.hasPassword||p.remember||store.snapshot.preview||(p.protocol==='atrust'&&p.options?.authType==='auth/smsCheckCode')){try{await task(()=>call('Connect',p.id,'',false));return;}catch(err){if(!err?.message?.toLowerCase().includes('password'))return;}}dialog('auth',`<form id="password-form">${field('password','password','','password','required autocomplete="current-password" autofocus')}<label class="check"><input type="checkbox" name="remember" checked ${store.snapshot.canRemember?'':'disabled'}>${t('remember')}</label><div class="dialog-actions">${button('cancel','close')}<button class="primary" type="submit">${t('connect')}</button></div></form>`);const inp=document.querySelector('#password-form input[type=password]');if(inp)setTimeout(()=>inp.focus(),50);document.querySelector('#password-form').onsubmit=e=>{e.preventDefault();const value=e.target.elements.password.value;const remember=e.target.elements.remember?.checked??true;e.target.reset();modal.close();task(()=>call('Connect',p.id,value,remember));};};if(store.snapshot.active&&!automatic){dialog('switchConfirm',`<div class="dialog-actions">${button('cancel','close')}${button('confirm','confirm','primary')}</div>`);modal.querySelector('[data-action=confirm]').onclick=()=>{modal.close();run();};}else run();}
// 切换线路时仅更新选中状态，绝不自动发起连接
async function chooseRoute(id){
	if(store.busy||store.snapshot.selected===id)return;
	await task(()=>call('SelectProfile',id));
}

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
				input.value=points.map(p=>`${p.x},${p.y}`).join(';');
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
			await task(async()=>{
				await call('Disconnect');
				await start(active()||selected(),true);
			});
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
	if(e.target.id!=='editor')return;
	e.preventDefault();
	handleSaveProfile(e.target);
});
document.addEventListener('click',e=>{const target=e.target.closest('[data-action],[data-select],[data-accent]');if(!target)return;if(target.dataset.accent)return task(()=>call('SavePreferences',{...store.snapshot.preferences,accent:target.dataset.accent}));const a=target.dataset.action;
if(a==='save-profile'){
	e.preventDefault();
	return handleSaveProfile(document.querySelector('#editor'));
}
if(a==='toggle-connect'||a==='toggle-global'){
	const s=store.snapshot;
	if(s.state==='connected'||s.state==='connecting'){
		return task(()=>call('Disconnect'));
	}
	return start(active()||selected());
}
if(a==='minimize-window')return call('MinimizeWindow');
if(a==='toggle-route'){
	const id=target.dataset.id||target.closest('[data-id]')?.dataset.id;
	if(id)return chooseRoute(id);
	return;
}
if(a==='run-diag'){store.diagTesting=true;updateHome();return task(async()=>{try{store.diagReport=await call('RunDiagnostics');}finally{store.diagTesting=false;updateHome();}});}
// 首页组件自定义编辑交互
if(a==='toggle-home-edit'){store.homeEdit=!store.homeEdit;render();return;}
if(a==='toggle-widget-size'){
	const w=target.dataset.widget||target.closest('[data-widget]')?.dataset.widget;
	if(!w)return;
	const sizes={...(store.snapshot.preferences.widgetSizes||{})};
	const current=sizes[w]||((w==='diag'||w==='logs')?'full':'half');
	sizes[w]=current==='full'?'half':'full';
	return task(()=>call('SavePreferences',{...store.snapshot.preferences,widgetSizes:sizes}));
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
if(target.dataset.select)return chooseRoute(target.dataset.select);if(a==='edit-route')return edit(store.snapshot.profiles.find(p=>p.id===target.dataset.id));if(a==='add')return edit();if(a==='edit')return edit(selected());if(a==='edit-account')return edit(store.snapshot.profiles.find(p=>p.id===target.dataset.id));if(a==='goto-connections')return go('connections');if(a==='connect')return start(active()||selected());if(a==='close')return modal.close();if(a==='cancel-challenge')return task(async()=>{await call('Disconnect');modal.close();});if(a?.startsWith('view-'))return task(()=>call('SavePreferences',{...store.snapshot.preferences,view:a.slice(5)}));if(a==='delete')return confirmAction('removeConfirm',async()=>{await call('DeleteProfile',editorProfile.id);editorProfile=null;go('connections');});if(a==='forget')return confirmAction('forget',()=>call('ForgetPassword',target.dataset.id));if(a==='export')return task(async()=>download(await call('ExportData'),'CampusWay-routes.json'));if(a==='check')return task(async()=>{versionResult=await call('CheckVersion','');render();});if(a==='export-logs')download(JSON.stringify(store.snapshot.logs,null,2),'CampusWay-events.json');if(a==='copy-logs')task(async()=>{await navigator.clipboard.writeText(store.snapshot.logs.map(l=>`${l.time} ${message(l.message)}`).join('\n'));toast('copied');});});
app.addEventListener('input',e=>{if(e.target.id==='search'){store.query=e.target.value;document.querySelector('#route-list').innerHTML=routes();}});
app.addEventListener('change',e=>{if(e.target.closest('#editor')){syncProtocol();return;}
if(e.target.dataset.widget){
	const w=e.target.dataset.widget;
	const current=[...(store.snapshot.preferences.widgets||['status','speed','diag','logs'])];
	const next=e.target.checked?Array.from(new Set([...current,w])):current.filter(x=>x!==w);
	return task(()=>call('SavePreferences',{...store.snapshot.preferences,widgets:next}));
}
if(['theme','view','density','sort','language','closeAction'].includes(e.target.name))return task(()=>call('SavePreferences',{...store.snapshot.preferences,[e.target.name]:e.target.value}));if(e.target.id==='import-file'){const file=e.target.files[0];if(file)task(async()=>{if(file.size>1024*1024)throw new Error('Import exceeds 1 MB');await call('ImportData',await file.text());toast('imported');render();});}});

// 首页组件原生拖拽排序交互
let draggedWidget = null;
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

// 首页组件拖拽右下角拉伸放大/缩小交互（无按键，直接手势拖拽缩放）
let resizingWidget = null;
let resizeStartX = 0;
let resizeInitialSize = 'half';
let resizeShell = null;

app.addEventListener('pointerdown', e => {
	const handle = e.target.closest('.widget-resize-handle');
	if (!handle) return;
	e.preventDefault();
	e.stopPropagation();
	resizingWidget = handle.dataset.resizeWidget;
	resizeShell = handle.closest('.widget-shell');
	if (!resizeShell) return;
	resizeStartX = e.clientX;
	const sizes = store.snapshot.preferences.widgetSizes || {};
	const defaultSize = (resizingWidget === 'diag' || resizingWidget === 'logs') ? 'full' : 'half';
	resizeInitialSize = sizes[resizingWidget] || defaultSize;
	handle.classList.add('active');
	resizeShell.classList.add('resizing');
	if (handle.setPointerCapture) {
		try { handle.setPointerCapture(e.pointerId); } catch (_) {}
	}
});

app.addEventListener('pointermove', e => {
	if (!resizingWidget || !resizeShell) return;
	const dx = e.clientX - resizeStartX;
	if (resizeInitialSize === 'half' && dx > 40) {
		resizeShell.classList.remove('size-half');
		resizeShell.classList.add('size-full');
	} else if (resizeInitialSize === 'full' && dx < -40) {
		resizeShell.classList.remove('size-full');
		resizeShell.classList.add('size-half');
	} else {
		resizeShell.classList.toggle('size-full', resizeInitialSize === 'full');
		resizeShell.classList.toggle('size-half', resizeInitialSize === 'half');
	}
});

const finishResize = (e) => {
	if (!resizingWidget || !resizeShell) return;
	const dx = e.clientX - resizeStartX;
	let newSize = resizeInitialSize;
	if (resizeInitialSize === 'half' && dx > 40) {
		newSize = 'full';
	} else if (resizeInitialSize === 'full' && dx < -40) {
		newSize = 'half';
	} else if (Math.abs(dx) <= 6) {
		// 单击手柄也可快速双向切换，方便快捷
		newSize = resizeInitialSize === 'full' ? 'half' : 'full';
	}

	const w = resizingWidget;
	const handle = resizeShell.querySelector('.widget-resize-handle');
	if (handle) {
		handle.classList.remove('active');
		if (handle.releasePointerCapture && e.pointerId) {
			try { handle.releasePointerCapture(e.pointerId); } catch (_) {}
		}
	}
	resizeShell.classList.remove('resizing');
	resizingWidget = null;
	resizeShell = null;

	if (newSize !== resizeInitialSize) {
		const sizes = { ...(store.snapshot.preferences.widgetSizes || {}) };
		sizes[w] = newSize;
		task(() => call('SavePreferences', { ...store.snapshot.preferences, widgetSizes: sizes }));
	} else {
		render();
	}
};

app.addEventListener('pointerup', finishResize);
app.addEventListener('pointercancel', finishResize);
modal.addEventListener('cancel',e=>{if(modal.dataset.kind==='challenge'){e.preventDefault();task(async()=>{await call('Disconnect');modal.close();});}});
window.addEventListener('hashchange',()=>{render();document.querySelector('#main').focus({preventScroll:true});});
// Only snapshots update on polling; an editor draft is never replaced by background events.
async function poll(){try{if(!store.busy)await refresh();}catch(e){if(!snapshotKey){render();toast(e.message);}}finally{setTimeout(poll,700);}}
poll();setInterval(()=>{elapsed();if((route()[0]||'home')==='home')updateHome();},1000);

matchMedia('(prefers-color-scheme: dark)').addEventListener('change',()=>{if(store.snapshot.preferences.theme==='system')applyAppearance(store.snapshot.preferences);});
