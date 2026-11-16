import {t} from '../../core/i18n.js';
import {store} from '../../core/store.js';
import {esc,glyph,iconButton} from '../../core/ui.js';

/**
 * diagChart draws latency as horizontal SVG bars: one row for the gateway
 * (probed directly) and one per target (probed through the tunnel).
 * Bars are scaled to the slowest successful probe; failures show a red stub.
 */
function chart(rows){
	const row=22,left=96,right=46,w=320,h=rows.length*row+4;
	const max=Math.max(100,...rows.filter(r=>r.ok).map(r=>r.ms));
	const bars=rows.map((r,i)=>{
		const y=i*row+2,len=r.ok?Math.max(4,Math.round((w-left-right)*r.ms/max)):(r.skipped?0:4);
		const cls=r.skipped?'idle':r.ok?'':'bad';
		const value=r.skipped?'—':r.ok?`${r.ms} ms`:esc(r.err||'×');
		return `<text class="lbl" x="0" y="${y+13}">${esc(r.label.slice(0,14))}</text><rect class="bar ${cls}" x="${left}" y="${y+3}" width="${len}" height="12" rx="3"/><text class="val" x="${left+len+6}" y="${y+13}">${value}</text>`;
	}).join('');
	return `<svg class="chart" viewBox="0 0 ${w} ${h}" role="img" aria-label="${t('networkCheck')}"><line class="axis" x1="${left}" x2="${left}" y1="0" y2="${h}"/>${bars}</svg>`;
}

/**
 * networkDiag renders the connectivity widget: an icon button to run the test,
 * a status strip (LAN IP, gateway resolution) and the latency chart.
 */
export function networkDiag(){
	const r=store.diagReport,testing=store.diagTesting;
	const rows=r?[{...r.gateway,label:t('gateway')},...r.targets.map(x=>({...x,label:x.name}))].filter(x=>x.label):[];
	const gwTone=!r?.gateway?.URL?'idle':r.gateway.ok?'ok':'bad';
	return `<section class="home-module diag-module" aria-labelledby="diag-title">
		<div class="module-heading"><h2 id="diag-title">${t('networkCheck')}</h2>
			<span class="module-actions">${iconButton(testing?'refresh':'zap','run-diag','testNow',testing?'disabled':'',false).replace('icon-btn','icon-btn '+(testing?'spin':''))}</span></div>
		${r?.fakeIP?`<div class="diag-banner" role="alert">${glyph('alert','warn')}<span>${t('fakeIpHint')}</span></div>`:''}
		<div class="diag-head">
			<span>${glyph('server',gwTone)}${t('gateway')} <b>${esc(r?.gatewayIP||'—')}</b></span>
			<span>${glyph('route','idle')}${t('localIP')} <b>${esc(r?.localIP||'—')}</b></span>
			<span>${glyph(r?.connected?'check':'minus',r?.connected?'ok':'idle')}${t(r?.connected?'connected':'disconnected')}</span>
		</div>
		${rows.length?chart(rows):`<p class="hint">${glyph('zap','idle')} ${t('testHint')}</p>`}
	</section>`;
}
