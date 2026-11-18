import {t} from '../../core/i18n.js';
import {store} from '../../core/store.js';
import {icon} from '../../core/ui.js';

export function formatBytes(bytes){if(!Number.isFinite(bytes)||bytes<0)return '—';const units=['B','KB','MB','GB'];let i=0;while(bytes>=1024&&i<units.length-1){bytes/=1024;i++;}return `${bytes.toFixed(i===0?0:1)} ${units[i]}`;}

function backgroundLineplot(history){
	const w=320,h=72,pad=2,n=history.length;
	if(n<2)history=[{down:0,up:0},...history,{down:0,up:0}];
	const max=Math.max(1,...history.map(p=>Math.max(p.down,p.up)));
	const x=i=>(i*(w/(history.length-1))).toFixed(1),y=v=>(h-pad-(v/max)*(h-2*pad)).toFixed(1);
	const line=k=>history.map((p,i)=>`${x(i)},${y(p[k])}`).join(' ');
	return `<svg class="speed-lineplot-bg" viewBox="0 0 ${w} ${h}" preserveAspectRatio="none" aria-hidden="true">
		<defs>
			<linearGradient id="downGrad" x1="0%" y1="0%" x2="0%" y2="100%">
				<stop offset="0%" stop-color="var(--ink)" stop-opacity="0.14" />
				<stop offset="100%" stop-color="var(--ink)" stop-opacity="0.01" />
			</linearGradient>
		</defs>
		<polygon class="area" fill="url(#downGrad)" points="0,${h} ${line('down')} ${w},${h}"/>
		<polyline class="down-line" points="${line('down')}" fill="none" stroke="currentColor" stroke-width="1.8" opacity="0.35"/>
		<polyline class="up-line" points="${line('up')}" fill="none" stroke="currentColor" stroke-dasharray="3,3" stroke-width="1.2" opacity="0.25"/>
	</svg>`;
}

export function networkSpeed(){
	const s=store.snapshot,m=s.traffic,live=s.state==='connected',valid=live&&m&&Date.now()-m.sampledAt<4000;
	const rate=k=>!live?'0 B/s':valid?formatBytes(m[k])+'/s':'—';
	return `<section class="home-module speed-module" aria-labelledby="speed-title">
		<div class="module-heading">
			<div style="display:flex;align-items:center;gap:8px;">
				${icon('activity')}
				<h2 id="speed-title">${t('networkSpeed')}</h2>
			</div>
			${live && s.connectedAt ? `
				<div class="speed-duration">
					<span class="speed-duration-label">${t('duration')}:</span>
					<strong data-elapsed="${s.connectedAt}">00:00:00</strong>
				</div>
			` : `<span class="module-caption">${t(s.preview?'simulatedTraffic':'proxyTraffic')}</span>`}
		</div>
		<div class="speed-content-wrap">
			<div class="speed-values-compact">
				<div class="speed-metric down">
					<span class="speed-label">${icon('arrow-down')} ${t('downloadSpeed')}</span>
					<strong class="speed-rate">${rate('downloadRate')}</strong>
					<small class="speed-total">${t('downloadTotal')}: ${m?formatBytes(m.downloaded):'—'}</small>
				</div>
				<div class="speed-metric up">
					<span class="speed-label">${icon('arrow-up')} ${t('uploadSpeed')}</span>
					<strong class="speed-rate">${rate('uploadRate')}</strong>
					<small class="speed-total">${t('uploadTotal')}: ${m?formatBytes(m.uploaded):'—'}</small>
				</div>
			</div>
		</div>
		<div class="speed-chart">${backgroundLineplot(live?store.speedHistory:[])}</div>
	</section>`;
}
