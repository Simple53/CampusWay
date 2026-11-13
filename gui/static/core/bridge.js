export async function call(method,...args){
 const api=window.go?.main?.AppServer;
 if(!api?.[method])throw new Error('offline');
 return api[method](...args);
}
