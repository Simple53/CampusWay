import {t} from '../../core/i18n.js';
import {store} from '../../core/store.js';

export function formatBytes(bytes){if(!Number.isFinite(bytes)||bytes<0)return '—';const units=['B','KB','MB','GB'];let i=0;while(bytes>=1024&&i<units.length-1){bytes/=1024;i++;}return `${bytes.toFixed(i===0?0:1)} ${units[i]}`;}

/**
 * sparkline plots recent download (solid, filled) and upload (dashed) rates as
 * an SVG polyline. Both series share one scale so they stay comparable.
 */
function sparkline(history){
	const w=240,h=56,pad=3,n=history.length;
	if(n<2)return `<svg class="spark" viewBox="0 0 ${w} ${h}" aria-hidden="true"><line class="axis" x1="0" x2="${w}" y1="${h-pad}" y2="${h-pad}" stroke="currentColor" opacity=".2"/></svg>`;
	const max=Math.max(1,...history.map(p=>Math.max(p.down,p.up)));
	const x=i=>(i*(w/(history.length-1))).toFixed(1),y=v=>(h-pad-(v/max)*(h-pad*2)).toFixed(1);
	const line=k=>history.map((p,i)=>`${x(i)},${y(p[k])}`).join(' ');
	return `<svg class="spark" viewBox="0 0 ${w} ${h}" preserveAspectRatio="none" role="img" aria-label="${t('networkSpeed')}"><polygon class="area" points="0,${h} ${line('down')} ${w},${h}"/><polyline class="down" points="${line('down')}"/><polyline class="up" points="${line('up')}"/></svg>`;
}

/**
 * networkSpeed shows live rates, running totals and a rolling rate chart.
 * Samples are collected by app.js into store.speedHistory while connected.
 */
export function networkSpeed(){
	const s=store.snapshot,m=s.traffic,live=s.state==='connected',valid=live&&m&&Date.now()-m.sampledAt<4000;
	const rate=k=>!live?'0 B/s':valid?formatBytes(m[k])+'/s':'—';
	return `<section class="home-module" aria-labelledby="speed-title"><div class="module-heading"><h2 id="speed-title">${t('networkSpeed')}</h2><span class="module-caption">${t(s.preview?'simulatedTraffic':'proxyTraffic')}</span></div>
	<div class="speed-values"><div><span>${t('downloadSpeed')}</span><strong>${rate('downloadRate')}</strong><small>${t('downloadTotal')} ${m?formatBytes(m.downloaded):'—'}</small></div><div><span>${t('uploadSpeed')}</span><strong>${rate('uploadRate')}</strong><small>${t('uploadTotal')} ${m?formatBytes(m.uploaded):'—'}</small></div></div>
	${sparkline(live?store.speedHistory:[])}<div class="speed-legend"><span><i></i>${t('downloadSpeed')}</span><span><i class="up"></i>${t('uploadSpeed')}</span></div>
	${live&&!valid?`<p class="hint">${t('waitingSample')}</p>`:''}</section>`;
}
