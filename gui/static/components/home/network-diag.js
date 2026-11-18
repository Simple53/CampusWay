import {t} from '../../core/i18n.js';
import {store,selected,active} from '../../core/store.js';
import {esc,icon} from '../../core/ui.js';

export function getTemplateSites(p){
	if(!p) return [
		{name: 'Apple', url: 'https://www.apple.com'},
		{name: 'GitHub', url: 'https://github.com'},
		{name: 'Google', url: 'https://www.google.com'},
		{name: '知网 (CNKI)', url: 'https://www.cnki.net'}
	];
	if(p.customDomains){
		const domains=p.customDomains.split(/[,;\n\s]+/).filter(Boolean);
		if(domains.length){
			return domains.slice(0, 4).map(d => {
				const title = d.includes('.') ? (d.split('.')[0].toUpperCase()) : d;
				return {name: title, url: 'https://' + d};
			});
		}
	}
	const s = (p.server + ' ' + p.name).toLowerCase();
	if(s.includes('zju') || s.includes('浙大')){
		return [
			{name: '浙大办公网', url: 'https://service.zju.edu.cn'},
			{name: '教务系统', url: 'http://zdbk.zju.edu.cn/'},
			{name: '图书资源', url: 'https://lib.zju.edu.cn'},
			{name: 'CC98', url: 'http://cc98.org/'}
		];
	}
	if(s.includes('nnu') || s.includes('njnu') || s.includes('南师')){
		return [
			{name: '办事大厅', url: 'https://ehallapp.nnu.edu.cn'},
			{name: '教务系统', url: 'http://jwgl.njnu.edu.cn'},
			{name: '知网 (CNKI)', url: 'https://www.cnki.net'},
			{name: '公共网关', url: 'https://www.baidu.com'}
		];
	}
	return [
		{name: 'Apple', url: 'https://www.apple.com'},
		{name: 'GitHub', url: 'https://github.com'},
		{name: 'Google', url: 'https://www.google.com'},
		{name: '知网 (CNKI)', url: 'https://www.cnki.net'}
	];
}

