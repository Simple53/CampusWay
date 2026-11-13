from pathlib import Path
import json,hashlib
root=Path('..')/'.references'
result=[]
for name in ['EZ4Connect','ZJU-Connect-for-Windows']:
 d=root/name; tree=json.loads((d/'tree.json').read_text(encoding='utf-8-sig')); entries={i['path']:i.get('sha') for i in tree['tree']}
 files=[]
 for p in d.rglob('*'):
  if p.is_file() and p.suffix in ('.h','.cpp','.ui'):
   b=p.read_bytes();sha=hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest();path=p.relative_to(d).as_posix()
   if name=='ZJU-Connect-for-Windows' and path not in ['mainwindow.cpp','settingwindow/settingwindow.cpp','zjuconnectcontroller/zjuconnectcontroller.cpp']: continue
   files.append({'path':path,'sha':sha,'matchesTree':sha==entries.get(path)})
 result.append({'repository':name,'treeSHA':tree['sha'],'files':files})
Path('docs/redesign-v2/SOURCES.json').write_text(json.dumps(result,indent=2),encoding='utf-8')
p=Path('docs/redesign-v2/PLAN.md');s=p.read_text(encoding='utf-8-sig');s=s.replace('P0 源码研究完成，运行界面审查进行中','P0 源码研究及现版界面基线完成；P1 隔离预览工具已建立，架构拆分尚未开始');s=s.replace('- [ ] P0-C：','- [x] P0-C：');s+='\n执行记录：\n- P0：完成源码研究、计划和四步现版截图审查，见 BASELINE.md。\n- P1 第一步：新增 scripts/preview-ui.py，使用隔离空数据夹具；无账户读取、写入或联网连接能力。生产静态资源没有修改。\n- P1 后续：拆分 bridge/store/i18n 后再重建 shell，不能将预览工具完成标记为整阶段完成。\n- 上游源码完整性见 SOURCES.json；按 Git blob SHA 校验下载内容与已取回树，避免把浏览仓库首页当作读过实现。\n';p.write_text(s,encoding='utf-8')
print('Reference files verified:',sum(len(r['files']) for r in result),'matches:',all(f['matchesTree'] for r in result for f in r['files']))
