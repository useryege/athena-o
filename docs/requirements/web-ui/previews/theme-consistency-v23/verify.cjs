// 静态设计修订验证，不替代正式 React / API / 登录验收。
const fs = require('node:fs');
const path = require('node:path');
const {createRequire} = require('node:module');
const root = path.resolve(__dirname, '../../../../..');
const requireUi = createRequire(path.join(root, 'ui/package.json'));
const {chromium} = requireUi('playwright');
const out = path.join(root, '.superpowers/ui-theme-fixes-20260914');
const base = 'http://127.0.0.1:41823/docs/requirements/web-ui/previews/theme-consistency-v23/';
const antOnly = process.argv.includes('--ant-only');
const columnsOnly = process.argv.includes('--columns-only');
const cases = [
  ['ant','ant-specimen.html'], ['service','theme-service-status-v16.html'], ['wallets','theme-wallets-v20.html'],
  ['radar-hot','theme-market-radar-v21.html?view=hot'], ['radar-realtime','theme-market-radar-v21.html?view=realtime'],
  ['radar-movers','theme-market-radar-v21.html?view=movers'], ['assets','theme-worm-assets-v22.html'],
  ['combinations','theme-worm-combinations-v22.html?view=list'], ['executions','theme-worm-executions-v22.html?view=list'],
  ['profile','theme-common-adaptations-v22.html?view=profile'], ['token','wallet-analytics-v1.html'],
].filter(([id])=>(!antOnly||id==='ant')&&(!columnsOnly||['assets','wallets'].includes(id)));
const sizes = [ ['desktop',1440,1050,16], ['mobile',390,844,16], ['narrow',320,844,16], ['text-200',720,1000,32] ];
const report = {mode:'static-design-revision', scope:antOnly?'ant-targeted-recheck':columnsOnly?'columns-targeted-recheck':'full-revision', base, cases:[], checks:[], failures:[], resources:[], errors:[]};
function check(name, passed, actual) {
  const item = {name, passed, actual}; report.checks.push(item); if (!passed) report.failures.push(item);
}
const rgb = hex => `rgb(${hex.match(/\w\w/g).map(x=>parseInt(x,16)).join(', ')})`;
const style = (page, selector) => page.locator(selector).first().evaluate(e=>{
  const s=getComputedStyle(e), r=e.getBoundingClientRect();
  return {color:s.color, background:s.backgroundColor, border:s.borderTopColor, borderStyle:s.borderTopStyle,
    font:s.fontSize, align:s.textAlign, weight:s.fontWeight, width:r.width,height:r.height,readOnly:e.readOnly,disabled:e.disabled};
});
async function run() {
  fs.mkdirSync(path.join(out,'screenshots'),{recursive:true});
  const browser = await chromium.launch({headless:true});
  report.runtime = {node:process.version, browser:browser.version(), playwright:requireUi('playwright/package.json').version};
  const antFonts = {};
  try {
    for (const [id,url] of cases) for (const [size,width,height,fontSize] of sizes) {
      const page = await browser.newPage({viewport:{width,height}});
      page.on('pageerror',e=>report.errors.push({id,size,message:e.message}));
      page.on('requestfailed',req=>report.resources.push({id,size,url:req.url(),error:req.failure()}));
      page.on('response',response=>{if(response.status()>=400)report.resources.push({id,size,url:response.url(),status:response.status()});});
      await page.goto(base+url); await page.evaluate(()=>document.fonts.ready);
      if (fontSize !== 16) {
        await page.evaluate(value=>{document.documentElement.style.fontSize=`${value}px`;},fontSize);
        // Ant 的 all transition 包含字号；必须在 0.2s 过渡结束后读取最终字体。
        await page.waitForTimeout(500);
      }
      const measured = await page.evaluate(()=>({
        viewport:innerWidth, scroll:Math.max(document.documentElement.scrollWidth,document.body.scrollWidth),
        bodyColor:getComputedStyle(document.body).backgroundColor,
        heading:document.querySelector('h1')?.textContent, font:getComputedStyle(document.body).fontFamily,
      }));
      report.cases.push({id,url,size,...measured});
      check(`${id}/${size}: page fits`,measured.scroll<=width+1,measured);
      check(`${id}/${size}: page color`,measured.bodyColor===rgb('06080B'),measured.bodyColor);
      if (id==='ant') {
        const fonts={};
        for (const s of ['#primary','#editable','.ant-alert-description','.body-copy']) fonts[s]=await style(page,s);
        antFonts[size]=fonts;
        if (size==='desktop') {
          for (const [button,values] of [['primary',['00FFA7','51FFC3','09C385']],['danger',['F58C9B','FFB0BD','E5778B']]]) {
            for (let i=0;i<3;i++) {
              if(i===0)await page.mouse.move(0,0);
              if(i>=1)await page.locator(`#${button}`).hover();
              if(i===2)await page.mouse.down();
              await page.waitForTimeout(240);
              const computed=await style(page,`#${button}`);
              check(`${button}/${['normal','hover','active'][i]}: final paint`,computed.background===rgb(values[i])&&computed.color===rgb('06080B'),computed);
              if(i===2)await page.mouse.up();
            }
          }
          for (const [role,color,bg,border] of [['success','55D9A1','0E201B','2D5747'],['warning','E6BF72','211C13','665231'],['error','F58C9B','24171C','673B47'],['info','9BC6F3','141D27','35516D']]) {
            const box=await style(page,`.ant-alert-${role}`), icon=await style(page,`.ant-alert-${role} .ant-alert-icon`);
            check(`ant/${role}: feedback roles`,box.background===rgb(bg)&&box.border===rgb(border)&&icon.color===rgb(color),{box,icon});
          }
          const selected=await style(page,'.ant-segmented-item-selected'), pager=await style(page,'.ant-pagination-item-active');
          check('ant: segmented selection',selected.background===rgb('102C24')&&selected.color===rgb('00FFA7'),selected);
          check('ant: current page',pager.border===rgb('00FFA7')&&pager.background===rgb('0F1114'),pager);
          const readonly=await style(page,'#readonly'), disabled=await style(page,'#unavailable');
          await page.locator('#readonly').evaluate(e=>{e.focus();e.select();});
          const selection=await page.locator('#readonly').evaluate(e=>({length:e.value.length,selected:e.selectionEnd-e.selectionStart}));
          check('ant: readonly stays selectable',readonly.readOnly&&!readonly.disabled&&selection.selected===selection.length,{readonly,selection});
          check('ant: disabled has distinct state',disabled.disabled&&disabled.borderStyle!==readonly.borderStyle,{readonly,disabled});
          await page.locator('#primary').focus();
          await page.keyboard.press('Tab'); await page.keyboard.press('Shift+Tab');
          const outline=await page.locator('#primary').evaluate(e=>getComputedStyle(e).outlineColor);
          check('ant: keyboard focus visible',outline===rgb('00FFA7'),outline);
        }
      }
      if (id==='service') {
        const overlaps=await page.locator('.services-table tr').evaluateAll(rows=>rows.flatMap(row=>{
          const status=row.querySelector('td:nth-child(2) .tag'), error=row.querySelector('td.service-error');
          if(!status||!error)return [];
          const a=status.getBoundingClientRect(),b=error.getBoundingClientRect();
          return [{status:status.textContent,error:error.textContent,a:{x:a.x,y:a.y,w:a.width,h:a.height},b:{x:b.x,y:b.y,w:b.width,h:b.height},overlap:Math.min(a.right,b.right)>Math.max(a.left,b.left)&&Math.min(a.bottom,b.bottom)>Math.max(a.top,b.top)}];
        }));
        check(`service/${size}: status and error do not overlap`,overlaps.length>0&&overlaps.every(x=>!x.overlap),overlaps);
      }
      if (id==='wallets') {
        const selected=await style(page,'.segmented [aria-pressed=true]');
        check(`wallets/${size}: selected role`,selected.background===rgb('102C24')&&selected.color===rgb('00FFA7'),selected);
        const columns=await page.locator('.wallet-type,.wallet-source').evaluateAll(es=>es.map(e=>({text:e.textContent,align:getComputedStyle(e).textAlign})));
        check(`wallets/${size}: text columns keep their alignment`,columns.length>0&&columns.every(x=>['start','left'].includes(x.align)),columns);
      }
      if (['combinations','executions'].includes(id)) {
        const selected=await style(page,'.pagination [aria-current=page]');
        check(`${id}/${size}: current page`,selected.color===rgb('00FFA7')&&selected.border===rgb('00FFA7')&&selected.background===rgb('0F1114'),selected);
      }
      if (id.startsWith('radar')) {
        for (const [selector,color] of [['.radar-up','55D9A1'],['.radar-connected','55D9A1'],['.radar-down','F58C9B'],['.radar-warmup','E6BF72']]) {
          if (await page.locator(selector).count()) {const computed=await style(page,selector);check(`${id}/${size}: ${selector}`,computed.color===rgb(color),computed);}
        }
      }
      if (size==='desktop'&&['assets','radar-hot','radar-movers'].includes(id)) {
        const selector=id==='assets'?'.wallet-head > :nth-child(2), .wallet-head > :nth-child(3), .wallet-row > :nth-child(2), .wallet-row > :nth-child(3)':'.radar-hot-metrics .radar-metric, .radar-leader .radar-metric:nth-child(2)';
        const aligns=await page.locator(selector).evaluateAll(es=>es.map(e=>({text:e.textContent,align:getComputedStyle(e).textAlign})));
        check(`${id}: comparable columns align right`,aligns.length>0&&aligns.every(x=>x.align==='right'),aligns);
      }
      if (id==='profile') {
        const readonly=await style(page,'#readonly-username'), editable=await style(page,'#display-name');
        check(`profile/${size}: readonly appearance`,readonly.readOnly&&!readonly.disabled&&readonly.background===rgb('181D22')&&readonly.border===rgb('252A30')&&readonly.color===rgb('9FA0A1')&&editable.border!==readonly.border,{readonly,editable});
        if(size==='desktop') {
          const icons=await page.locator('.sidebar .nav-item').evaluateAll(es=>es.map(e=>({label:e.textContent.trim(),icon:e.querySelector('use')?.getAttribute('href')})));
          check('admin: canonical icons',[['Accounts','#grid'],['Service Status','#activity'],['Etherscan Gateways','#api-link']].every(([label,icon])=>icons.some(x=>x.label===label&&x.icon===icon)),icons);
        }
      }
      if (id==='token'&&size==='text-200') {
        const avatars=await page.locator('.avatar').evaluateAll(es=>es.filter(e=>e.getBoundingClientRect().width).map(e=>{
          const r=document.createRange();r.selectNodeContents(e);const a=e.getBoundingClientRect(),b=r.getBoundingClientRect();return {width:a.width,height:a.height,textHeight:b.height,contained:b.top>=a.top-1&&b.bottom<=a.bottom+1};
        }));
        check('token: enlarged avatar contains text',avatars.length>0&&avatars.every(x=>x.contained),avatars);
      }
      await page.mouse.move(0,0);await page.evaluate(()=>{document.querySelectorAll('input[type=text],input:not([type])').forEach(e=>e.setSelectionRange(0,0));document.activeElement?.blur();window.getSelection()?.removeAllRanges();});
      await page.screenshot({path:path.join(out,'screenshots',`${id}-${size}.png`),fullPage:true});
      await page.close();
    }
    for(const selector of Object.keys(antFonts.desktop||{})) {
      const a=antFonts.desktop[selector],b=antFonts['text-200'][selector];
      check(`ant: ${selector} actually doubles`,Math.abs(parseFloat(b.font)/parseFloat(a.font)-2)<.01,{normal:a,enlarged:b});
      if (selector==='#primary'||selector==='#editable') check(`ant: ${selector} keeps scalable target`,a.height>=44&&b.height>=88,{normal:a.height,enlarged:b.height});
    }
    check('no script errors',report.errors.length===0,report.errors);
    check('all resources load',report.resources.length===0,report.resources);
  } finally {await browser.close();}
  fs.writeFileSync(path.join(out,antOnly?'checks-ant-recheck.json':columnsOnly?'checks-columns-recheck.json':'checks.json'),JSON.stringify(report,null,2)+'\n');
  console.log(JSON.stringify({views:report.cases.length,checks:report.checks.length,failures:report.failures},null,2));
  if(report.failures.length)process.exitCode=1;
}
run().catch(e=>{console.error(e);process.exitCode=1;});
