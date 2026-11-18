// Accent palettes remain independent of the user's light/dark/system preference.
export const accents={
 graphite:{light:['#202020','#ffffff','#f1f1f1'],dark:['#eeeeee','#202020','#303030']},
 ocean:{light:['#245ca8','#ffffff','#edf3fc'],dark:['#91b5f4','#17202b','#243347']},
 jade:{light:['#24634a','#ffffff','#edf6f0'],dark:['#7fc9a3','#15251c','#213a2d']},
 violet:{light:['#6b43a8','#ffffff','#f3effa'],dark:['#bd9cec','#241b30','#352b45']},
 rose:{light:['#a63661','#ffffff','#fbeff3'],dark:['#eda1ba','#2b1921','#432a35']},
 amber:{light:['#825a0e','#ffffff','#faf3e5'],dark:['#e1bc6c','#292213','#3b3322']}
};
export function palette(accent,mode){return (accents[accent]||accents.graphite)[mode==='dark'?'dark':'light'];}
export function applyAppearance(preferences){
 const next=['system','light','dark'].includes(preferences.theme)?preferences.theme:'system';
 const previous=document.documentElement.dataset.theme;
 if(previous!==next){
  const method={system:'WindowSetSystemDefaultTheme',light:'WindowSetLightTheme',dark:'WindowSetDarkTheme'}[next];
  window.runtime?.[method]?.();
 }
 const dark=next==='dark'||(next==='system'&&matchMedia('(prefers-color-scheme: dark)').matches);
 const [color,on,soft]=palette(preferences.accent,dark?'dark':'light');
 const root=document.documentElement;root.dataset.theme=next;root.dataset.accent=preferences.accent||'graphite';
 for(const [key,value] of Object.entries({'--accent':color,'--button':color,'--on':on,'--accent-soft':soft}))root.style.setProperty(key,value);
}
