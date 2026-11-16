import {t,message,clock} from '../../core/i18n.js';
import {store} from '../../core/store.js';
import {esc,icon} from '../../core/ui.js';

/**
 * connectionLogs 渲染主页面板中的实时连接与接口日志流。
 * 展示底层认证接口、网关解析、路由添加、端口绑定与运行状态。
 */
export function connectionLogs(){
	const logs = store.snapshot.logs.slice(-6).reverse();
	return `<section class="home-module log-module" aria-labelledby="logs-title">
		<div class="module-heading">
			<h2 id="logs-title">${t('connectionLogs')}</h2>
			<a href="#settings/diagnostics">${t('allLogs')}${icon('arrow-right')}</a>
		</div>
		${logs.length ? `
			<ol class="event-list" style="max-height:260px;overflow-y:auto;font-family:var(--font-mono, 'Consolas', monospace);font-size:12px;">
				${logs.map(l=>`
					<li>
						<time style="flex-shrink:0;">${clock(l.time)}</time>
						<span style="word-break:break-all;user-select:text;">${esc(message(l.message))}</span>
					</li>
				`).join('')}
			</ol>
		` : `<p class="module-empty">${t('noLogs')}</p>`}
	</section>`;
}
