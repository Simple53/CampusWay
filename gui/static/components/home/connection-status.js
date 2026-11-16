import {t,message} from '../../core/i18n.js';
import {store,selected,active} from '../../core/store.js';
import {esc,icon,status,button} from '../../core/ui.js';

export function maskIP(ip, masked){
	if(!ip||ip==='—')return '—';
	if(!masked)return ip;
	const parts=ip.split('.');
	if(parts.length===4)return `${parts[0]}.${parts[1]}.***.***`;
	return ip.slice(0, Math.floor(ip.length/2)) + '***';
}

/**
 * connectionStatus renders the compact IP & Gateway Info card matching reference UI.
 */
export function connectionStatus(){
	const s = store.snapshot;
	const p = active() || selected();
	const r = store.diagReport;
	const isConnected = s.state === 'connected';
	const rawIP = r?.localIP || '127.0.0.1';
	const displayIP = maskIP(rawIP, store.ipMasked);

	return `<section class="home-module ip-info-module" aria-labelledby="ip-info-title">
		<div class="module-heading">
			<div style="display:flex;align-items:center;gap:8px;">
				<span class="icon-accent" style="color:var(--accent);">${icon('map-pin')}</span>
				<h2 id="ip-info-title">${t('ipInfo')}</h2>
			</div>
			<div style="display:flex;align-items:center;gap:8px;">
				${status(s.state)}
				<button type="button" class="icon-btn quiet btn-sm" data-action="run-diag" title="${t('check')}" aria-label="${t('check')}">${icon('refresh')}</button>
			</div>
		</div>
		${!p ? `
			<div class="welcome" style="padding:24px 12px;">
				<h3>${t('emptyTitle')}</h3>
				<p>${t('emptyBody')}</p>
				${button('add','add','primary')}
			</div>
		` : `
			<div class="ip-info-grid">
				<div class="ip-info-col">
					<div class="ip-status-headline">${isConnected ? (r?.fakeIP ? t('fakeIpHint') : t('vpnProxy')) : t('disconnected')}</div>
					<div class="ip-row">
						<span class="ip-label">IP:</span>
						<strong class="ip-val">${esc(displayIP)}</strong>
						<button type="button" class="icon-btn quiet ip-eye-btn" data-action="toggle-ip-mask" title="${store.ipMasked?'显示明文':'隐藏脱敏'}">${icon(store.ipMasked?'eye-off':'eye')}</button>
					</div>
					<div class="ip-row ip-sub">
						<span class="ip-label">${t('localProxy')}:</span>
						<span>127.0.0.1:${p.options?.socksPort||1080} / :${p.options?.httpPort||1081}</span>
					</div>
				</div>
				<div class="ip-info-col ip-details-col">
					<div class="ip-meta-item"><span>${t('gateway')}:</span> <strong>${esc(p.server)}</strong></div>
					<div class="ip-meta-item"><span>${t('protocol')}:</span> <strong>${p.protocol==='atrust'?'aTrust':'EasyConnect'}</strong></div>
					<div class="ip-meta-item"><span>${t('dnsServer')}:</span> <strong>${esc(r?.dns||p.options?.remoteDNS||'System')}</strong></div>
				</div>
			</div>
			${s.state==='error'?`<p class="error" role="alert" style="margin-top:10px;">${esc(message(s.message))}</p>`:''}
			<div class="ip-info-foot">
				<span class="hint">${t('autoRefresh')}: 15s</span>
			</div>
		`}
	</section>`;
}