function siteLogo(name){
	const n=name.toLowerCase();
	if(n.includes('apple')) return `<svg class="site-icon" viewBox="0 0 170 170" width="28" height="28" fill="currentColor"><path d="M150.37 130.25c-2.45 5.66-5.35 10.87-8.71 15.66-4.58 6.53-8.33 11.05-11.22 13.56-4.48 4.12-9.28 6.23-14.42 6.35-3.69 0-8.14-1.05-13.32-3.18-5.19-2.12-9.97-3.17-14.34-3.17-4.58 0-9.49 1.05-14.74 3.17-5.26 2.13-9.5 3.24-12.74 3.35-4.35.13-9.16-1.9-14.42-6.08-3.7-3.04-7.58-7.7-11.64-13.98-5.77-8.87-10.37-18.91-13.8-30.13-3.43-11.22-5.15-21.75-5.15-31.59 0-14.58 3.73-26.43 11.19-35.54 7.46-9.11 16.71-13.78 27.75-14.02 4.58 0 9.87 1.25 15.88 3.75 6 2.5 9.77 3.82 11.3 3.96 1.85-.27 5.79-1.66 11.82-4.16 6.04-2.5 11.22-3.64 15.55-3.41 12.06.63 21.64 5.08 28.75 13.34-10.42 6.3-15.54 15.02-15.35 26.17.2 9.07 3.63 16.63 10.3 22.68 6.67 6.05 14.52 9.49 23.55 10.33-2.22 6.95-4.88 14.07-7.98 21.36zM119.22 31.84c0-7.23 2.65-13.96 7.96-20.19 5.31-6.23 11.86-10.15 19.64-11.75.14 1.13.21 2.14.21 3.03 0 7.15-2.8 14.02-8.39 20.61-5.59 6.59-12.06 10.43-19.42 11.53-.14-1.07-.21-2.14-.21-3.23z"/></svg>`;
	if(n.includes('git')) return `<svg class="site-icon" viewBox="0 0 98 96" width="28" height="28" fill="currentColor"><path fill-rule="evenodd" clip-rule="evenodd" d="M48.854 0C21.839 0 0 22 0 49.217c0 21.756 13.993 40.172 33.405 46.69 2.427.49 3.316-1.059 3.316-2.36 0-1.141-.08-5.052-.08-9.127-13.59 2.934-16.42-5.867-16.42-5.867-2.184-5.704-5.42-7.17-5.42-7.17-4.448-3.015.324-3.015.324-3.015 4.934.326 7.523 5.052 7.523 5.052 4.367 7.496 11.404 5.378 14.235 4.074.404-3.178 1.699-5.378 3.074-6.6-10.839-1.141-22.243-5.378-22.243-24.283 0-5.378 1.94-9.778 5.014-13.2-.485-1.222-2.184-6.275.486-13.038 0 0 4.125-1.304 13.426 5.052a46.97 46.97 0 0 1 12.215-1.63c4.125 0 8.33.571 12.213 1.63 9.302-6.356 13.427-5.052 13.427-5.052 2.67 6.763.97 11.816.485 13.038 3.155 3.422 5.015 7.822 5.015 13.2 0 18.905-11.404 23.06-22.324 24.283 1.78 1.548 3.316 4.481 3.316 9.126 0 6.6-.08 11.897-.08 13.526 0 1.304.89 2.853 3.316 2.364 19.412-6.52 33.405-24.935 33.405-46.691C97.707 22 75.788 0 48.854 0z"/></svg>`;
	if(n.includes('google')) return `<svg class="site-icon" viewBox="0 0 48 48" width="28" height="28"><path fill="#EA4335" d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"/><path fill="#4285F4" d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"/><path fill="#FBBC05" d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"/><path fill="#34A853" d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"/></svg>`;
	return `<svg class="site-icon" viewBox="0 0 24 24" width="28" height="28" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path></svg>`;
}

export function websiteURL(value) {
	try {
		const u = new URL(value);
		if ((u.protocol === 'https:' || u.protocol === 'http:') && !u.username && !u.password) return u.href;
	} catch {}
	return '';
}

export function networkDiag(){
	const {diagReport,diagTesting,snapshot}=store;
	const links=snapshot.preferences.websiteLinks?.length?snapshot.preferences.websiteLinks:getTemplateSites(active()||selected());
	return `<section class="home-module diag-module" aria-labelledby="diag-title">
		<div class="module-heading">
			<h2 id="diag-title">${t('websiteLinks')}</h2>
			<div class="website-actions">
				<button type="button" class="icon-btn quiet btn-sm" data-action="edit-links" title="${t('editLinks')}" aria-label="${t('editLinks')}">${icon('pencil')}</button>
				<button type="button" class="icon-btn quiet btn-sm ${diagTesting?'spin':''}" data-action="run-diag" title="${t('testNow')}" aria-label="${t('testNow')}" ${diagTesting?'disabled':''}>${icon(diagTesting?'refresh':'zap')}</button>
			</div>
		</div>
		<p class="hint website-hint">${t('linkActions')}</p>
		<div class="site-test-grid">
			${links.map(item => {
				const url=websiteURL(item.url);
				if(!url)return '';
				const report=diagReport?.targets?.find(x=>websiteURL(x.url)===url);
				const result=report&&!report.skipped;
				const label=item.name||new URL(url).hostname;
				return `<button type="button" class="site-card" data-action="website-link" data-url="${esc(url)}" title="${esc(url)}">
					<span class="site-card-icon">${siteLogo(label)}</span>
					<span class="site-card-name">${esc(label)}</span>
					${result?`<span class="site-card-status ${report.ok?'ok':'err'}">${report.ok?`${report.ms} ms`:t('error')}</span>`:''}
				</button>`;
			}).join('')}
		</div>
	</section>`;
}
