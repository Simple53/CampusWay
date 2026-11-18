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
	const info = store.ipInfo;
	const renderAddress=(ip,family)=>`<strong class="ip-val">${esc(maskIP(ip,store.ipMasked))}</strong>${family?` <span class="hint">${esc(family)}</span>`:''}`;
	const tunnelIP=s.tunnelIP;

	return `<section class="home-module ip-info-module" aria-labelledby="ip-info-title">
		<div class="module-heading">
			<div style="display:flex;align-items:center;gap:8px;">
				<span class="icon-accent" style="color:var(--accent);">${icon('map-pin')}</span>
				<h2 id="ip-info-title">${t('ipInfo')}</h2>
			</div>
			<div style="display:flex;align-items:center;gap:8px;">
				${status(s.state)}
				<button type="button" class="icon-btn quiet ip-eye-btn" data-action="toggle-ip-mask" aria-label="${t('maskIP')}">${icon(store.ipMasked?'eye-off':'eye')}</button>
				<button type="button" class="icon-btn quiet btn-sm" data-action="refresh-ip" title="${t('refreshIP')} · ipapi.co" aria-label="${t('refreshIP')}" ${store.ipRefreshing?'disabled':''}>${icon('refresh')}</button>
			</div>
		</div>
		<div class="ip-info-grid">
			<div class="ip-meta-item">
				<span>${t('localIP')}</span>
				<div class="address-list">${store.localIPs?.length?store.localIPs.map(row=>`<div>${renderAddress(row.ip,row.family)}</div>`).join(''):'—'}</div>
			</div>
			<div class="ip-meta-item">
				<span>${t('tunnelIP')}</span>
				<div>${renderAddress(tunnelIP,tunnelIP?(tunnelIP.includes(':')?'IPv6':'IPv4'):'')}</div>
			</div>
			${info?.public?`<div class="ip-meta-item"><span>${t('publicIP')}</span><div>${renderAddress(info.ip,info.family)}</div></div>`:''}
				${info?.public?`<div class="ip-meta-item"><span>${t('ipRegion')}</span><strong>${esc([...new Set([info.country,info.region].filter(Boolean))].join(' · ')||t('ipRegionUnavailable'))}</strong></div>`:''}
		</div>
		${!p ? `
			<div class="welcome" style="padding:24px 12px;">
				<h3>${t('emptyTitle')}</h3>
				<p>${t('emptyBody')}</p>
				${button('add','add','primary')}
			</div>
		` : `
			<div class="ip-info-grid">
				<div class="ip-info-col ip-details-col">
					<div class="ip-meta-item"><span>${t('gateway')}:</span> <strong>${esc(p.server)}</strong></div>
					<div class="ip-meta-item"><span>${t('protocol')}:</span> <strong>${p.protocol==='atrust'?'aTrust':'EasyConnect'}</strong></div>
					<div class="ip-meta-item"><span>${t('dnsServer')}:</span> <strong>${esc(r?.dns||p.options?.remoteDNS||'System')}</strong></div>
					<div class="ip-meta-item"><span>${t('localProxy')}</span><strong>SOCKS ${p.options?.socksPort||1080} · HTTP ${p.options?.httpPort||1081}</strong></div>
				</div>
			</div>
			${s.state==='error'?`<p class="error" role="alert" style="margin-top:10px;">${esc(message(s.message))}</p>`:''}
			${isConnected&&r?.fakeIP?`<p class="hint">${t('fakeIpHint')}</p>`:''}
		`}
	</section>`;
}
