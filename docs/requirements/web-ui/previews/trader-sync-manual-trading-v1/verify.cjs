// Static design-preview verification. Does not start or contact an ATHENA service.
const fs = require('node:fs');
const path = require('node:path');
const {pathToFileURL} = require('node:url');
const root = path.resolve(__dirname, '../../../../..');
const {chromium} = require(path.join(root, 'ui/node_modules/@playwright/test'));
const axe = require.resolve(path.join(root, 'ui/node_modules/axe-core/axe.js'));
const target = pathToFileURL(path.join(__dirname, 'index.html')).href;
const screens = ['wallet','buy','sell','redeem','positions','history','trades','record'];
const report = {mode:'static-design-preview',target,synthetic:true,startedAt:new Date().toISOString(),screenshots:[],layout:[],a11y:[],interactions:[],errors:[],networkRequests:[]};
async function main(){
 const browser=await chromium.launch({headless:true});
 try{
  const context=await browser.newContext();
  const page=await context.newPage();
  page.on('pageerror',e=>report.errors.push(e.message));
  page.on('request',r=>{if(/^https?:/.test(r.url()))report.networkRequests.push(r.url())});
  async function visit(p,s='',width=1440,height=1080){await page.setViewportSize({width,height});await page.goto(target+'?page='+p+(s?'&state='+s:''));await page.evaluate(()=>document.fonts.ready);}
  async function layout(p,variant){const result=await page.evaluate(()=>({viewport:innerWidth,width:document.documentElement.scrollWidth,bodyFont:getComputedStyle(document.body).fontSize,headingFont:getComputedStyle(document.querySelector('h1')).fontSize,overflow:[...document.querySelectorAll('main *')].filter(e=>{const r=e.getBoundingClientRect();return r.width>0&&(r.right>innerWidth+1||r.left< -1)&&!e.closest('.tabs,thead,.sr-only')}).slice(0,8).map(e=>({tag:e.tagName,text:e.textContent.slice(0,100),right:e.getBoundingClientRect().right}))}));report.layout.push({page:p,variant,...result});}
  for(const p of screens){
   for(const [variant,width,height] of [['desktop',1440,1080],['mobile',390,844]]){
    await visit(p,'',width,height);await layout(p,variant);
    const file=`screenshots/${p}-${variant}.png`;await page.screenshot({path:path.join(__dirname,file),fullPage:true});report.screenshots.push(file);
    await page.addScriptTag({path:axe});const result=await page.evaluate(()=>axe.run(document,{runOnly:{type:'tag',values:['wcag2a','wcag2aa','wcag21aa']}}));report.a11y.push({page:p,variant,violations:result.violations});
   }
   console.log('Captured and checked '+p);
  }
  const more=[['wallet','enable-unknown'],['buy','expired'],['sell','first'],['redeem','preparation'],['positions','partial-error'],['history','page-expired'],['record','unknown']];
  for(const [p,s] of more){await visit(p,s,390,844);await layout(p,s+'-mobile');const file=`screenshots/${p}-${s}-mobile.png`;await page.screenshot({path:path.join(__dirname,file),fullPage:true});report.screenshots.push(file);}
  for(const p of screens){await visit(p,'',320,844);await layout(p,'320px');await page.evaluate(()=>document.documentElement.style.fontSize='32px');await layout(p,'320px-text-200');}
  const assert=(condition,name)=>{if(!condition)throw new Error(name);report.interactions.push({name,passed:true})};
  await visit('buy','first');assert(await page.locator('#amount').inputValue()==='','First BUY amount is empty');assert(await page.locator('#slip').inputValue()==='','First price deviation is empty');assert(await page.getByRole('button',{name:'Confirm BUY',exact:true}).isDisabled(),'First BUY cannot submit without input');
  await page.locator('#amount').fill('10.001');await page.locator('#slip').fill('0');await page.getByRole('button',{name:'Get quote',exact:true}).click();assert((await page.locator('#input-error').innerText()).includes('2 decimal'),'Overprecision principal is rejected');
  await page.locator('#amount').fill('10.00');await page.getByRole('button',{name:'Get quote',exact:true}).click();assert(await page.getByRole('button',{name:'Confirm BUY',exact:true}).isEnabled(),'Valid inputs produce a reviewable quote');await page.getByRole('button',{name:'Confirm BUY',exact:true}).click();assert(new URL(page.url()).searchParams.get('page')==='record','Same-page confirmation opens the original result without a modal');
  await visit('sell','first');assert(await page.locator('[data-percent][aria-pressed="true"]').count()===0,'First SELL has no preselected percentage');await page.getByRole('button',{name:'100%',exact:true}).click();await page.locator('#slip').fill('0');await page.getByRole('button',{name:'Get quote',exact:true}).click();assert((await page.locator('main').innerText()).includes('Sell 60 shares · 0.007 available shares remain.'),'100% SELL keeps reserved shares separate and displays dust');
  await page.getByRole('button',{name:'Confirm SELL',exact:true}).click();assert((await page.locator('main').innerText()).includes('60 shares remain reserved'),'Submitted SELL result retains confirmed share quantity');assert(await page.getByRole('button',{name:/retry|resubmit/i}).count()===0,'Unknown result has no duplicate submit action');
  await visit('redeem');assert((await page.locator('main').innerText()).includes('The No shares are also processed, with zero payout.'),'Full redemption displays the zero-payout outcome');assert(await page.locator('#slip').count()===0,'Redemption has no slippage input');
  await visit('redeem','preparation');assert(await page.getByRole('button',{name:'Confirm redemption',exact:true}).isDisabled(),'Required preparation blocks redemption confirmation');
  await visit('wallet');assert(await page.getByRole('button',{name:'Confirm enablement',exact:true}).isDisabled(),'Ongoing approval requires explicit acknowledgment');await page.locator('#enable-consent').check();await page.getByRole('button',{name:'Confirm enablement',exact:true}).click();assert(new URL(page.url()).searchParams.get('state')==='enabling','Enablement has its own progress context');
  await visit('positions');await page.locator('#wallet-filter').selectOption('old');assert((await page.locator('table').innerText()).includes('Previous wallet'),'Unbound wallet positions remain filterable');await page.getByRole('link',{name:'Sell',exact:true}).click();assert((await page.locator('main').innerText()).includes('0x33333333333333333333333333333333333333C3'),'Unbound position retains its original account');
  await visit('buy','expired');assert(await page.getByRole('button',{name:'Confirm BUY',exact:true}).isDisabled(),'Expired quote cannot submit');
  await visit('buy','session-replaced');assert(await page.getByRole('button',{name:'Confirm BUY',exact:true}).count()===0,'Replaced login removes the trade form');
  await visit('buy','',390,844);await page.keyboard.press('Tab');assert(await page.locator('.skip').evaluate(e=>e===document.activeElement),'Keyboard can reach the skip link');
  report.browser=browser.version();report.finishedAt=new Date().toISOString();
 }finally{await browser.close();report.browserClosed=true;fs.writeFileSync(path.join(__dirname,'checks.json'),JSON.stringify(report,null,2)+'\n');}
 const layoutFailures=report.layout.filter(x=>x.width>x.viewport+1||x.overflow.length);
 const a11yFailures=report.a11y.filter(x=>x.violations.length);
 const passed=!layoutFailures.length&&!a11yFailures.length&&!report.errors.length&&!report.networkRequests.length;
 console.log(JSON.stringify({passed,screenshots:report.screenshots.length,layoutChecks:report.layout.length,layoutFailures,a11yRuns:report.a11y.length,a11yFailures:a11yFailures.map(x=>({page:x.page,variant:x.variant,violations:x.violations.map(v=>({id:v.id,impact:v.impact,nodes:v.nodes.length}))})),interactions:report.interactions.length,errors:report.errors,networkRequests:report.networkRequests},null,2));if(!passed)process.exitCode=1;
}
main().catch(e=>{report.errors.push(e.stack);fs.writeFileSync(path.join(__dirname,'checks.json'),JSON.stringify(report,null,2)+'\n');console.error(e);process.exitCode=1});
