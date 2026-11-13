import subprocess,json
exe='dist/campusway/CampusWay.exe'
p=subprocess.Popen([exe,'--engine'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
request={'Profile':{'name':'Smoke test','server':'127.0.0.1','port':1,'protocol':'easyconnect','username':'smoke-test'},'Password':'CAMPUSWAY_SECRET_SENTINEL'}
output,err=p.communicate(json.dumps(request)+'\n'+json.dumps({'Kind':'stop'})+'\n',timeout=20)
assert 'CAMPUSWAY_SECRET_SENTINEL' not in output+err
assert p.returncode in (0,1), (p.returncode,output,err)
print('Engine cancellation and secret-output smoke check passed; exit',p.returncode)
