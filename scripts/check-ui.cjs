const {chromium,expect}=require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs=require('node:fs');
const path=require('node:path').resolve(__dirname,'..');
(async()=>{
 let browser;
 try {
 browser=await chromium.launch({channel:'chrome',headless:true});
 const page=await browser.newPage({viewport:{width:1440,height:1050},deviceScaleFactor:1,reducedMotion:'reduce'});
 page.setDefaultTimeout(15000); page.setDefaultNavigationTimeout(15000); console.log("browser launched");
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('http://localhost:8080');
 await page.locator('.summary-strip strong').first().waitFor();
 await page.screenshot({path:path+'/docs/qa/clustering-desktop.png',fullPage:true});
 const css=fs.readFileSync(path+'/web/static/theme.css','utf8')+fs.readFileSync(path+'/web/static/style.css','utf8');
 async function snapshot(name){ let html=await page.content();html=html.replace(/<link[^>]*rel="stylesheet"[^>]*>/g,'').replace(/<script\b[^>]*>[\s\S]*?<\/script>/g,'');html=html.replace('</head>','<style>'+css+'</style></head>');fs.writeFileSync(path+'/docs/qa/'+name+'.html',html); }
 await snapshot('clustering');console.log('snapshot ready');
 await page.locator('#tab-plane').click();
 if(!await page.locator('#plot-plane').isVisible()||await page.locator('#plot-cuts').isVisible())throw Error('plot tabs failed');
 await page.locator('#tab-plane').press('ArrowLeft');
 if(!await page.locator('#plot-cuts').isVisible())throw Error('plot keyboard failed');
 await page.locator('#tab-plane').click();
 await page.locator('#alpha-number').fill('0.7');
 await page.waitForFunction(()=>document.querySelector('.summary-strip > div:nth-child(3) strong')?.textContent==='0.70');
 if(await page.locator('#alpha-range').inputValue()!=='0.7')throw Error('alpha slider not synchronized');
 if(!await page.locator('#plot-plane').isVisible())throw Error('plot selection lost after update');
 await page.locator('#tab-cuts').click();console.log('alpha passed');await page.locator('#algorithm').selectOption('agglo');
 await page.waitForFunction(()=>document.querySelector('#k-field').hidden);
 if(!await page.locator('#threshold-field').isVisible())throw Error('threshold hidden');
 await page.waitForFunction(()=>document.querySelector('#view').getAttribute('aria-busy')!=='true');
 console.log('algorithm passed');await page.locator('#generation-mode').selectOption('random');
 await page.locator('[name=n]').fill('120');
 await page.locator('#gen button').click();
 await page.waitForFunction(()=>document.querySelector('.summary-strip strong')?.textContent==='120');
 console.log('before theme');await page.locator('#theme-toggle').click();
 if(await page.locator('html').getAttribute('data-theme')!=='dark')throw Error('theme failed');
 await page.screenshot({path:path+'/docs/qa/clustering-dark.png',fullPage:true});
 await page.goto('http://localhost:8080/ops');
 await page.locator('.poly').nth(2).waitFor();
 console.log('before theme');await page.locator('#theme-toggle').click();
 await page.screenshot({path:path+'/docs/qa/arithmetic-desktop.png',fullPage:true});
 await snapshot('arithmetic');
 for(const op of ['sub','mul','div']){
  await page.locator('[name=op]').selectOption(op);
  await page.waitForFunction(op=>document.querySelector('#opsview tbody tr:last-child th')?.textContent.includes({sub:'−',mul:'×',div:'÷'}[op]),op);
  if(await page.locator('.error').count())throw Error('operation '+op+' failed');
 }
 await page.locator('[name=sigma2]').fill('0');
 if(await page.locator('[name=sigma2]').evaluate(el=>el.validity.valid))throw Error('invalid sigma accepted by form');
 const invalid=await page.request.get('http://localhost:8080/ops/calc?mu1=0&sigma1=0.3&mu2=1&sigma2=0&op=add');
 if(!(await invalid.text()).includes('class="error"'))throw Error('server accepted invalid sigma');
 await page.locator('[name=sigma2]').fill('0.4');
 await page.waitForFunction(()=>!document.querySelector('#opsview .error'));
 const overflows=[];
 for(const route of ['/','/ops']){
  for(const width of [320,390,768,1024,1440]){
   await page.setViewportSize({width,height:900}); await page.goto('http://localhost:8080'+route);
   const size=await page.evaluate(()=>({body:document.documentElement.scrollWidth,view:innerWidth}));
   if(size.body>size.view+1)overflows.push({route,width,...size});
   if(width===390)await page.screenshot({path:path+'/docs/qa/'+(route==='/'?'clustering':'arithmetic')+'-mobile.png',fullPage:true});
  }
 }
 console.log(JSON.stringify({pageErrors:errors,horizontalOverflow:overflows,checks:['alpha sync and htmx refresh','algorithm fields','generation 120 numbers','persistent theme','all four operations','sigma validation','10 responsive viewports']},null,2));
 fs.writeFileSync(path+'/docs/qa/browser-results.json',JSON.stringify({pageErrors:errors,horizontalOverflow:overflows},null,2));
 if(errors.length||overflows.length)throw Error("UI errors found");
 } finally { await browser?.close(); }
})().catch(e=>{console.error(e);process.exit(1)});
