import {t} from '../core/i18n.js';
import {store} from '../core/store.js';
import {esc,icon,button} from '../core/ui.js';

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

function editIcon(p){
	return `<button type="button" class="route-edit" data-action="edit-route" data-id="${p.id}" aria-label="${t('edit')} · ${esc(p.name)}" title="${t('edit')}">${icon('pencil')}</button>`;
}

export function connections(){
	const s=store.snapshot;
	return `<div class="page-heading"><h1>${t('connections')}</h1>${button('add','add','primary')}</div>${!s.profiles.length?`<div class="empty">${icon('server')}<h2>${t('emptyTitle')}</h2><p>${t('emptyBody')}</p></div>`:`<div class="list-toolbar"><label class="search">${icon('search')}<input id="search" type="search" value="${esc(store.query)}" placeholder="${t('search')}" aria-label="${t('search')}"></label><div class="segmented">${['grid','table'].map(v=>button(v,'view-'+v,s.preferences.view===v?'chosen':'',`aria-pressed="${s.preferences.view===v}"`)).join('')}</div></div><div id="route-list">${routes()}</div>`}`;
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
			<tbody>${profiles.map(p=>`<tr class="${p.id===s.selected?'selected':''}">
				<td><button type="button" class="route-select" data-action="toggle-route" data-id="${p.id}" aria-pressed="${p.id===s.selected}">${entry(p)}</button></td>
				<td>${p.protocol==='atrust'?'aTrust':'EasyConnect'}</td>
				<td><button type="button" class="route-status-btn" data-action="toggle-route" data-id="${p.id}">${indicator(p)}</button></td>
				<td>${editIcon(p)}</td>
			</tr>`).join('')}</tbody>
		</table>`;
	}

	return `<div class="route-grid">${profiles.map(p=>`
		<article class="route-tile ${p.id===s.selected?'selected':''}">
			<div class="route-card" data-action="toggle-route" data-id="${p.id}" role="button" tabindex="0" aria-pressed="${p.id===s.selected}">
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
					${editIcon(p)}
				</div>
			</div>
		</article>
	`).join('')}</div>`;
}

