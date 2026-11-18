import {t,message,clock} from '../core/i18n.js';
import {accents} from '../core/themes.js';
import {store} from '../core/store.js';
import {esc,icon,button,select,field} from '../core/ui.js';

export const settingGroups=[
	['appearance','settings-2','appearanceHint'],
	['data','server','dataHint'],
	['advanced','route','dnsSettingsHint'],
	['diagnostics','activity','diagnosticsHint'],
	['about','circle-help','']
];

/**
 * settings renders either the settings overview or dedicated sub-pages.
 * Appearance and layout now have their own dedicated page as requested.
 */
export function settings(section='',result=null){
	const s = store.snapshot, p = s.preferences;

	if(!settingGroups.some(([id])=>id===section)) {
		return `<div class="page-heading"><h1>${t('settings')}</h1></div>
		<div class="settings-page settings-groups">
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
		<section class="settings-section">
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

		<section class="settings-section">
			<div class="module-heading"><h2>${t('appearance')}</h2></div>
			<div class="preference-fields">
				<div class="field theme-field"><span>${t('theme')}</span><div class="segmented" role="group" aria-label="${t('theme')}">${['system','light','dark'].map(mode=>button(mode,'theme-mode',p.theme===mode?'chosen':'',`aria-pressed="${p.theme===mode}" data-theme-mode="${mode}"`)).join('')}</div></div>
				${select('density','density',p.density,[['ultra-compact','ultraCompact'],['compact','compact'],['standard','standard'],['comfortable','comfortable']])}
				${select('language','language',p.language||'zh-CN',[['zh-CN','简体中文'],['en','English']])}
				${select('view','view',p.view,[['grid','grid'],['table','table']])}
				${select('sort','sort',p.sort,[['default','default'],['name','name']])}
				${select('closeBehavior','closeAction',p.closeAction||'minimize',[['minimize','closeToTray'],['exit','closeDirectly']])}
			</div>
		</section>

		<section class="settings-section">
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

	if(section==='advanced'){
		const dns=p.dns||{};
		content=`<form id="dns-settings">
			<p class="dns-warning">${t('dnsWarning')}</p>
			<label class="check"><input name="enabled" type="checkbox" ${dns.enabled?'checked':''}>${t('enableDNSOverrides')}</label>
			<div class="preference-fields dns-fields">
				${field('remoteDNS','remote',dns.remote||'','text','placeholder="auto" autocomplete="off"')}
				${field('fallbackDNS','secondary',dns.secondary||'','text','placeholder="1.1.1.1" autocomplete="off"')}
				${field('dnsTTL','ttl',dns.ttl||'','number','min="1" max="86400" placeholder="300"')}
			</div>
			<label class="field"><span>${t('dnsRecords')}</span><textarea rows="4" name="records" maxlength="8192" placeholder="host.example.org 192.0.2.10">${esc(dns.records||'')}</textarea></label>
			<p class="hint">${t('dnsRecordHint')}</p>
			<button class="primary" type="submit">${t('save')}</button>
		</form>`;
	}

	if(section==='data') content = `
		<p class="section-description">${t('exportHint')}</p>
		<div class="data-actions">
			${button('export','export','primary')}
			<label class="file-button">${t('import')}<input id="import-file" type="file" accept=".json,application/json"></label>
		</div>`;

	if(section==='about') content = `
		<div class="about-brand">RouteDesk<span>${t('version')} · ${esc(result?.current||s.version||'dev')}</span></div>
		<label class="check update-preference"><input type="checkbox" name="autoUpdate" ${p.autoUpdate!==false?'checked':''}>${t('autoUpdate')}</label>
		<div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap;">
			${button('check','check')}
			${result?.latest && result?.status==='A different release is available' ? button(result.install==='true'?'updateNow':'openRelease','apply-update','primary') : ''}
		</div>
		<p role="status" style="margin-top:12px;">${result?esc(message(result.status)+(result.latest?' ('+result.latest+')':'')):''}</p>`;

	if(section==='diagnostics') content = `
		<div class="diagnostic-toolbar">
			${button('copy','copy-logs')}
			${button('download','export-logs')}
			${button('exportDailyLogs','export-daily-logs')}
			${button('openLogsFolder','open-logs-folder')}
		</div>
		${s.logs.length?`<ol class="event-list diagnostic-events">${s.logs.slice().reverse().map(l=>`<li><time>${clock(l.time)}</time><span class="log-level level-${esc(l.level||'info')}">${esc((l.level||'info').toUpperCase())}</span><span class="diagnostic-message">${esc(message(l.message))}</span></li>`).join('')}</ol>`:`<div class="empty">${t('noLogs')}</div>`}`;

	return `<div class="settings-page"><a class="back-link" href="#settings">${icon('arrow-right')} ${t('back')}</a>
		<div class="page-heading"><h1>${t(section)}</h1></div>
		${content}</div>`;
}

