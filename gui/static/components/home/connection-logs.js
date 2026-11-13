import {t,message,clock} from '../../core/i18n.js';
import {store} from '../../core/store.js';
import {esc,icon} from '../../core/ui.js';
export function connectionLogs(){const logs=store.snapshot.logs.slice(-3).reverse();return `<section class="home-module log-module" aria-labelledby="logs-title"><div class="module-heading"><h2 id="logs-title">${t('connectionLogs')}</h2><a href="#settings/diagnostics">${t('allLogs')}${icon('arrow-right')}</a></div>${logs.length?`<ol class="event-list">${logs.map(l=>`<li><time>${clock(l.time)}</time><span>${esc(message(l.message))}</span></li>`).join('')}</ol>`:`<p class="module-empty">${t('noLogs')}</p>`}</section>`;}
