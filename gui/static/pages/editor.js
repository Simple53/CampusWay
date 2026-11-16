import {t} from '../core/i18n.js';
import {store} from '../core/store.js';
import {esc,field,select,button} from '../core/ui.js';

/**
 * editor 渲染线路与账户凭据编辑页面：
 * 1. 采用 novalidate 避免 Chromium 在折叠面板或隐藏项上的原生校验静默拦截；
 * 2. 记住密码默认开启（p.remember !== false），杜绝用户输入密码后未勾选导致凭据丢失；
 * 3. 保存按钮带有显式 data-action="save-profile"，兼顾原生表单回车与按钮直接单击交互。
 */
export function editor(p={}){
	const o=p.options||{},s=store.snapshot;
	return `<form id="editor" data-id="${p.id||''}" novalidate>
		<div class="page-heading">
			<h1>${t(p.id?'edit':'add')}</h1>
			${button('cancel','goto-connections','quiet')}
		</div>
		<section class="form-section">
			<h2>${t('basic')}</h2>
			${field('name','name',p.name||'','text','maxlength="80" placeholder="例如：校内主线路"')}
			${select('protocol','protocol',p.protocol||'atrust',[['atrust','aTrust'],['easyconnect','EasyConnect']])}
			<div class="field-pair">
				${field('gateway','server',p.server||'','text','maxlength="253" spellcheck="false" placeholder="例如：vpn.nnu.edu.cn"')}
				${field('port','port',p.port||443,'number','min="1" max="65535"')}
			</div>
		</section>
		<section class="form-section">
			<h2>${t('auth')}</h2>
			<div data-atrust>
				${select('auth','authType',o.authType||'auth/psw',[['auth/psw','passwordAuth'],['auth/smsCheckCode','smsAuth']])}
				${field('domain','loginDomain',o.loginDomain||'')}
			</div>
			${field('account','username',p.username||'','text','autocomplete="off" placeholder="请输入您的校园卡号或账号"')}
			<div data-password>
				${field('password','password','','password',`autocomplete="new-password" placeholder="${t(p.hasPassword?'saved':'password')}"`)}
				<label class="check">
					<input type="checkbox" name="remember" ${p.remember!==false?'checked':''} ${s.canRemember?'':'disabled'}>
					${t('remember')}
				</label>
			</div>
			<p class="hint">${t('passwordHint')}</p>
		</section>
		<details class="advanced">
			<summary>${t('advanced')}</summary>
			<section class="form-section">
				<h2>${t('proxy')}</h2>
				<div class="field-pair equal">
					${field('socksPort','socksPort',o.socksPort||1080,'number','min="1" max="65535"')}
					${field('httpPort','httpPort',o.httpPort||1081,'number','min="1" max="65535"')}
				</div>
				<label class="field">
					<span>${t('customDomains')}</span>
					<textarea name="customDomains" maxlength="8192">${esc(p.customDomains||'')}</textarea>
				</label>
			</section>
			<section class="form-section">
				<h2>${t('dns')}</h2>
				<div class="field-pair equal">
					${field('remoteDNS','remoteDNS',o.remoteDNS||'auto')}
					${field('fallbackDNS','secondaryDNS',o.secondaryDNS||'1.1.1.1')}
				</div>
				${field('dnsTTL','dnsTTL',o.dnsTTL||300,'number','min="1" max="86400"')}
				${check('disableRemoteDNS',o)}
			</section>
			<section class="form-section">
				<h2>${t('behavior')}</h2>
				${check('tcpOnly',o)}
				<div data-easy>
					${['disableMultiLine','disableServerConfig','skipDomainResource','proxyAll'].map(k=>check(k,o)).join('')}
				</div>
				<div data-atrust>
					${field('nodeInterval','updateInterval',o.updateInterval||300,'number','min="10" max="86400"')}
				</div>
			</section>
		</details>
		<div class="form-footer">
			${p.id?button('remove','delete','danger'):''}
			<span></span>
			${button('cancel','goto-connections','quiet')}
			<button type="submit" class="primary" data-action="save-profile">${t('save')}</button>
		</div>
	</form>`;
}

