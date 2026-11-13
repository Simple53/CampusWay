const frame=document.querySelector('#preview'),device=document.querySelector('#device'),stage=document.querySelector('#stage');let mode='desktop',scenario='routes';
function fit(){const w=mode==='mobile'?390:mode==='web'?Math.max(800,stage.clientWidth-48):800,h=mode==='mobile'?844:mode==='web'?Math.max(620,stage.clientHeight-80):620;const border=mode==='mobile'?14:2;device.style.width=(w+border)+'px';device.style.height=(h+border)+'px';frame.style.width=w+'px';frame.style.height=h+'px';device.className=mode;const scale=Math.min(1,(stage.clientWidth-40)/w,(stage.clientHeight-90)/h);device.style.transform=`scale(${Math.max(.1,scale)})`;const size=`${w} × ${h}`;document.querySelector('#dimensions').textContent=size;document.querySelector('#caption').textContent=({desktop:'桌面',mobile:'手机',web:'网页'}[mode])+' · '+size+(scale<1?` · ${Math.round(scale*100)}%`:'');}
function setScenario(){frame.contentWindow.postMessage({type:'campusway-preview',scenario},location.origin);}
document.querySelectorAll('[data-device]').forEach(b=>b.onclick=()=>{mode=b.dataset.device;document.querySelectorAll('[data-device]').forEach(q=>q.setAttribute('aria-pressed',q===b));fit();});
document.querySelector('#scenario').onchange=e=>{scenario=e.target.value;frame.src='/app?scene='+encodeURIComponent(scenario)+'&reload='+Date.now()+'#home';};
document.querySelector('#reload').onclick=()=>{frame.src='/app?reload='+Date.now()+'#home';};
document.querySelectorAll('[data-route]').forEach(b=>b.onclick=()=>{frame.contentWindow.location.hash=b.dataset.route;});
window.addEventListener('message',e=>{if(e.origin===location.origin&&e.source===frame.contentWindow&&e.data?.type==='campusway-preview-ready')setScenario();});
new ResizeObserver(fit).observe(stage);fit();
