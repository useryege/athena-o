// 从冻结原件生成修订稿及真实 Ant SSR 样式样本；不触碰 ui/。
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const {createRequire} = require('node:module');
const root = path.resolve(__dirname, '../../../../..');
const requireUi = createRequire(path.join(root, 'ui/package.json'));
const React = requireUi('react');
const {renderToString} = requireUi('react-dom/server');
const {App, ConfigProvider, Button, Input, Alert, Segmented, Pagination, theme} = requireUi('antd');
const {StyleProvider, createCache, extractStyle, px2remTransformer} = requireUi('@ant-design/cssinjs');
const {createAthenaTheme} = require('./theme-reference.cjs');
const base = path.dirname(__dirname);
const sources = [
  'theme-service-status-v16.html', 'theme-wallets-v20.html', 'theme-market-radar-v21.html',
  'theme-worm-assets-v22.html', 'theme-worm-combinations-v22.html', 'theme-worm-executions-v22.html',
  'theme-common-adaptations-v22.html', 'wallet-analytics-v1.html',
];
const digest = text => crypto.createHash('sha256').update(text).digest('hex');
const manifest = {kind: 'design-review-correction', version: 23, productionImplemented: false, sources: []};
const admin = fs.readFileSync(path.join(base, 'theme-admin-accounts-v15.html'), 'utf8');
for (const file of sources) {
  const source = file === 'wallet-analytics-v1.html'
    ? path.join(root, 'docs/requirements/token/previews', file) : path.join(base, file);
  const original = fs.readFileSync(source, 'utf8');
  let html = original.replace(/url\(['"]?(?:[^)'"\s]*\/)?files\//g, "url('../files/");
  // 页面链接优先留在本修订包，其余仍到未改动的原件。
  html = html.replace(/(?<![\w/-])(theme-[\w-]+\.html)/g, match => sources.includes(match) ? match : `../${match}`);
  html = html.replace(/<title>(.*?)<\/title>/s, '<title>$1 · v23 consistency revision</title>');
  // Token 原件只允许内联样式；嵌入同一权威文件的构建结果，不放宽 CSP。
  const revisionStyles = file === 'wallet-analytics-v1.html'
    ? `<style>${fs.readFileSync(path.join(__dirname,'tokens.css'),'utf8')}\n${fs.readFileSync(path.join(__dirname,'corrections.css'),'utf8')}</style>`
    : '<link rel="stylesheet" href="tokens.css"><link rel="stylesheet" href="corrections.css">';
  html = html.replace('</head>', revisionStyles+'</head>');
  if (file === 'theme-market-radar-v21.html') {
    html = html.replace("state==='stale'?'radar-warmup':'radar-up'", "state==='stale'?'radar-warmup':'radar-connected'");
  }
  if (file === 'theme-worm-assets-v22.html') {
    for (const label of ['SOL balance', 'USDC balance']) {
      html = html.replace(`<span>${label}</span>`, `<span class="athena-numeric-column">${label}</span>`);
      html = html.replace(`<div><span class="mobile-label">${label}</span>`, `<div class="athena-numeric-column"><span class="mobile-label">${label}</span>`);
    }
  }
  if (file === 'theme-common-adaptations-v22.html') {
    for (const [label, before, after] of [['Accounts','v11-user','grid'], ['Service Status','v11-shield','activity'], ['Etherscan Gateways','layers','api-link']]) {
      const pattern = new RegExp(`(\\['${label}','[^']+','?)${before}('\\])`);
      html = html.replace(pattern, `$1${after}$2`);
    }
    for (const glyph of ['activity', 'api-link']) {
      const symbol = admin.match(new RegExp(`<symbol id="i-${glyph}"[^>]*>[\\s\\S]*?<\\/symbol>`));
      if (!symbol) throw new Error(`Missing canonical admin glyph ${glyph}`);
      html = html.replace('</defs>', symbol[0].replace(`id="i-${glyph}"`, `id="${glyph}"`) + '</defs>');
    }
  }
  fs.writeFileSync(path.join(__dirname, file), html);
  manifest.sources.push({file, source: path.relative(root, source), sourceSha256: digest(original), revisionSha256: digest(html)});
}

const css = fs.readFileSync(path.join(__dirname, 'tokens.css'), 'utf8');
const read = name => {
  const value = css.match(new RegExp(`--athena-${name}:\\s*([^;]+);`))?.[1].trim();
  if (!value || value.startsWith('var(')) throw new Error(`Missing literal token ${name}`);
  return value;
};
const config = createAthenaTheme(theme, read);
const h = React.createElement;
const cache = createCache();
const field = (id, label, props, hint) => h('div', {className:'spec-field'},
  h('label', {htmlFor:id}, label), h(Input, {id, 'aria-describedby':`${id}-hint`, ...props}), h('p', {id:`${id}-hint`}, hint));
const specimen = h(App, {}, h('main', {className:'specimen'},
  h('header', {}, h('h1', {}, 'Shared interface states'), h('p', {className:'body-copy'}, 'Consistent color, readable data and controls that grow with your text.')),
  h('section', {className:'spec-panel', 'aria-labelledby':'actions-title'}, h('h2', {id:'actions-title'}, 'Actions'),
    h('div', {className:'spec-actions'},
      h(Button,{id:'primary', type:'primary'},'Query wallet'), h(Button,{id:'secondary'},'Refresh data'),
      h(Button,{id:'danger', type:'primary', danger:true},'Discard draft'), h(Button,{id:'disabled', disabled:true},'Unavailable'))),
  h('section', {className:'spec-panel', 'aria-labelledby':'profile-title'}, h('h2', {id:'profile-title'}, 'Profile fields'),
    h('div', {className:'spec-fields'},
      field('editable', 'Display name', {defaultValue:'Alex Chen'}, 'Editable'),
      field('readonly', 'Username', {defaultValue:'@alexchen', readOnly:true}, 'Read-only · select and copy'),
      field('unavailable', 'Display name unavailable', {defaultValue:'Alex Chen', disabled:true}, 'Disabled · temporarily unavailable'))),
  h('section', {className:'spec-panel', 'aria-labelledby':'selection-title'}, h('h2', {id:'selection-title'}, 'Current selection'),
    h('div', {className:'spec-selection'}, h('div', {}, h('p', {className:'field-label'}, 'Wallet type'),
      h(Segmented, {options:['All','EVM','Solana'], value:'EVM', 'aria-label':'Wallet type specimen'})),
      h('div', {}, h('p', {className:'field-label'}, 'Current page'), h(Pagination, {defaultCurrent:2, total:50, pageSize:20, showSizeChanger:false})))),
  h('section', {className:'spec-panel', 'aria-labelledby':'feedback-title'}, h('h2', {id:'feedback-title'}, 'Feedback'),
    h('div', {className:'spec-feedback'},
      h(Alert, {type:'success', title:'Connection ready', description:'The latest account snapshot is available.', showIcon:true}),
      h(Alert, {type:'warning', title:'Sampling in progress', description:'This window does not yet have enough observations.', showIcon:true}),
      h(Alert, {type:'error', title:'Service unreachable', description:'Health check timed out after 5 seconds.', showIcon:true}),
      h(Alert, {type:'info', title:'Previous result available', description:'You can keep reading the saved result while refreshing.', showIcon:true}))),
  h('footer', {}, 'Design revision v23 · synthetic examples · Ant Design server-rendered specimen; pagination and filters demonstrate appearance only.')
));
const body = renderToString(h(StyleProvider, {cache, transformers:[px2remTransformer({rootValue:16, mediaQuery:false})]},
  h(ConfigProvider, {theme:config}, specimen)));
const layout = `
*{box-sizing:border-box}body{margin:0;background:var(--athena-bg);color:var(--athena-text);font-family:var(--athena-font)}
.specimen{max-width:80rem;margin:auto;padding:2rem}.specimen h1{font-size:1.75rem;font-weight:600;line-height:1.3;margin:0 0 .75rem}
.body-copy{font-size:1rem;line-height:1.6;color:var(--athena-muted);margin:0 0 2rem;max-width:70ch}
.spec-panel{padding:1.5rem;background:var(--athena-panel);border:1px solid var(--athena-border);border-radius:.875rem;margin-block:1.5rem;min-width:0}
.spec-panel h2{font-size:1.25rem;font-weight:600;line-height:1.4;margin:0 0 1.25rem}.spec-actions{display:flex;flex-wrap:wrap;gap:.75rem}
.spec-fields{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,16rem),1fr));gap:1.5rem}.spec-field{min-width:0}
.spec-field label,.field-label{display:block;font-size:.875rem;font-weight:500;margin:0 0 .5rem}.spec-field p{font-size:.8125rem;line-height:1.55;color:var(--athena-muted);margin:.5rem 0 0}
.spec-selection{display:flex;flex-wrap:wrap;gap:1.5rem 3rem}.spec-selection>div{min-width:0;max-width:100%}
.spec-feedback{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,24rem),1fr));gap:1rem}
footer{font-size:.8125rem;line-height:1.55;color:var(--athena-muted);max-width:75ch;margin:2rem 0}
@media(max-width:600px){.specimen{padding:1.25rem}.spec-panel{padding:1.25rem}.specimen h1{font-size:1.5rem}.spec-actions{flex-direction:column;align-items:stretch}}
`;
fs.writeFileSync(path.join(__dirname, 'ant-specimen.html'), `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>ATHENA · v23 shared states</title><link rel="stylesheet" href="tokens.css">${extractStyle(cache)}<link rel="stylesheet" href="corrections.css"><style>${layout}</style></head><body>${body}</body></html>`);
manifest.runtime = {node:process.version, react:requireUi('react/package.json').version, antd:requireUi('antd/package.json').version, cssinjs:requireUi('@ant-design/cssinjs/package.json').version};
fs.writeFileSync(path.join(__dirname, 'manifest.json'), JSON.stringify(manifest, null, 2)+'\n');
console.log(JSON.stringify({generated:sources.length+1, runtime:manifest.runtime}));
