import {t} from '../core/i18n.js';
import {store} from '../core/store.js';
import {esc,icon,button,toggleSwitch} from '../core/ui.js';

export function routeState(p){const s=store.snapshot;if(p.id===s.active)return s.state;if(s.state==='error'&&p.id===(store.failedID||s.selected))return 'error';return 'disconnected';}

/**
 * indicator renders connection state tag and interactive circular slide switch.
 */
function indicator(p){
	const s = store.snapshot;
	const isConnected = p.id === s.active && s.state === 'connected';
	const isBusy = (p.id === s.active || p.id === s.selected) && (s.state === 'connecting' || s.state === 'disconnecting' || s.state === 'awaiting_auth');

	return `<div class="route-switch-cell" style="display:inline-flex;align-items:center;gap:8px;">
		<span class="route-dot ${routeState(p)}" aria-hidden="true"></span>
		${toggleSwitch(isConnected, 'toggle-route', `data-id="${p.id}" ${isBusy ? 'disabled' : ''}`)}
	</div>`;
}

function editIcon(p){return `<button type="button" class="route-edit" data-action="edit-route" data-id="${p.id}" aria-label="${t('edit')} · ${esc(p.name)}" title="${t('edit')}">${icon('pencil')}</button>`;}

export function connections(){const s=store.snapshot;return `<div class="page-heading"><h1>${t('connections')}</h1>${button('add','add','primary')}</div>${!s.profiles.length?`<div class="empty">${icon('server')}<h2>${t('emptyTitle')}</h2><p>${t('emptyBody')}</p></div>`:`<div class="list-toolbar"><label class="search">${icon('search')}<input id="search" type="search" value="${esc(store.query)}" placeholder="${t('search')}" aria-label="${t('search')}"></label><div class="segmented">${['grid','table'].map(v=>button(v,'view-'+v,s.preferences.view===v?'chosen':'',`aria-pressed="${s.preferences.view===v}"`)).join('')}</div></div><p class="route-hint">${t('clickSwitch')}</p><div id="route-list">${routes()}</div>`}`;}

export function routes(){const s=store.snapshot;let profiles=s.profiles.filter(p=>(p.name+' '+p.server+' '+p.protocol).toLowerCase().includes(store.query.toLowerCase()));if(s.preferences.sort==='name')profiles.sort((a,b)=>a.name.localeCompare(b.name));if(!profiles.length)return `<p class="empty">${t('noMatch')}</p>`;const entry=p=>`<strong>${esc(p.name)}</strong><small>${esc(p.server)}</small>`;return s.preferences.view==='table'?`<table class="routes-table"><thead><tr><th>${t('route')}</th><th>${t('protocol')}</th><th>${t('currentStatus')}</th><th><span class="sr-only">${t('edit')}</span></th></tr></thead><tbody>${profiles.map(p=>`<tr class="${p.id===s.selected?'selected':''}"><td><button class="route-select" data-select="${p.id}" aria-pressed="${p.id===s.selected}">${entry(p)}</button></td><td>${p.protocol==='atrust'?'aTrust':'EasyConnect'}</td><td>${indicator(p)}</td><td>${editIcon(p)}</td></tr>`).join('')}</tbody></table>`:`<div class="route-grid">${profiles.map(p=>`<article class="route-tile ${p.id===s.selected?'selected':''}"><button class="route-card" data-select="${p.id}" aria-pressed="${p.id===s.selected}"><div>${icon('server')}${indicator(p)}</div>${entry(p)}<span>${p.protocol==='atrust'?'aTrust':'EasyConnect'}</span></button>${editIcon(p)}</article>`).join('')}</div>`;}

