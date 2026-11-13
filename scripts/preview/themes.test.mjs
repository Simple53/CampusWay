import assert from 'node:assert/strict';
import fs from 'node:fs';
import {accents,palette} from '../../gui/static/core/themes.js';
import {createPreviewBridge} from './mock.mjs';
function luminance(hex){return hex.slice(1).match(/../g).map(c=>parseInt(c,16)/255).map(c=>c<=.04045?c/12.92:((c+.055)/1.055)**2.4).reduce((s,c,i)=>s+c*[.2126,.7152,.0722][i],0);}
for(const key of Object.keys(accents))for(const mode of ['light','dark']){const [bg,fg]=palette(key,mode),a=luminance(bg),b=luminance(fg);assert.ok((Math.max(a,b)+.05)/(Math.min(a,b)+.05)>=4.5,`${key}/${mode} button contrast`);}
const {api}=createPreviewBridge();for(const accent of Object.keys(accents)){const prefs=(await api.GetSnapshot()).preferences;await api.SavePreferences({...prefs,accent});assert.equal((await api.GetSnapshot()).preferences.accent,accent);}
await assert.rejects(api.SavePreferences({...((await api.GetSnapshot()).preferences),accent:'invalid'}));
assert.ok(fs.readFileSync('gui/static/icons.svg','utf8').includes('id="pencil"'));
assert.ok(fs.readFileSync('gui/static/pages/connections.js','utf8').includes("icon('pencil')"));
console.log('Six color themes, light/dark contrast, preference validation and pencil sprite passed.');
