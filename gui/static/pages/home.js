import {t} from '../core/i18n.js';
import {store} from '../core/store.js';
import {icon, iconButton} from '../core/ui.js';
import {connectionStatus} from '../components/home/connection-status.js';
import {networkSpeed} from '../components/home/network-speed.js';
import {networkDiag} from '../components/home/network-diag.js';
import {connectionLogs} from '../components/home/connection-logs.js';

// 缓存已渲染的组件 HTML，避免无数据变动时的 DOM 抖动
let renderedModules = {};

/**
 * modules 生成当前所有可用首页功能组件的 HTML 字典。
 * 包括：连接状态、实时图表测速、网络连通性诊断与最近日志。
 */
function modules() {
	return {
		'home-status': connectionStatus(),
		'home-speed': networkSpeed(),
		'home-diag': networkDiag(),
		'home-logs': connectionLogs()
	};
}

/**
 * home 渲染首页主仪表板视图：
 * 1. 标题栏右侧配有铅笔 SVG 按钮，用于进入/退出首页组件编辑模式；
 * 2. 编辑模式下显示添加隐藏组件栏，并在每个卡片右上角提供上移、下移、隐藏按钮；
 * 3. 严格遵循用户偏好设置（store.snapshot.preferences.widgets）决定展示组件及其排布顺序；
 * 4. 内层各功能容器保留专属 id（如 home-speed、home-diag），确保定时刷新时仅局部差量更新。
 */
export function home() {
	renderedModules = modules();
	const activeWidgets = store.snapshot.preferences.widgets || ['status', 'speed', 'diag', 'logs'];
	const widgetSizes = store.snapshot.preferences.widgetSizes || {};
	const widgetMap = {
		'status': 'home-status',
		'speed': 'home-speed',
		'diag': 'home-diag',
		'logs': 'home-logs'
	};

	// 循环生成活跃组件卡片，并附加编辑模式工具条与尺寸调整
	const renderedList = activeWidgets
		.map((w, index) => {
			const id = widgetMap[w];
			if (!id || !renderedModules[id]) return '';
			const isFirst = index === 0;
			const isLast = index === activeWidgets.length - 1;

			// 计算当前组件大小，默认 diag 和 logs 为 full，其余为 half
			const defaultSize = (w === 'diag' || w === 'logs') ? 'full' : 'half';
			const currentSize = widgetSizes[w] || defaultSize;
			const isFull = currentSize === 'full';

			// 工具栏：包括大小切换按钮（全宽/半宽），编辑模式下还包含上移、下移、隐藏
			const sizeBtn = iconButton(isFull ? 'minimize-2' : 'maximize-2', 'toggle-widget-size', isFull ? 'shrinkWidget' : 'expandWidget', `data-widget="${w}"`);
			const tools = store.homeEdit ? `
				<div class="widget-tools">
					${sizeBtn}
					${iconButton('arrow-up', 'move-widget-up', 'moveUp', `data-widget="${w}" ${isFirst ? 'disabled' : ''}`)}
					${iconButton('arrow-down', 'move-widget-down', 'moveDown', `data-widget="${w}" ${isLast ? 'disabled' : ''}`)}
					${iconButton('eye-off', 'hide-widget', 'hideWidget', `data-widget="${w}"`)}
				</div>
			` : `
				<div class="widget-tools hover-only">
					${sizeBtn}
				</div>
			`;

			return `
				<div class="widget-shell size-${currentSize} ${store.homeEdit ? 'editing' : ''}" draggable="true" data-widget="${w}">
					${tools}
					<div id="${id}">${renderedModules[id]}</div>
				</div>
			`;
		})
		.filter(Boolean)
		.join('');

	// 找出当前被隐藏的组件，方便在编辑栏中一键添加还原
	const allWidgets = ['status', 'speed', 'diag', 'logs'];
	const hiddenWidgets = allWidgets.filter(w => !activeWidgets.includes(w));

	// 编辑模式提示与组件添加栏
	const editBar = store.homeEdit ? `
		<div class="home-edit-bar">
			<span style="display:flex;align-items:center;gap:6px;font-size:12px;color:var(--muted);margin-right:8px;">
				${icon('pencil')} ${t('customizeWidgets')}:
			</span>
			<span style="font-size:11px;color:var(--muted);margin-right:8px;">${t('dragHint')}</span>
			${hiddenWidgets.length ? hiddenWidgets.map(w => `
				<button type="button" class="chip" data-action="add-widget" data-widget="${w}">
					${icon('plus')} ${t(w)}
				</button>
			`).join('') : `<span class="hint" style="align-self:center;">${t('allLogs')}</span>`}
		</div>
	` : '';

	return `<section class="home-panel">
		<div class="page-heading">
			<h1>${t('home')}</h1>
			<div style="display:flex;align-items:center;gap:8px;">
				${iconButton('pencil', 'toggle-home-edit', 'customizeWidgets', '', store.homeEdit)}
			</div>
		</div>
		${editBar}
		<div class="home-grid">
			${renderedList || `<div class="empty" style="grid-column:1/-1;padding:32px 12px;"><p>${t('customizeWidgets')}</p></div>`}
		</div>
		${store.snapshot.preview ? `<p class="simulation-note">${t('previewHint')}</p>` : ''}
	</section>`;
}

/**
 * updateHome 周期性局部刷新首页组件：
 * 保持用户当前的焦点和计时器状态，仅在流量或连通性诊断产生数据差异时更新对应模块的 innerHTML。
 */
export function updateHome() {
	for (const [id, html] of Object.entries(modules())) {
		const e = document.getElementById(id);
		if (e && renderedModules[id] !== html) {
			e.innerHTML = html;
			renderedModules[id] = html;
		}
	}
}

