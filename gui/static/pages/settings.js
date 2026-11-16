import {t,message,clock} from '../core/i18n.js';
import {accents} from '../core/themes.js';
import {store} from '../core/store.js';
import {esc,icon,button,select} from '../core/ui.js';

export const settingGroups=[
	['appearance','settings-2','appearanceHint'],
	['accounts','shield','accountsHint'],
	['data','server','dataHint'],
	['diagnostics','activity','diagnosticsHint'],
	['about','circle-help','']
];

/**
 * settings renders either the settings overview or dedicated sub-pages.
 * Appearance and layout now have their own dedicated page as requested.
 */
export function settings(section='',result=null){
	const s = store.snapshot, p = s.preferences;

	if(!section) {
		return `<div class="page-heading"><h1>${t('settings')}</h1></div>
		<div class="settings-groups">
			${settingGroups.map(([id,i,hint])=>`
				<a class="setting-row" href="#settings/${id}">
					${icon(i)}
					<span>
						<strong>${t(id)}</strong>
						${hint?`<small>${t(hint)}</small>`:''}
					</span>
					<span class="setting-value">
						${id==='appearance'?`${t(p.accent||'graphite')} · ${t(p.theme)}`:''}
					</span>
					${icon('arrow-right')}
				</a>
			`).join('')}
		</div>`;
	}

	let content = '';

	if(section==='appearance') {
		const currentWidgets = p.widgets || ['status', 'speed', 'diag', 'logs'];
		const isWidgetActive = w => currentWidgets.includes(w);

		content = `
		<section class="home-module">
			<div class="module-heading"><h2>${t('colorTheme')}</h2></div>
			<fieldset class="accent-options">
				<legend class="sr-only">${t('colorTheme')}</legend>
				<div class="accent-grid">
					${Object.keys(accents).map(key=>`
						<button type="button" class="accent-choice" data-accent="${key}" aria-pressed="${(p.accent||'graphite')===key}">
							<span class="accent-swatch" aria-hidden="true"></span>
							${t(key)}
						</button>
					`).join('')}
				</div>
			</fieldset>
		</section>

		<section class="home-module" style="margin-top:16px;">
			<div class="module-heading"><h2>${t('appearance')}</h2></div>
			<div class="preference-fields">
				${select('theme','theme',p.theme,[['system','system'],['light','light'],['dark','dark']])}
				${select('density','density',p.density,[['ultra-compact','ultraCompact'],['compact','compact'],['standard','standard'],['comfortable','comfortable']])}
				${select('language','language',p.language||'zh-CN',[['zh-CN','简体中文'],['en','English']])}
				${select('view','view',p.view,[['grid','grid'],['table','table']])}
				${select('sort','sort',p.sort,[['default','default'],['name','name']])}
				${select('closeBehavior','closeAction',p.closeAction||'minimize',[['minimize','closeToTray'],['exit','closeDirectly']])}
			</div>
		</section>

		<section class="home-module" style="margin-top:16px;">
			<div class="module-heading"><h2>${t('customizeWidgets')}</h2></div>
			<div class="preference-fields">
				<label class="check">
					<input type="checkbox" name="widget_status" data-widget="status" ${isWidgetActive('status')?'checked':''}>
					${t('showStatus')}
				</label>
				<label class="check">
					<input type="checkbox" name="widget_speed" data-widget="speed" ${isWidgetActive('speed')?'checked':''}>
					${t('showSpeed')}
				</label>
				<label class="check">
					<input type="checkbox" name="widget_diag" data-widget="diag" ${isWidgetActive('diag')?'checked':''}>
					${t('showDiag')}
				</label>
				<label class="check">
					<input type="checkbox" name="widget_logs" data-widget="logs" ${isWidgetActive('logs')?'checked':''}>
					${t('showLogs')}
				</label>
			</div>
		</section>
		`;
	}

	if(section==='accounts') content = s.profiles.length ? s.profiles.map(q=>`
		<section class="account-row">
			<div>
				<strong>${esc(q.name)}</strong>
				<p>${esc(q.username||'—')}</p>
				<small>${t(q.hasPassword?'saved':'notSaved')}</small>
			</div>
			<div class="actions">
				${button('edit','edit-account','',`data-id="${q.id}"`)}
				${q.hasPassword?button('forget','forget','quiet',`data-id="${q.id}"`):''}
			</div>
		</section>`).join('') : `<p class="empty">${t('noAccounts')}</p>`;

	if(section==='data') content = `
		<p class="section-description">${t('exportHint')}</p>
		<div class="data-actions">
			${button('export','export','primary')}
			<label class="file-button">${t('import')}<input id="import-file" type="file" accept=".json,application/json"></label>
		</div>`;

	if(section==='about') content = `
		<div class="about-brand">CampusWay<span>${t('version')} · ${esc(result?.current||'0.1.2')}</span></div>
		<p class="section-description">${t('releaseSource')}</p>
		<div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap;">
			${button('check','check')}
			${result?.latest && result?.status==='A different release is available' ? button('updateNow','apply-update','primary') : ''}
		</div>
		<p role="status" style="margin-top:12px;">${result?esc(message(result.status)+(result.latest?' ('+result.latest+')':'')):''}</p>`;

	if(section==='diagnostics') content = `
		<p class="section-description">${t('logsHint')}</p>
		<div class="diagnostic-toolbar">
			${button('copy','copy-logs')}
			${button('download','export-logs')}
			${button('exportDailyLogs','export-daily-logs','primary')}
			${button('openLogsFolder','open-logs-folder')}
		</div>
		${s.logs.length?`<ol class="event-list" style="font-family:var(--font-mono, 'Consolas', monospace);font-size:12px;">${s.logs.slice().reverse().map(l=>`<li><time style="flex-shrink:0;">${clock(l.time)}</time><span style="word-break:break-all;user-select:text;">${esc(message(l.message))}</span></li>`).join('')}</ol>`:`<div class="empty">${t('noLogs')}</div>`}`;

	return `<a class="back-link" href="#settings">${icon('arrow-right')} ${t('back')}</a>
		<div class="page-heading"><h1>${t(section)}</h1></div>
		${content}`;
}

