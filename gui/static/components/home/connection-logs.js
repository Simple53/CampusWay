import {t,message,clock} from '../../core/i18n.js';
import {store} from '../../core/store.js';
import {esc,icon} from '../../core/ui.js';

/**
 * connectionLogs 渲染主页面板中的终端式日志文本框。
 * 具备固定独立高度与内部滚动条，滚动时其他板块原地保留不被隐藏。
 */
export function connectionLogs(){
	const logs = store.snapshot.logs || [];
	return `<section class="home-module log-module" aria-labelledby="logs-title">
		<div class="module-heading">
			<div style="display:flex;align-items:center;gap:8px;">
				<h2 id="logs-title">${t('connectionLogs')}</h2>
				<span class="console-count">${logs.length}</span>
			</div>
			<div class="module-actions" style="display:flex;align-items:center;gap:6px;">
				<button type="button" class="icon-btn quiet" data-action="copy-logs" title="${t('copy')}" aria-label="${t('copy')}">${icon('copy')}</button>
			</div>
		</div>
		<div class="console-container" id="home-log-console">
			${logs.length ? `
				<div class="console-viewport">
					${logs.map(l=>`
						<div class="console-row level-${esc(l.level||'info')}">
							<span class="console-ts">${clock(l.time)}</span>
							<span class="log-level">${esc((l.level||'info').toUpperCase())}</span>
							<span class="console-text">${esc(message(l.message))}</span>
						</div>
					`).join('')}
				</div>
			` : `<div class="console-empty">${t('noLogs')}</div>`}
		</div>
	</section>`;
}