function check(k,o){
	return `<label class="check"><input type="checkbox" name="${k}" ${o[k]?'checked':''}>${t(k)}</label>`;
}

/**
 * syncProtocol 响应协议（aTrust / EasyConnect）与认证方式变化，
 * 动态显示/隐藏专属表单字段，并同步禁用隐藏项以防止干扰。
 */
export function syncProtocol(){
	const form=document.querySelector('#editor');
	if(!form)return;
	const atrust=form.elements.protocol.value==='atrust',sms=atrust&&form.elements.authType.value==='auth/smsCheckCode';
	form.querySelectorAll('[data-atrust]').forEach(e=>{
		e.hidden=!atrust;
		e.querySelectorAll('input,select').forEach(inp=>inp.disabled=!atrust);
	});
	form.querySelectorAll('[data-easy]').forEach(e=>{
		e.hidden=atrust;
		e.querySelectorAll('input,select').forEach(inp=>inp.disabled=atrust);
	});
	const pwBlock=form.querySelector('[data-password]');
	if(pwBlock){
		pwBlock.hidden=sms;
		pwBlock.querySelectorAll('input').forEach(inp=>inp.disabled=sms);
	}
	const userLabel=form.querySelector('[name=username]')?.previousElementSibling;
	if(userLabel) userLabel.textContent=t(sms?'phone':'account');
}

/**
 * readProfile 读取表单输入并清洗数据：
 * 1. 自动剥离网关中可能误贴的 https://、http://、尾部斜杠与嵌入端口；
 * 2. 只要用户填写了密码或勾选了记住密码，即永久标记 remember=true；
 * 3. 所有数值与字符串属性均赋予完备的缺省值，杜绝后端校验异常。
 */
export function readProfile(form){
	const d=new FormData(form);
	let server=(d.get('server')||'').trim();
	let port=Number(d.get('port'))||443;

	// 自动去除 URL 协议头与多余斜杠
	server=server.replace(/^[a-zA-Z]+:\/\//,'').replace(/\/.*$/,'').trim();
	if(server.includes(':')){
		const parts=server.split(':');
		server=parts[0].trim();
		const pNum=Number(parts[1]);
		if(pNum>0&&pNum<=65535) port=pNum;
	}

	const username=(d.get('username')||'').trim();
	const passwordInput=form.elements.password;
	const passwordVal=(passwordInput?passwordInput.value:'')||'';
	const rememberChecked=d.has('remember');
	// 填写了新密码或明确勾选记住密码时，自动持久化
	const shouldRemember=passwordVal.length>0||rememberChecked;

	const p={
		id:form.dataset.id||'',
		name:(d.get('name')||'').trim()||server||'校内网关',
		protocol:d.get('protocol')||'atrust',
		server:server,
		port:port,
		username:username,
		remember:shouldRemember,
		customDomains:(d.get('customDomains')||'').trim(),
		hasPassword:false,
		options:{}
	};

	p.options.authType=d.get('authType')||'auth/psw';
	p.options.loginDomain=(d.get('loginDomain')||'').trim();
	p.options.socksPort=Number(d.get('socksPort'))||1080;
	p.options.httpPort=Number(d.get('httpPort'))||1081;
	p.options.remoteDNS=(d.get('remoteDNS')||'auto').trim()||'auto';
	p.options.secondaryDNS=(d.get('secondaryDNS')||'1.1.1.1').trim()||'1.1.1.1';
	p.options.dnsTTL=Number(d.get('dnsTTL'))||300;
	p.options.updateInterval=Number(d.get('updateInterval'))||300;

	for(const k of ['tcpOnly','disableRemoteDNS','disableMultiLine','disableServerConfig','skipDomainResource','proxyAll']){
		p.options[k]=d.has(k);
	}
	if(p.protocol==='easyconnect'){
		p.options.authType='auth/psw';
		p.options.loginDomain='';
	}
	return p;
}
