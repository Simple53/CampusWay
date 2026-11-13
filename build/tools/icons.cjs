const fs = require('fs');
const {Resvg} = require('./node_modules/@resvg/resvg-js');
const source = fs.readFileSync('design/campusway/icons/route.svg','utf8').replace('<svg ', '<svg style="color:#111" ').replace('currentColor','#111');
fs.writeFileSync('build/assets/campusway.png', new Resvg(source,{fitTo:{mode:'width',value:512},background:'#ffffff'}).render().asPng());
