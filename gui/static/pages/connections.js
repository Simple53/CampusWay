import {t} from '../core/i18n.js';
import {store} from '../core/store.js';
import {esc,icon,button,toggleSwitch} from '../core/ui.js';

export function routeState(p){const s=store.snapshot;if(p.id===s.active)return s.state;if(s.state==='error'&&p.id===(store.failedID||s.selected))return 'error';return 'disconnected';}

/**
 * indicator renders connection state badge (colored dot + status text) matching the reference design.
 */
export function indicator(p){
	const state = routeState(p);
	return `<span class="route-status-tag ${state}">
		<span class="route-dot ${state}" aria-hidden="true"></span>
		<span class="route-status-text">${t(state)}</span>
	</span>`;
}

function routeToggle(p){
 const on=p.id===store.snapshot.active&&store.snapshot.state!=='disconnecting';
 return toggleSwitch(on,'connect-route',`data-id="${p.id}" aria-label="${t(on?'disconnect':'connect')} · ${esc(p.name)}" ${store.snapshot.state==='disconnecting'?'disabled':''}`);
}

function editIcon(p){
	return `<button type="button" class="route-edit" data-action="edit-route" data-id="${p.id}" aria-label="${t('edit')} · ${esc(p.name)}" title="${t('edit')}">${icon('pencil')}</button>`;
}

export function connections(){
	const s=store.snapshot;
	return `<div class="page-heading">
		<h1>${t('connections')}</h1>
		<button type="button" class="icon-btn primary add-btn" data-action="add" aria-label="${t('add')}" title="${t('add')}">${icon('plus')}</button>
	</div>${!s.profiles.length?`<div class="empty">${icon('server')}<h2>${t('emptyTitle')}</h2><p>${t('emptyBody')}</p></div>`:`<div class="list-toolbar"><label class="search">${icon('search')}<input id="search" type="search" value="${esc(store.query)}" placeholder="${t('search')}" aria-label="${t('search')}"></label><div class="segmented">${['grid','table'].map(v=>button(v,'view-'+v,s.preferences.view===v?'chosen':'',`aria-pressed="${s.preferences.view===v}"`)).join('')}</div></div><div id="route-list">${routes()}</div>`}`;
}

export function routes(){
	const s=store.snapshot;
	let profiles=s.profiles.filter(p=>(p.name+' '+p.server+' '+p.protocol).toLowerCase().includes(store.query.toLowerCase()));
	if(s.preferences.sort==='name')profiles.sort((a,b)=>a.name.localeCompare(b.name));
	if(!profiles.length)return `<p class="empty">${t('noMatch')}</p>`;

	const entry=p=>`<strong>${esc(p.name)}</strong><small>${esc(p.server)}</small>`;

	if(s.preferences.view==='table'){
		return `<table class="routes-table">
			<thead><tr><th>${t('route')}</th><th>${t('protocol')}</th><th>${t('currentStatus')}</th><th><span class="sr-only">${t('edit')}</span></th></tr></thead>
			<tbody>${profiles.map(p=>`<tr data-route-connect="${esc(p.id)}" class="${p.id===s.selected?'selected':''}">
				<td>${entry(p)}</td>
				<td>${p.protocol==='atrust'?'aTrust':'EasyConnect'}</td>
				<td>${indicator(p)}</td>
				<td><div class="route-actions">${routeToggle(p)}${editIcon(p)}</div></td>
			</tr>`).join('')}</tbody>
		</table>`;
	}

	return `<div class="route-grid">${profiles.map(p=>`
		<article data-route-connect="${esc(p.id)}" class="route-tile ${p.id===s.selected?'selected':''}">
			<div class="route-card">
				<div class="route-card-top">
					${icon('server')}
					${indicator(p)}
				</div>
				<div class="route-card-body">
					<strong>${esc(p.name)}</strong>
					<small>${esc(p.server)}</small>
				</div>
				<div class="route-card-foot">
					<span>${p.protocol==='atrust'?'aTrust':'EasyConnect'}</span>
					<div class="route-actions">${routeToggle(p)}${editIcon(p)}</div>
				</div>
			</div>
		</article>
	`).join('')}</div>`;
}

