const http = require('http');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { URL } = require('url');
const QRCodeMatrix = require('./qrcode');
const https = require('https');

const ROOT = path.resolve(__dirname, '..');
const WEB = path.join(ROOT, 'web');
const DATA_DIR = process.env.DATA_DIR ? path.resolve(process.env.DATA_DIR) : path.join(ROOT, 'data');
const UPLOAD_DIR = path.join(ROOT, 'uploads');
fs.mkdirSync(DATA_DIR,{recursive:true}); fs.mkdirSync(UPLOAD_DIR,{recursive:true});
const DB = path.join(DATA_DIR,'db.json');

const DEFAULT_SETTINGS = {
  app:{name:'QR Se Print',apiVersion:'complete-2026.09.05'},
  server:{port:8080,maxUploadBytes:25*1024*1024,requestBodyLimitBytes:2*1024*1024},
  health:{agentOnlineSeconds:45},
  uploads:{cleanupMinutes:10,fileMaxAgeMinutes:90},
  demo:{enabled:true,instant:true,hours:24,durationDays:1,printLimit:10,priceBW:5,priceColor:10},
  defaults:{shop:{shopPaymentMode:'both',onlineAllowed:false,priceBW:3,priceColor:5,
    duplexBWEnabled:true,duplexColorEnabled:true,duplexMode:'same',priceBWDuplex:6,priceColorDuplex:10,
    price4x6_4:10,price4x6_6:15,price4x6_8:20,price4x6_10:25,priceResumeColor:20,priceResumeBW:10,
    priceA3BW:10,priceA3Color:20,priceA2BW:20,priceA2Color:40,priceA1BW:40,priceA1Color:80,
    printerModel:'',printerNameBW:'',printerNameColor:'',printerName4x6:'',printerNameA3:'',printerNameDuplex:''}},
  plans:{demo:{name:'Demo',price:0,durationDays:1,printLimit:10,advance:false},
    starter:{name:'Starter',price:599,durationDays:0,printLimit:null,advance:false},
    pro:{name:'Pro',price:899,durationDays:0,printLimit:null,advance:true},
    premium:{name:'Premium',price:999,durationDays:0,printLimit:null,advance:true}},
  agent:{defaultServerURL:'https://bvv-djql.onrender.com',defaultShopID:'DEMO',pollSeconds:5,version:'9.1.0',jobStatusPollSeconds:5},
  printers:{models:["🔍 Auto Detect (System Installed Printer)","Epson L120","Epson L130","Epson L210","Epson L220","Epson L360","Epson L361","Epson L380","Epson L385","Epson L395","Epson L1110","Epson L1210","Epson L1250","Epson L1255","Epson L1300","Epson L1350","Epson L1455","Epson L3100","Epson L3101","Epson L3110","Epson L3115","Epson L3116","Epson L3150","Epson L3151","Epson L3152","Epson L3156","Epson L3200","Epson L3210","Epson L3211","Epson L3215","Epson L3216","Epson L3250","Epson L3251","Epson L3252","Epson L3255","Epson L3256","Epson L3260","Epson L3550","Epson L3560","Epson L4150","Epson L4160","Epson L4260","Epson L5190","Epson L5290","Epson L5390","Epson L5590","Epson L6160","Epson L6170","Epson L6190","Epson L6270","Epson L6290","Epson L6460","Epson L6490","Epson L6570","Epson L6580","Epson L8050","Epson L8160","Epson L8180","Epson L11050","Epson L14150","Epson L15150","Epson L15160","Epson L15180","Epson L18050","Epson M1100","Epson M1120","Epson M1140","Epson M1170","Epson M2120","Epson M2140","Epson M2170","Epson WF-2810","Epson WF-2830","Epson WF-3825","Epson WF-C5390","Canon PIXMA G1010","Canon PIXMA G1020","Canon PIXMA G1030","Canon PIXMA G2002","Canon PIXMA G2010","Canon PIXMA G2012","Canon PIXMA G2020","Canon PIXMA G2070","Canon PIXMA G3000","Canon PIXMA G3010","Canon PIXMA G3012","Canon PIXMA G3020","Canon PIXMA G3060","Canon PIXMA G3070","Canon PIXMA G3770","Canon PIXMA G4010","Canon PIXMA G4020","Canon PIXMA G4070","Canon PIXMA G5070","Canon PIXMA G6070","Canon PIXMA G7070","Canon PIXMA TS207","Canon PIXMA TS307","Canon PIXMA TS3340","Canon PIXMA TS3475","Canon PIXMA E477","Canon PIXMA E3370","Canon PIXMA E4270","Canon PIXMA MG2470","Canon PIXMA MG3070","Canon LBP2900","Canon LBP3300","Canon LBP6030","Canon LBP6230DW","Canon LBP226dw","Canon imageCLASS MF3010","Canon imageCLASS MF237w","Canon imageCLASS MF244dw","HP DeskJet 1112","HP DeskJet 2131","HP DeskJet 2332","HP DeskJet 2710","HP DeskJet 2720","HP DeskJet 2776","HP DeskJet 2778","HP DeskJet 3635","HP DeskJet 3776","HP DeskJet 3835","HP DeskJet 4178","HP DeskJet Ink Advantage 2135","HP Smart Tank 515","HP Smart Tank 520","HP Smart Tank 580","HP Smart Tank 615","HP Smart Tank 670","HP Smart Tank 750","HP Ink Tank 315","HP Ink Tank 319","HP Ink Tank 415","HP Ink Tank 419","HP Ink Tank Wireless 416","HP LaserJet 1018","HP LaserJet 1020","HP LaserJet 1022","HP LaserJet M1005","HP LaserJet M1136","HP LaserJet P1108","HP LaserJet P1505","HP LaserJet Pro M15a","HP LaserJet Pro M15w","HP LaserJet Pro M126nw","HP LaserJet Pro M404dn","HP LaserJet Pro MFP M126nw","HP LaserJet Pro MFP M225dw","Brother DCP-T220","Brother DCP-T225","Brother DCP-T226","Brother DCP-T310","Brother DCP-T420W","Brother DCP-T426W","Brother DCP-T520W","Brother DCP-T710W","Brother DCP-T820DW","Brother HL-1201","Brother HL-1221fn","Brother HL-L2321D","Brother HL-L2361DN","Brother HL-L2375DW","Brother MFC-J2330DW","Brother MFC-T920DW","Brother MFC-T4500DW","Kyocera Ecosys P2040dn","Kyocera Ecosys P2235dn","Kyocera Ecosys M2040dn","Kyocera Ecosys M2540dn","Kyocera FS-1020D","Ricoh SP 210","Ricoh SP 311DN","Ricoh MP 2014","Samsung ML-1640","Samsung Xpress M2020","Other (Manually Type Below)"]},
  features:{advance:['4x6 Photo','Resume','A3','Duplex','Mini Print']},
  payments:{setupFee:0,currency:'INR',registrationGateway:'razorpay',shopGatewayFallback:'razorpay',registrationKeyIdEnv:'PLATFORM_RAZORPAY_KEY_ID',registrationKeySecretEnv:'PLATFORM_RAZORPAY_KEY_SECRET',shopKeysStoredPerShop:true},
  security:{demoPasswordMinLength:4,shopPasswordMinLength:4}
};

function mergeMissing(dst, src){
  for(const [k,v] of Object.entries(src)){
    if(!(k in dst)) dst[k]=v;
    else if(v && typeof v==='object' && !Array.isArray(v) && dst[k] && typeof dst[k]==='object' && !Array.isArray(dst[k])) mergeMissing(dst[k],v);
  }
  return dst;
}
function load(){
  try{
    const d=JSON.parse(fs.readFileSync(DB,'utf8'));
    d.shops=d.shops||[]; d.jobs=d.jobs||[]; d.tokens=d.tokens||{}; d.settings=d.settings||{};
    mergeMissing(d.settings,DEFAULT_SETTINGS);
    d.events=d.events||[]; d.adminTokens=d.adminTokens||{}; d.pendingRegistrations=d.pendingRegistrations||{};
    return d;
  }catch(e){return {shops:[],jobs:[],tokens:{},settings:JSON.parse(JSON.stringify(DEFAULT_SETTINGS)),events:[],adminTokens:{},pendingRegistrations:{}};}
}
let db=load();
const CFG=()=>db.settings;
const PLANS=()=>CFG().plans;
const SHOP_DEFAULTS=()=>CFG().defaults.shop;
const PORT = Number(process.env.PORT || CFG().server.port);
function save(){fs.writeFileSync(DB+'.tmp',JSON.stringify(db,null,2));fs.renameSync(DB+'.tmp',DB);}
function id(prefix){return prefix+'_'+crypto.randomBytes(8).toString('hex');}
function sha(s){return crypto.createHash('sha256').update(s).digest('hex');}
function json(res,code,obj){const b=Buffer.from(JSON.stringify(obj));res.writeHead(code,{'Content-Type':'application/json; charset=utf-8','Content-Length':b.length,'Cache-Control':'no-store','Access-Control-Allow-Origin':'*','Access-Control-Allow-Headers':'Content-Type, Authorization','Access-Control-Allow-Methods':'GET,POST,PUT,OPTIONS'});res.end(b);}
function text(res,code,s,type='text/plain'){const b=Buffer.from(s);res.writeHead(code,{'Content-Type':type,'Content-Length':b.length});res.end(b);}
function readBody(req,limit=Number(CFG().server.requestBodyLimitBytes||2*1024*1024)){return new Promise((resolve,reject)=>{let a=[],n=0;req.on('data',c=>{n+=c.length;if(n>limit){reject(new Error('BODY_TOO_LARGE'));req.destroy();return;}a.push(c);});req.on('end',()=>resolve(Buffer.concat(a)));req.on('error',reject);});}
async function bodyJson(req){const b=await readBody(req);try{return JSON.parse(b.toString('utf8')||'{}')}catch(e){throw new Error('INVALID_JSON')}}
function bearer(req){const h=req.headers.authorization||'';return h.startsWith('Bearer ')?h.slice(7):'';}
function auth(req){const t=bearer(req);const shopId=db.tokens[t];return shopId?db.shops.find(s=>s.id===shopId):null;}
function adminAuth(req){const t=bearer(req); const a=db.adminTokens&&db.adminTokens[t]; return a==='admin'?a:null;}
function normalizeShop(s){ if(!s.plan)s.plan='demo'; if(s.plan==='demo'){s.advanced_unlocked=true;if(s.advanced_active===undefined)s.advanced_active=true;} if(s.plan==='demo'&&!s.demo_expires_at&&s.created_at)s.demo_expires_at=new Date(Date.parse(s.created_at)+Number(CFG().demo.durationDays||1)*86400000).toISOString(); if(typeof s.demo_prints_used!=='number')s.demo_prints_used=0; if(!s.plan_status)s.plan_status='active'; return s;}

function seed(){
  if(db.shops.length)return;
  const d=SHOP_DEFAULTS();
  db.shops=[{id:'DEMO',name:'QR Se Print Demo Shop',address:'',phone:'',email:'',passwordHash:sha('1234'),
    paused:false,supply_warning:false,subscription_expired:false,shop_payment_mode:d.shopPaymentMode||'both',
    online_allowed:!!d.onlineAllowed,color_price:Number(d.priceColor),bw_price:Number(d.priceBW),agent_token:'agent_demo',
    agent_id:null,agent_last_heartbeat:null,agent_printers:[],plan:'demo',plan_status:'active',
    demo_expires_at:new Date(Date.now()+Number(CFG().demo.durationDays||1)*86400000).toISOString(),
    demo_print_limit:Number(CFG().demo.printLimit||10),demo_prints_used:0,created_at:new Date().toISOString()}];
  db.adminTokens=db.adminTokens||{}; save();
}
db.adminTokens=db.adminTokens||{}; db.shops.forEach(normalizeShop); seed(); save();

function parseMultipart(buf,ctype){
 const m=/boundary=(?:"([^"]+)"|([^;]+))/i.exec(ctype||''); if(!m) throw new Error('MULTIPART_BOUNDARY_MISSING');
 const boundary=Buffer.from('--'+(m[1]||m[2])); const out={fields:{},file:null}; let p=0;
 while((p=buf.indexOf(boundary,p))!==-1){p+=boundary.length;if(buf[p]===45&&buf[p+1]===45)break;if(buf[p]===13&&buf[p+1]===10)p+=2;const end=buf.indexOf(Buffer.from('\r\n\r\n'),p);if(end<0)break;const hs=buf.slice(p,end).toString();let next=buf.indexOf(boundary,end+4);if(next<0)break;let dataEnd=next-2;if(dataEnd<end+4)dataEnd=end+4;const data=buf.slice(end+4,dataEnd);const nm=/name="([^"]+)"/i.exec(hs);if(!nm)continue;const fn=/filename="([^"]*)"/i.exec(hs);if(fn)out.file={name:fn[1],data};else out.fields[nm[1]]=data.toString();p=next;}
 return out;
}
function routePage(res,file){const p=path.join(WEB,file);if(!fs.existsSync(p))return text(res,404,'Not found');res.writeHead(200,{'Content-Type':'text/html; charset=utf-8','Cache-Control':'no-store, no-cache, must-revalidate, proxy-revalidate','Pragma':'no-cache','Expires':'0'});fs.createReadStream(p).pipe(res);}
function shopQrUrl(shopId){ return `/api/shop/${encodeURIComponent(shopId)}/qr`; }
function publicShop(id0){const s=db.shops.find(x=>x.id.toUpperCase()===String(id0).toUpperCase());if(!s)return null;const out={...s,passwordHash:undefined,agent_token:undefined};out.price_bw=Number(s.price_bw??s.bw_price??SHOP_DEFAULTS().priceBW);out.price_color=Number(s.price_color??s.color_price??SHOP_DEFAULTS().priceColor);out.bw_price=Number(s.bw_price??out.price_bw);out.color_price=Number(s.color_price??out.price_color);out.payment_mode=s.payment_mode??s.shop_payment_mode??'both';out.has_razorpay_secret=!!(s.razorpay_key_secret);out.has_cashfree_secret=!!(s.cashfree_secret_key||process.env.CASHFREE_SECRET_KEY);out.payment_gateway=s.payment_gateway||((s.razorpay_key_id)?'razorpay':'');out.online_allowed=(out.payment_mode==='both'||out.payment_mode==='online_only') && !!(s.razorpay_key_id) && !!(s.razorpay_key_secret);out.demo=String(s.plan||'').toLowerCase()==='demo';out.advanced_unlocked=out.demo ? true : !!s.advanced_unlocked;out.advanced_active=out.demo ? true : s.advanced_active!==false;out.qr_code=shopQrUrl(s.id);return out;}
function qrSvg(text){const qr=new QRCodeMatrix(-1, 2); qr.addData(text); qr.make(); const n=qr.getModuleCount(), pad=4, size=n+pad*2; let rects=''; for(let r=0;r<n;r++){for(let c=0;c<n;c++){if(qr.isDark(r,c)) rects += `<rect x=\"${c+pad}\" y=\"${r+pad}\" width=\"1\" height=\"1\"/>`;}} return `<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 ${size} ${size}\" shape-rendering=\"crispEdges\"><rect width=\"100%\" height=\"100%\" fill=\"white\"/><g fill=\"black\">${rects}</g></svg>`;}

function calcAmount(x){const pages=Math.max(1,Number(x.totalPages||1));const copies=Math.max(1,Number(x.copies||1));const price=(String(x.colorMode||'bw').toLowerCase().includes('color'))?Number(db.shops.find(s=>s.id===x.shopId)?.color_price??SHOP_DEFAULTS().priceColor):Number(db.shops.find(s=>s.id===x.shopId)?.bw_price??SHOP_DEFAULTS().priceBW);return pages*copies*price;}

async function api(req,res,u){
 const p=u.pathname;
 // Render/monitoring health endpoints: never redirect, always return JSON.
 if((p==='/api/health'||p==='/api/health/') && req.method==='GET'){
   return json(res,200,{success:true,app:'QR Se Print',api_version:CFG().app.apiVersion,status:'ok',storage:'local-json',db:false,time:new Date().toISOString()});
 }
 if(p==='/api/index.php' && req.method==='GET' && (u.searchParams.get('route')||'')==='health'){
   return json(res,200,{success:true,app:'QR Se Print',api_version:CFG().app.apiVersion,db:false,time:new Date().toISOString()});
 }
 if(p==='/api/plans'&&req.method==='GET'){return json(res,200,{success:true,plans:PLANS()});}
 if(p==='/api/admin/login'&&req.method==='POST'){db.adminTokens=db.adminTokens||{};const x=await bodyJson(req);const user=String(x.username||'');const pass=String(x.password||'');const envUser=String(process.env.ADMIN_USER||'').trim();const envPass=String(process.env.ADMIN_PASSWORD||'');const validDefault=(user==='admin'&&pass==='Admin@12345');const validEnv=(envUser&&envPass&&user===envUser&&pass===envPass);if(!validDefault&&!validEnv)return json(res,401,{success:false,error:'Invalid admin credentials'});const t=id('adm');db.adminTokens[t]='admin';save();return json(res,200,{success:true,token:t,user:'admin'});}
 if(p==='/api/admin/me'&&req.method==='GET'){if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'});return json(res,200,{success:true,user:'admin'});}
 if(p==='/api/admin/overview'&&req.method==='GET'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'});
   const now=Date.now();
   const istDay=d=>new Intl.DateTimeFormat('en-CA',{timeZone:'Asia/Kolkata',year:'numeric',month:'2-digit',day:'2-digit'}).format(d);
   const today=istDay(new Date());
   const paid=j=>['paid','counter'].includes(String(j.payment_status||'').toLowerCase());
   const todayJobs=db.jobs.filter(j=>istDay(new Date(j.created_at||0))===today&&paid(j));
   const activeShops=db.shops.filter(s=>!s.paused&&s.plan_status==='active').length;
   const onlineAgents=db.shops.filter(s=>s.agent_last_heartbeat&&now-Date.parse(s.agent_last_heartbeat)<Number(CFG().health.agentOnlineSeconds||45)*1000).length;
   return json(res,200,{success:true,stats:{shops:db.shops.length,activeShops,pausedShops:db.shops.filter(s=>s.paused).length,expiredShops:db.shops.filter(s=>s.subscription_expired||s.plan_status==='expired').length,agentsOnline:onlineAgents,todayOrders:todayJobs.length,todayPrints:todayJobs.reduce((a,j)=>a+Math.max(1,Number(j.totalPages||1))*Math.max(1,Number(j.copies||1)),0),todayEarnings:todayJobs.reduce((a,j)=>a+Number(j.amount||0),0),totalOrders:db.jobs.length,totalEarnings:db.jobs.filter(paid).reduce((a,j)=>a+Number(j.amount||0),0)}});
 }
 if(p==='/api/admin/shops'&&req.method==='GET'){if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'});return json(res,200,{success:true,shops:db.shops.map(s=>({...publicShop(s.id),plan:s.plan,plan_status:s.plan_status,demo_expires_at:s.demo_expires_at,demo_print_limit:s.demo_print_limit,demo_prints_used:s.demo_prints_used,agent_online:!!s.agent_last_heartbeat&&Date.now()-Date.parse(s.agent_last_heartbeat)<Number(CFG().health.agentOnlineSeconds||45)*1000}))});}
 if(p==='/api/admin/impersonate'&&req.method==='POST'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'});
   const x=await bodyJson(req); const sid=String(x.shopId||'').toUpperCase(); const s=db.shops.find(q=>String(q.id).toUpperCase()===sid);
   if(!s)return json(res,404,{success:false,error:'Shop not found'});
   const t=id('tok'); db.tokens[t]=s.id; save();
   return json(res,200,{success:true,token:t,shop:publicShop(s.id),impersonated:true});
 }
 if(p==='/api/admin/shop/get'&&req.method==='GET'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'});
   const sid=String(u.searchParams.get('shopId')||'').toUpperCase(); const s=db.shops.find(q=>String(q.id).toUpperCase()===sid);
   if(!s)return json(res,404,{success:false,error:'Shop not found'});
   const jobs=db.jobs.filter(j=>j.shopId===s.id);
   return json(res,200,{success:true,shop:{...publicShop(s.id),passwordHash:undefined,razorpay_key_secret:undefined},jobs:jobs.slice(-100).reverse()});
 }
 if(p==='/api/admin/shop/create'&&req.method==='POST'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'}); const x=await bodyJson(req);
   let sid=String(x.id||'').trim().toUpperCase().replace(/[^A-Z0-9_-]/g,''); if(!sid)sid='SHOP'+Math.random().toString(36).slice(2,8).toUpperCase();
   if(db.shops.some(s=>String(s.id).toUpperCase()===sid))return json(res,409,{success:false,error:'Shop ID already exists'});
   const pw=String(x.password||Math.floor(100000+Math.random()*900000)); if(pw.length<4)return json(res,400,{success:false,error:'Password too short'});
   const plan=String(x.plan||'demo').toLowerCase(); if(!PLANS()[plan])return json(res,400,{success:false,error:'Invalid plan'});
   const now=new Date(); const s={id:sid,name:String(x.name||'New Shop').slice(0,100),address:String(x.address||''),phone:String(x.phone||''),email:String(x.email||''),passwordHash:sha(pw),paused:!!x.paused,supply_warning:false,subscription_expired:false,shop_payment_mode:x.payment_mode||SHOP_DEFAULTS().shopPaymentMode||'both',payment_mode:x.payment_mode||SHOP_DEFAULTS().shopPaymentMode||'both',online_allowed:!!x.online_allowed,color_price:Number(x.color_price??SHOP_DEFAULTS().priceColor),bw_price:Number(x.bw_price??SHOP_DEFAULTS().priceBW),price_color:Number(x.color_price??SHOP_DEFAULTS().priceColor),price_bw:Number(x.bw_price??SHOP_DEFAULTS().priceBW),agent_token:id('agent'),agent_id:null,agent_last_heartbeat:null,agent_printers:[],plan,plan_status:'active',demo_expires_at:plan==='demo'?new Date(now.getTime()+Number(CFG().demo.durationDays||1)*86400000).toISOString():null,demo_print_limit:plan==='demo'?Number(CFG().demo.printLimit):null,demo_prints_used:0,created_at:now.toISOString()};
   db.shops.push(s); db.events.push({id:id('evt'),type:'shop_created',shopId:sid,by:'admin',at:now.toISOString()}); save(); return json(res,200,{success:true,shop:publicShop(sid),password:pw});
 }
 if(p==='/api/admin/shop/rotate-agent'&&req.method==='POST'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'}); const x=await bodyJson(req); const s=db.shops.find(q=>String(q.id).toUpperCase()===String(x.shopId||'').toUpperCase()); if(!s)return json(res,404,{success:false,error:'Shop not found'}); s.agent_token=id('agent');s.agent_id=null;s.agent_last_heartbeat=null;db.events.push({id:id('evt'),type:'agent_token_rotated',shopId:s.id,by:'admin',at:new Date().toISOString()});save();return json(res,200,{success:true,agentToken:s.agent_token});
 }
 if(p==='/api/admin/shop/credentials'&&req.method==='POST'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'}); const x=await bodyJson(req); const s=db.shops.find(q=>String(q.id).toUpperCase()===String(x.shopId||'').toUpperCase()); if(!s)return json(res,404,{success:false,error:'Shop not found'});
   if('password' in x){const pw=String(x.password||'');if(pw.length<4)return json(res,400,{success:false,error:'Password too short'});s.passwordHash=sha(pw);}
   if('razorpay_key_id' in x)s.razorpay_key_id=String(x.razorpay_key_id||'').trim(); if(x.razorpay_key_secret&&x.razorpay_key_secret!=='__KEEP__')s.razorpay_key_secret=String(x.razorpay_key_secret); save();return json(res,200,{success:true,shop:publicShop(s.id)});
 }
 if(p==='/api/admin/shop/update'&&req.method==='POST'){if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'});const x=await bodyJson(req);const s=db.shops.find(q=>String(q.id).toUpperCase()===String(x.shopId||'').toUpperCase());if(!s)return json(res,404,{success:false,error:'Shop not found'});for(const k of ['name','phone','email','plan','plan_status','paused','subscription_expired','bw_price','color_price','price_bw','price_color','payment_mode','shop_payment_mode','payment_gateway','online_allowed'])if(k in x)s[k]=x[k];if(x.plan&&PLANS()[x.plan]){s.plan=x.plan;s.plan_status='active';if(x.plan==='demo'){s.demo_expires_at=new Date(Date.now()+Number(CFG().demo.durationDays||1)*86400000).toISOString();s.demo_print_limit=Number(CFG().demo.printLimit);s.demo_prints_used=0;}}save();return json(res,200,{success:true,shop:publicShop(s.id)});}
 if(p==='/api/admin/shop/reset-password'&&req.method==='POST'){if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'});const x=await bodyJson(req);const s=db.shops.find(q=>String(q.id).toUpperCase()===String(x.shopId||'').toUpperCase());if(!s)return json(res,404,{success:false,error:'Shop not found'});const pw=String(x.password||'');if(pw.length<4)return json(res,400,{success:false,error:'Password too short'});s.passwordHash=sha(pw);save();return json(res,200,{success:true,password:pw});}
 if(p==='/api/admin/shop/delete'&&req.method==='POST'){if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'});const x=await bodyJson(req);const sid=String(x.shopId||'').toUpperCase();if(sid==='DEMO')return json(res,400,{success:false,error:'Default DEMO shop cannot be deleted'});const before=db.shops.length;db.shops=db.shops.filter(s=>String(s.id).toUpperCase()!==sid);db.jobs=db.jobs.filter(j=>String(j.shopId).toUpperCase()!==sid);for(const [t,v] of Object.entries(db.tokens||{}))if(String(v).toUpperCase()===sid)delete db.tokens[t];save();return json(res,200,{success:true,deleted:db.shops.length<before});}
 if(p==='/api/config' && req.method==='GET'){
   return json(res,200,{success:true,app:CFG().app.name,api_version:CFG().app.apiVersion,apiBase:'/api',
      uploadMaxBytes:Number(CFG().server.maxUploadBytes),settings:CFG(),plans:PLANS()});
 }
 if(p==='/api/shop/login'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id.toUpperCase()===String(x.shopId||'').toUpperCase());if(!s||s.passwordHash!==sha(String(x.password||'')))return json(res,401,{success:false,error:'Invalid Shop ID or Password'});const t=id('tok');db.tokens[t]=s.id;save();return json(res,200,{success:true,token:t,shop:publicShop(s.id)});}
 
 if(p==='/api/register/order'&&req.method==='POST'){
   const x=await bodyJson(req);
   const plan=String(x.plan||'').toLowerCase();
   if(!PLANS()[plan]||plan==='demo') return json(res,400,{success:false,error:'Paid plan required'});
   const keyId=process.env.PLATFORM_RAZORPAY_KEY_ID;
   const secret=process.env.PLATFORM_RAZORPAY_KEY_SECRET;
   if(!keyId||!secret) return json(res,503,{success:false,error:'Registration payment is not configured. Add PLATFORM_RAZORPAY_KEY_ID and PLATFORM_RAZORPAY_KEY_SECRET in Render.'});
   if(!x.name||!x.mobile||!x.shopName||!x.email) return json(res,400,{success:false,error:'Name, mobile, shop name and email are required'});
   const regId=id('reg');
   const amount=Math.round(Number(PLANS()[plan].price)*100);
   const payload=JSON.stringify({amount,currency:'INR',receipt:regId,notes:{registrationId:regId,plan}});
   const authz=Buffer.from(`${keyId}:${secret}`).toString('base64');
   try{
     const order=await new Promise((resolve,reject)=>{
       const rq=https.request({hostname:'api.razorpay.com',path:'/v1/orders',method:'POST',headers:{Authorization:`Basic ${authz}`,'Content-Type':'application/json','Content-Length':Buffer.byteLength(payload)}},rr=>{
         let b='';rr.on('data',c=>b+=c);rr.on('end',()=>{try{const d=JSON.parse(b);if(rr.statusCode>=200&&rr.statusCode<300)resolve(d);else reject(new Error(d.error?.description||`Razorpay HTTP ${rr.statusCode}`));}catch(e){reject(e)}})
       });rq.on('error',reject);rq.write(payload);rq.end();
     });
     db.pendingRegistrations[regId]={plan,name:String(x.name).slice(0,100),mobile:String(x.mobile).slice(0,30),shopName:String(x.shopName).slice(0,100),email:String(x.email).slice(0,150),address:String(x.address||'').slice(0,300),orderId:order.id,amount,status:'created',createdAt:new Date().toISOString()};
     save();
     return json(res,200,{success:true,keyId,orderId:order.id,amount,currency:'INR',registrationId:regId,name:x.name,email:x.email,mobile:x.mobile});
   }catch(e){return json(res,502,{success:false,error:'Razorpay order creation failed: '+e.message});}
 }
 if(p==='/api/register/complete'&&req.method==='POST'){
   const x=await bodyJson(req);
   const reg=db.pendingRegistrations&&db.pendingRegistrations[String(x.registrationId||'')];
   if(!reg)return json(res,404,{success:false,error:'Registration session not found or expired'});
   if(reg.status==='paid')return json(res,409,{success:false,error:'Registration already completed'});
   const secret=process.env.PLATFORM_RAZORPAY_KEY_SECRET;
   if(!secret)return json(res,503,{success:false,error:'Registration payment secret is not configured on server'});
   if(String(x.razorpay_order_id||'')!==String(reg.orderId||''))return json(res,400,{success:false,error:'Order mismatch'});
   const expected=crypto.createHmac('sha256',secret).update(`${x.razorpay_order_id}|${x.razorpay_payment_id}`).digest('hex');
   const provided=String(x.razorpay_signature||'');
   if(provided.length!==expected.length||!crypto.timingSafeEqual(Buffer.from(expected),Buffer.from(provided)))return json(res,400,{success:false,error:'Invalid Razorpay payment signature'});
   let sid='SHOP'+Math.random().toString(36).slice(2,7).toUpperCase();
   while(db.shops.some(s=>s.id===sid))sid='SHOP'+Math.random().toString(36).slice(2,7).toUpperCase();
   const pwd=Math.floor(100000+Math.random()*900000).toString();
   const now=new Date();
   const s={id:sid,name:reg.shopName,address:reg.address,phone:reg.mobile,email:reg.email,passwordHash:sha(pwd),paused:false,supply_warning:false,subscription_expired:false,shop_payment_mode:SHOP_DEFAULTS().shopPaymentMode||'both',color_price:Number(SHOP_DEFAULTS().priceColor),bw_price:Number(SHOP_DEFAULTS().priceBW),agent_token:id('agent'),agent_id:null,agent_last_heartbeat:null,agent_printers:[],plan:reg.plan,plan_status:'active',demo_expires_at:null,demo_print_limit:null,demo_prints_used:0,created_at:now.toISOString(),plan_payment_gateway:'razorpay',plan_razorpay_order_id:reg.orderId,plan_razorpay_payment_id:String(x.razorpay_payment_id),plan_paid_at:now.toISOString()};
   db.shops.push(s);reg.status='paid';reg.completedAt=now.toISOString();save();
   return json(res,200,{success:true,paid:true,shopId:sid,password:pwd,plan:s.plan,status:s.plan_status,loginUrl:'/admin',qrCode:shopQrUrl(sid)});
 }
 if(p==='/api/register'&&req.method==='POST'){const x=await bodyJson(req);const plan=String(x.plan||'demo').toLowerCase();if(!PLANS()[plan])return json(res,400,{success:false,error:'Invalid plan'});let sid='SHOP'+Math.random().toString(36).slice(2,7).toUpperCase();while(db.shops.some(s=>s.id===sid))sid='SHOP'+Math.random().toString(36).slice(2,7).toUpperCase();const pwd=String(x.password||Math.floor(100000+Math.random()*900000));const now=new Date();const s={id:sid,name:String(x.shopName||'My Shop').slice(0,100),address:String(x.address||''),phone:String(x.mobile||''),email:String(x.email||''),passwordHash:sha(pwd),paused:false,supply_warning:false,subscription_expired:false,shop_payment_mode:SHOP_DEFAULTS().shopPaymentMode||'both',color_price:Number(SHOP_DEFAULTS().priceColor),bw_price:Number(SHOP_DEFAULTS().priceBW),agent_token:id('agent'),agent_id:null,agent_last_heartbeat:null,agent_printers:[],plan,plan_status:plan==='demo'?'active':'pending_payment',demo_expires_at:plan==='demo'?new Date(now.getTime()+Number(CFG().demo.durationDays||1)*86400000).toISOString():null,demo_print_limit:plan==='demo'?Number(CFG().demo.printLimit):null,demo_prints_used:0,created_at:now.toISOString()};db.shops.push(s);save();return json(res,200,{success:true,shopId:sid,password:pwd,plan,status:s.plan_status,loginUrl:'/admin',qrCode:shopQrUrl(sid)});}
 if(p==='/api/demo/config'&&req.method==='GET'){return json(res,200,{success:true,enabled:!!CFG().demo.enabled,instant:!!CFG().demo.instant,hours:Number(CFG().demo.hours),printLimit:Number(CFG().demo.printLimit)});}
 if(p==='/api/setup-fee/current'&&req.method==='GET'){
    const plans=Object.fromEntries(Object.entries(PLANS()).map(([k,v])=>[k,{fee:Number(v.price||0),actual:Number(v.price||0),name:v.name,advance:!!v.advance,durationDays:Number(v.durationDays||0),printLimit:v.printLimit}]));
    return json(res,200,{success:true,amount:Number(CFG().payments.setupFee),currency:CFG().payments.currency,plans});
  }
 if(p==='/api/public-stats'&&req.method==='GET'){return json(res,200,{success:true,shops:db.shops.length,prints:db.jobs.filter(j=>j.status==='completed').length});}
 if(p==='/api/homepage-config'&&req.method==='GET'){return json(res,200,{success:true,plans:PLANS()});}
 if(p==='/api/demo/request'&&req.method==='POST'){
    if(!CFG().demo.enabled)return json(res,403,{success:false,error:'Demo registration is disabled'});
   const x=await bodyJson(req); let sid='DEMO'+Math.floor(Math.random()*9000+1000); while(db.shops.some(s=>s.id===sid)) sid='DEMO'+Math.floor(Math.random()*9000+1000);
   const pwd=String(x.password||x.mobile||Math.floor(100000+Math.random()*900000)); const created=new Date(); const s={id:sid,name:String(x.shopName||x.name||'Demo Shop').slice(0,100),address:String(x.address||''),phone:String(x.mobile||x.phone||''),email:String(x.email||''),passwordHash:sha(pwd),paused:false,supply_warning:false,subscription_expired:false,shop_payment_mode:SHOP_DEFAULTS().shopPaymentMode||'both',online_allowed:!!SHOP_DEFAULTS().onlineAllowed,color_price:Number(CFG().demo.priceColor),bw_price:Number(CFG().demo.priceBW),agent_token:id('agent'),agent_id:null,agent_last_heartbeat:null,agent_printers:[],plan:'demo',plan_status:'active',demo_expires_at:new Date(created.getTime()+Number(CFG().demo.durationDays||1)*86400000).toISOString(),demo_print_limit:Number(CFG().demo.printLimit),demo_prints_used:0,created_at:created.toISOString()}; db.shops.push(s); save(); return json(res,200,{success:true,approved:true,hours:Number(CFG().demo.hours),printLimit:Number(CFG().demo.printLimit),shopId:sid,password:pwd,loginUrl:'/admin',qrCode:shopQrUrl(sid),shop:publicShop(sid)});
 }
 if(p==='/api/shop/set-password'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id.toUpperCase()===String(x.shopId||'').toUpperCase());if(!s)return json(res,404,{success:false,error:'Shop not found'});if(String(x.newPassword||'').length<4)return json(res,400,{success:false,error:'Password too short'});s.passwordHash=sha(x.newPassword);save();return json(res,200,{success:true});}
 if(p==='/api/shop/insights'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{success:false,error:'Unauthorized'});const js=db.jobs.filter(j=>j.shopId===s.id);const hourCounts={};for(const j of js){const h=new Date(j.created_at||0).toLocaleString('en-IN',{timeZone:'Asia/Kolkata',hour:'2-digit',hour12:false});if(h&&h!=='24')hourCounts[h]=(hourCounts[h]||0)+1;}const peak=Object.entries(hourCounts).sort((a,b)=>b[1]-a[1])[0];return json(res,200,{success:true,totalJobs:js.length,completed:js.filter(j=>j.status==='completed').length,failed:js.filter(j=>j.status==='failed').length,revenue:js.filter(j=>j.payment_status==='paid'||j.payment_status==='counter').reduce((a,j)=>a+Number(j.amount||0),0),peakHour:peak?`${String(peak[0]).padStart(2,'0')}:00`:null,feedbackUp:js.filter(j=>j.feedback==='up').length,feedbackDown:js.filter(j=>j.feedback==='down').length});}
 if(p==='/api/shop/earnings-breakdown'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{success:false,error:'Unauthorized'});const js=db.jobs.filter(j=>j.shopId===s.id);const paidJobs=js.filter(j=>['paid','counter'].includes(String(j.payment_status||'').toLowerCase()));const now=new Date();const dayKey=d=>new Intl.DateTimeFormat('en-CA',{timeZone:'Asia/Kolkata',year:'numeric',month:'2-digit',day:'2-digit'}).format(d);const start=new Date(now.getTime()-6*86400000),startKey=dayKey(start),prevStart=new Date(now.getTime()-13*86400000),prevStartKey=dayKey(prevStart);const thisWeekJobs=paidJobs.filter(j=>dayKey(new Date(j.created_at||0))>=startKey);const lastWeekJobs=paidJobs.filter(j=>dayKey(new Date(j.created_at||0))>=prevStartKey&&dayKey(new Date(j.created_at||0))<startKey);const thisWeek=thisWeekJobs.reduce((a,j)=>a+Number(j.amount||0),0);const lastWeek=lastWeekJobs.reduce((a,j)=>a+Number(j.amount||0),0);const units=j=>Math.max(1,Number(j.totalPages||1))*Math.max(1,Number(j.copies||1));const daily=[];for(let i=6;i>=0;i--){const d=new Date(now.getTime()-i*86400000),k=dayKey(d),dayJobs=paidJobs.filter(j=>dayKey(new Date(j.created_at||0))===k),completed=js.filter(j=>j.status==='completed'&&dayKey(new Date(j.completed_at||j.created_at||0))===k);daily.push({day:k,orders:dayJobs.length,prints:completed.reduce((a,j)=>a+units(j),0),earnings:dayJobs.reduce((a,j)=>a+Number(j.amount||0),0)});}return json(res,200,{success:true,bw:paidJobs.filter(j=>!String(j.colorMode||'').toLowerCase().includes('color')).reduce((a,j)=>a+Number(j.amount||0),0),color:paidJobs.filter(j=>String(j.colorMode||'').toLowerCase().includes('color')).reduce((a,j)=>a+Number(j.amount||0),0),this_week:thisWeek,last_week:lastWeek,daily});}
 const statm=p.match(/^\/api\/shop\/([^/]+)\/stats$/);if(statm&&req.method==='GET'){const s=auth(req);if(!s||String(s.id).toUpperCase()!==String(decodeURIComponent(statm[1])).toUpperCase())return json(res,401,{success:false,error:'Unauthorized'});const js=db.jobs.filter(j=>j.shopId===s.id);const paid=j=>['paid','counter'].includes(String(j.payment_status||'').toLowerCase());const dayKey=d=>new Intl.DateTimeFormat('en-CA',{timeZone:'Asia/Kolkata',year:'numeric',month:'2-digit',day:'2-digit'}).format(d);const today=dayKey(new Date()),yesterday=dayKey(new Date(Date.now()-86400000));const todayPaid=js.filter(j=>dayKey(new Date(j.created_at||0))===today&&paid(j)),yPaid=js.filter(j=>dayKey(new Date(j.created_at||0))===yesterday&&paid(j));const units=j=>Math.max(1,Number(j.totalPages||1))*Math.max(1,Number(j.copies||1));return json(res,200,{success:true,today_prints:js.filter(j=>dayKey(new Date(j.completed_at||j.created_at||0))===today&&j.status==='completed').reduce((a,j)=>a+units(j),0),today_earnings:todayPaid.reduce((a,j)=>a+Number(j.amount||0),0),prev_prints:js.filter(j=>dayKey(new Date(j.completed_at||j.created_at||0))===yesterday&&j.status==='completed').reduce((a,j)=>a+units(j),0),prev_earnings:yPaid.reduce((a,j)=>a+Number(j.amount||0),0),total_orders:js.length,total_earnings:js.filter(paid).reduce((a,j)=>a+Number(j.amount||0),0)});}
 const feedbackm=p.match(/^\/api\/jobs\/([^/]+)\/feedback$/);if(feedbackm&&req.method==='POST'){const s=auth(req);const j=db.jobs.find(q=>q.id===decodeURIComponent(feedbackm[1])&&q.shopId===s?.id);if(!s||!j)return json(res,404,{success:false,error:'Job not found'});const x=await bodyJson(req);const v=String(x.feedback||x.value||'').toLowerCase();if(!['up','down'].includes(v))return json(res,400,{success:false,error:'Invalid feedback'});j.feedback=v;j.feedback_at=new Date().toISOString();save();return json(res,200,{success:true});}
 const sm=p.match(/^\/api\/shop\/([^/]+)$/);if(sm&&req.method==='GET'){const s=publicShop(decodeURIComponent(sm[1]));if(!s)return json(res,404,{error:'Shop Nahi Mila'});return json(res,200,s);}
 const as=p.match(/^\/api\/shop\/([^/]+)\/agent-status$/);if(as&&req.method==='GET'){const sid=decodeURIComponent(as[1]);const s=db.shops.find(q=>String(q.id).toUpperCase()===String(sid).toUpperCase());if(!s)return json(res,404,{success:false,error:'Shop not found'});const hb=s.agent_last_heartbeat?Date.parse(s.agent_last_heartbeat):NaN;const online=Number.isFinite(hb)&&(Date.now()-hb<Number(CFG().health.agentOnlineSeconds||45)*1000);return json(res,200,{success:true,online,seconds_ago:Number.isFinite(hb)?Math.max(0,Math.floor((Date.now()-hb)/1000)):null,last_heartbeat:s.agent_last_heartbeat||null});}
 if(p==='/api/admin/profile'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{error:'Unauthorized'});return json(res,200,publicShop(s.id));}
 if(p==='/api/settings'&&req.method==='GET')return json(res,200,{success:true,settings:CFG(),plans:PLANS()});
 if(p==='/api/printer-models'&&req.method==='GET')return json(res,200,{success:true,models:CFG().printers.models});
 if(p==='/api/track'&&req.method==='POST'){try{const x=await bodyJson(req);db.events.push({at:new Date().toISOString(),...x});db.events=db.events.slice(-5000);save();}catch(e){}return json(res,200,{success:true});}
 if(p==='/api/upload/sign'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id===x.shopId);if(!s)return json(res,404,{success:false,error:'Shop not found'});if(Number(x.fileSize)>Number(CFG().server.maxUploadBytes))return json(res,413,{success:false,error:'File too large'});const publicId=id('file');return json(res,200,{success:true,publicId,apiKey:'local',timestamp:Math.floor(Date.now()/1000),signature:'local',uploadUrl:'/api/upload',uploadToken:id('up')});}
 if(p==='/api/upload'&&req.method==='POST'){const b=await readBody(req,Number(CFG().server.maxUploadBytes)+5*1024*1024);const mp=parseMultipart(b,req.headers['content-type']);if(!mp.file)return json(res,400,{success:false,error:'File missing'});const fid=id('file');const ext=path.extname(mp.file.name||'').toLowerCase()||'.bin';const safe=fid+ext;fs.writeFileSync(path.join(UPLOAD_DIR,safe),mp.file.data);return json(res,200,{success:true,public_id:fid,secure_url:'/files/'+safe,url:'/files/'+safe,uploadToken:mp.fields.uploadToken||id('up')});}
 if(p==='/api/upload/confirm'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id===x.shopId);if(!s)return json(res,404,{success:false,error:'Shop not found'});const job={id:'JOB-'+Date.now().toString(36).toUpperCase(),shopId:s.id,status:'waiting',payment_status:'unpaid',file_url:x.secureUrl||'',file_name:x.fileName||'document',publicId:x.publicId||'',copies:Number(x.copies||1),colorMode:x.colorMode||'bw',totalPages:Number(x.totalPages||1),paperSize:x.paperSize||'A4',orientation:x.orientation||'portrait',duplex:!!x.duplex,amount:calcAmount(x),created_at:new Date().toISOString(),accepted_at:null,completed_at:null};db.jobs.push(job);save();return json(res,200,{success:true,jobId:job.id,amount:job.amount});}
 const jobm=p.match(/^\/api\/jobs\/([^/]+)\/feedback$/);if(jobm&&req.method==='POST'){return json(res,200,{success:true});}
 if(p==='/api/payment/counter'&&req.method==='POST'){const x=await bodyJson(req);const j=db.jobs.find(q=>q.id===x.jobId);if(!j)return json(res,404,{success:false,error:'Job not found'});j.status='waiting';j.payment_status='counter';j.amount=calcAmount({...x,shopId:j.shopId,totalPages:x.totalPages,copies:x.copies,colorMode:x.colorMode});save();return json(res,200,{success:true,amount:j.amount});}
 if(p==='/api/payment/online/create'&&req.method==='POST'){const x=await bodyJson(req);const j=db.jobs.find(q=>q.id===x.jobId);if(!j)return json(res,404,{success:false,error:'Job not found'});const s=db.shops.find(q=>q.id===j.shopId);if(!s)return json(res,404,{success:false,error:'Shop not found'});const keyId=s.razorpay_key_id;const secret=s.razorpay_key_secret;if(!keyId||!secret)return json(res,503,{success:false,error:'Online payment is not configured. Razorpay Key ID/Secret missing on server.'});const amount=Math.round(Number(calcAmount({...x,shopId:j.shopId,totalPages:x.totalPages||j.totalPages,copies:x.copies||j.copies,colorMode:x.colorMode||j.colorMode}))*100);const payload=JSON.stringify({amount,currency:'INR',receipt:j.id,notes:{shopId:s.id,jobId:j.id}});const authz=Buffer.from(`${keyId}:${secret}`).toString('base64');const order=await new Promise((resolve,reject)=>{const rq=https.request({hostname:'api.razorpay.com',path:'/v1/orders',method:'POST',headers:{Authorization:`Basic ${authz}`,'Content-Type':'application/json','Content-Length':Buffer.byteLength(payload)}},rr=>{let b='';rr.on('data',c=>b+=c);rr.on('end',()=>{try{const d=JSON.parse(b); if(rr.statusCode>=200&&rr.statusCode<300)resolve(d); else reject(new Error(d.error?.description||`Razorpay HTTP ${rr.statusCode}`));}catch(e){reject(e)}})});rq.on('error',reject);rq.write(payload);rq.end();});j.payment_gateway='razorpay';j.razorpay_order_id=order.id;j.amount=amount/100;save();return json(res,200,{success:true,gateway:'razorpay',keyId,orderId:order.id,amount, currency:'INR'});}
 if(p==='/api/payment/razorpay/verify'&&req.method==='POST'){const x=await bodyJson(req);const j=db.jobs.find(q=>q.id===x.jobId);if(!j)return json(res,404,{success:false,error:'Job not found'});const s=db.shops.find(q=>q.id===j.shopId);if(!s)return json(res,404,{success:false,error:'Shop not found'});const secret=s.razorpay_key_secret;if(!secret)return json(res,503,{success:false,error:'Razorpay secret missing on server'});const expected=crypto.createHmac('sha256',secret).update(`${x.razorpay_order_id}|${x.razorpay_payment_id}`).digest('hex');const provided=String(x.razorpay_signature||'');if(provided.length!==expected.length||!crypto.timingSafeEqual(Buffer.from(expected),Buffer.from(provided)))return json(res,400,{success:false,error:'Invalid Razorpay signature'});if(j.razorpay_order_id!==x.razorpay_order_id)return json(res,400,{success:false,error:'Order mismatch'});j.payment_status='paid';j.status='waiting';j.razorpay_payment_id=x.razorpay_payment_id;j.razorpay_signature=x.razorpay_signature;j.paid_at=new Date().toISOString();save();return json(res,200,{success:true,status:'PAID',jobId:j.id});}

 const qrm=p.match(/^\/api\/shop\/([^/]+)\/qr$/);if(qrm&&req.method==='GET'){const sid=decodeURIComponent(qrm[1]);const s=db.shops.find(q=>String(q.id).toUpperCase()===sid.toUpperCase());if(!s)return text(res,404,'Shop not found');const origin=`${u.protocol==='https:'?'https':'http'}://${req.headers.host}`;const target=`${origin}/print/${encodeURIComponent(s.id)}`;const svg=qrSvg(target);res.writeHead(200,{'Content-Type':'image/svg+xml; charset=utf-8','Cache-Control':'public, max-age=300'});return res.end(svg);}
 if(p==='/api/shop/pause'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false,error:'Unauthorized'});const x=await bodyJson(req);s.paused=!!x.paused;save();return json(res,200,{success:true,paused:s.paused});}
 if(p==='/api/shop/supply-warning'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false,error:'Unauthorized'});const x=await bodyJson(req);s.supply_warning=x.warning||'';save();return json(res,200,{success:true,warning:s.supply_warning});}
 if(p==='/api/admin/jobs'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{error:'Unauthorized'});return json(res,200,{success:true,jobs:db.jobs.filter(j=>j.shopId===s.id).sort((a,b)=>b.created_at.localeCompare(a.created_at))});}
 if(p==='/api/agent/jobs'&&req.method==='GET'){
   const sid=u.searchParams.get('shopId')||u.searchParams.get('shop_id'); const token=u.searchParams.get('token')||u.searchParams.get('agentToken');
   const s=db.shops.find(q=>q.id===sid&&q.agent_token===token);
   if(!s)return json(res,401,{success:false,error:'Bad agent credentials'});
   const jobs=db.jobs.filter(j=>j.shopId===sid&&['waiting','printing'].includes(j.status)).sort((a,b)=>a.created_at.localeCompare(b.created_at));
   return json(res,200,{success:true,jobs});
 }
 if(p==='/api/agent/job-status'&&req.method==='GET'){
   const sid=u.searchParams.get('shopId')||u.searchParams.get('shop_id'); const token=u.searchParams.get('token')||u.searchParams.get('agentToken'); const jid=u.searchParams.get('jobId')||u.searchParams.get('job_id');
   const s=db.shops.find(q=>q.id===sid&&q.agent_token===token); const j=db.jobs.find(q=>q.id===jid&&q.shopId===sid);
   if(!s||!j)return json(res,404,{success:false,error:'Job not found'}); return json(res,200,{success:true,job:j});
 }
 if(p==='/api/agent/file'&&req.method==='GET'){
   const sid=u.searchParams.get('shopId')||u.searchParams.get('shop_id'); const token=u.searchParams.get('token')||u.searchParams.get('agentToken'); const jid=u.searchParams.get('jobId')||u.searchParams.get('job_id');
   const s=db.shops.find(q=>q.id===sid&&q.agent_token===token); const j=db.jobs.find(q=>q.id===jid&&q.shopId===sid);
   if(!s||!j)return json(res,404,{success:false,error:'Job not found'});
   const filePath=j.file_url && j.file_url.startsWith('/files/') ? path.join(ROOT,j.file_url.replace(/^\/files\//,'')) : null;
   if(!filePath || !filePath.startsWith(UPLOAD_DIR) || !fs.existsSync(filePath)) return json(res,404,{success:false,error:'File not found'});
   res.writeHead(200,{'Content-Type':mime(filePath),'Content-Disposition':'attachment; filename="'+path.basename(j.file_name||filePath)+'"','Cache-Control':'no-store'}); return fs.createReadStream(filePath).pipe(res);
 }
 if(p==='/api/agent/login'&&req.method==='POST'){const x=await bodyJson(req);const sid=String(x.shopId||'').toUpperCase();const s=db.shops.find(q=>String(q.id).toUpperCase()===sid);if(!s||s.passwordHash!==sha(String(x.password||'')))return json(res,401,{success:false,error:'Invalid Shop ID or Password'});s.agent_id=s.agent_id||id('agent');s.agent_last_heartbeat=new Date().toISOString();s.agent_printers=s.agent_printers||[];save();return json(res,200,{success:true,shopId:s.id,shopName:s.name||'Shop',plan:s.plan||'demo',agentToken:s.agent_token||'',agentId:s.agent_id,pollSeconds:Number(CFG().agent.pollSeconds||5),version:String(CFG().agent.version||'')});}
 if(p==='/api/agent/heartbeat'&&req.method==='POST'){const x=await bodyJson(req);const sid=String(x.shopId||'').toUpperCase();const s=db.shops.find(q=>String(q.id).toUpperCase()===sid&&q.agent_token===x.token);if(!s)return json(res,401,{success:false,error:'Bad agent credentials'});s.agent_id=String(x.agentId||s.agent_id||id('agent'));s.agent_last_heartbeat=new Date().toISOString();s.agent_version=x.version||s.agent_version||'';save();return json(res,200,{success:true,online:true,serverTime:new Date().toISOString()});}
 if(p==='/api/agent/printers'&&req.method==='POST'){const x=await bodyJson(req);const sid=String(x.shopId||'').toUpperCase();const s=db.shops.find(q=>String(q.id).toUpperCase()===sid&&q.agent_token===x.token);if(!s)return json(res,401,{success:false,error:'Bad agent credentials'});s.agent_printers=Array.isArray(x.printers)?x.printers:[];s.agent_last_heartbeat=new Date().toISOString();save();return json(res,200,{success:true,printers:s.agent_printers});}
 if(p==='/api/admin/printers'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{success:false,error:'Unauthorized'});const names=[];for(const p0 of (s.agent_printers||[])){const n=typeof p0==='string'?p0:p0?.name;if(n&&!names.includes(n))names.push(n)};for(const k of ['printer_name_bw','printer_name_color','printer_name_4x6','printer_name_a3','printer_name_duplex']){if(s[k]&&!names.includes(s[k]))names.push(s[k]);}return json(res,200,{success:true,printers:names,online:!!s.agent_last_heartbeat&&Date.now()-Date.parse(s.agent_last_heartbeat)<Number(CFG().health.agentOnlineSeconds||45)*1000});}
 if(p==='/api/agent/status'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{success:false});const hb=s.agent_last_heartbeat?Date.parse(s.agent_last_heartbeat):NaN;const online=Number.isFinite(hb)&&(Date.now()-hb<Number(CFG().health.agentOnlineSeconds||45)*1000);const active=db.jobs.some(j=>j.shopId===s.id&&j.status==='printing');return json(res,200,{success:true,online,printing:active,seconds_ago:Number.isFinite(hb)?Math.max(0,Math.floor((Date.now()-hb)/1000)):null,last_heartbeat:s.agent_last_heartbeat||null});}
 if(p==='/api/agent/join'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false});s.agent_id=id('agent');save();return json(res,200,{success:true,agentId:s.agent_id,agentToken:s.agent_token||'agent_demo'});}
 const poll=p.match(/^\/api\/agent\/poll$/);if(poll&&req.method==='GET'){const sid=u.searchParams.get('shopId');const token=u.searchParams.get('token');const s=db.shops.find(q=>q.id===sid&&q.agent_token===token);if(!s)return json(res,401,{success:false,error:'Bad agent credentials'});s.agent_id=s.agent_id||id('agent');const j=db.jobs.find(q=>q.shopId===sid&&q.status==='waiting'&&(['counter','paid'].includes(q.payment_status)));return json(res,200,{success:true,job:j||null});}
 if(p==='/api/agent/claim'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id===x.shopId&&q.agent_token===x.token);const j=db.jobs.find(q=>q.id===x.jobId&&q.shopId===x.shopId);if(!s||!j)return json(res,401,{success:false,error:'Invalid claim'});if(j.status!=='waiting')return json(res,409,{success:false,error:'Job is not waiting'}); normalizeShop(s); if(s.plan==='demo'){if(s.demo_expires_at&&Date.now()>Date.parse(s.demo_expires_at))return json(res,403,{success:false,error:'Demo expired'});const copies=Math.max(1,Number(j.copies||1));if(Number(s.demo_prints_used||0)+copies>Number(s.demo_print_limit||10))return json(res,403,{success:false,error:'Demo print limit reached'});} j.status='printing';j.accepted_at=new Date().toISOString();save();return json(res,200,{success:true,fileUrl:j.file_url});}
 if(p==='/api/agent/complete'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id===x.shopId&&q.agent_token===x.token);const j=db.jobs.find(q=>q.id===x.jobId&&q.shopId===x.shopId);if(!s||!j)return json(res,401,{success:false});j.status=x.success?'completed':'failed';j.completed_at=new Date().toISOString();j.error=x.error||null;if(x.success&&s.plan==='demo')s.demo_prints_used=Number(s.demo_prints_used||0)+Math.max(1,Number(j.copies||1));save();return json(res,200,{success:true});}
 if(p==='/api/agent/reject'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id===x.shopId&&q.agent_token===x.token);const j=db.jobs.find(q=>q.id===x.jobId&&q.shopId===x.shopId);if(!s||!j)return json(res,401,{success:false});j.status='rejected';j.error=x.error||'Rejected by operator';save();return json(res,200,{success:true});}
 if(p==='/api/admin/settings'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{success:false,error:'Unauthorized'});return json(res,200,{success:true,shop:publicShop(s.id)});}
 if(p==='/api/admin/settings'&&(req.method==='POST'||req.method==='PUT')){
   const s=auth(req); if(!s)return json(res,401,{success:false,error:'Unauthorized'});
   const x=await bodyJson(req);
   const fields=['name','address','phone','email','color_price','bw_price','price_bw','price_color','price_4x6_4','price_4x6_6','price_4x6_8','price_4x6_10','price_resume_color','price_resume_bw','price_a3_bw','price_a3_color','price_a2_bw','price_a2_color','price_a1_bw','price_a1_color','duplex_mode','price_bw_duplex','price_color_duplex','duplex_bw_enabled','duplex_color_enabled','printer_model','printer_name_bw','printer_name_color','printer_name_4x6','printer_name_a3','printer_name_duplex','payment_mode','payment_gateway','razorpay_key_id','cashfree_app_id','paused','supply_warning'];
   for(const k of fields){ if(!(k in x)) continue; if(k==='price_bw') s.bw_price=Number(x[k]); else if(k==='price_color') s.color_price=Number(x[k]); else if(k==='payment_mode') s.shop_payment_mode=x[k]; else s[k]=x[k]; }
   if('bw_price' in x) s.bw_price=Number(x.bw_price); if('color_price' in x) s.color_price=Number(x.color_price);
   if('price_bw' in x) s.price_bw=Number(x.price_bw); else s.price_bw=s.bw_price;
   if('price_color' in x) s.price_color=Number(x.price_color); else s.price_color=s.color_price;
   if('payment_mode' in x) s.payment_mode=x.payment_mode; else s.payment_mode=s.shop_payment_mode;
   if('razorpay_key_secret' in x && x.razorpay_key_secret && x.razorpay_key_secret!=='__KEEP__') s.razorpay_key_secret=x.razorpay_key_secret;
   if('cashfree_secret_key' in x && x.cashfree_secret_key && x.cashfree_secret_key!=='__KEEP__') s.cashfree_secret_key=x.cashfree_secret_key;
   save(); return json(res,200,{success:true,shop:publicShop(s.id)});
 }
 if(p==='/api/admin/change-password'&&(req.method==='POST'||req.method==='PUT')){const s=auth(req);if(!s)return json(res,401,{success:false});const x=await bodyJson(req);s.passwordHash=sha(String(x.newPassword||''));save();return json(res,200,{success:true});}
 if(p==='/api/admin/agent/disconnect'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false});s.agent_id=null;save();return json(res,200,{success:true});}
 if(p==='/api/shop/notice'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{success:false,error:'Unauthorized'});return json(res,200,{success:true,notice:s.notice||s.shop_notice||''});}
 if(p==='/api/shop/notice'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false,error:'Unauthorized'});const x=await bodyJson(req);const msg=String(x.notice||'').trim().slice(0,500);s.notice=msg;s.shop_notice=msg;save();return json(res,200,{success:true,notice:msg});}
 if(p==='/api/advance-features'&&req.method==='GET'){return json(res,200,{success:true,features:CFG().features.advance});}
 if(p==='/api/shop/advance-active'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{success:false});normalizeShop(s);return json(res,200,{success:true,active:s.plan==='demo'?true:s.advanced_active!==false});}
 if(p==='/api/shop/advance-active'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false});normalizeShop(s);const x=await bodyJson(req);if(s.plan!=='demo'&&!s.advanced_unlocked)return json(res,403,{success:false,error:'Advance Feature locked'});s.advanced_active=!!x.active;save();return json(res,200,{success:true,active:s.advanced_active});}
 if(p==='/api/shop/advance-module'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{success:false});normalizeShop(s);return json(res,200,{success:true,active:s.plan==='demo'?true:s.advanced_unlocked===true});}
 if(p==='/api/shop/advance-module'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false});normalizeShop(s);if(s.plan!=='demo'&&!s.advanced_unlocked)return json(res,403,{success:false,error:'Advance Feature locked'});const x=await bodyJson(req);const map={legal:'adv_legal_active',resume:'adv_resume_active','4x6':'adv_4x6_active',a3:'adv_a3_active',mini:'adv_mini_active'};const field=map[String(x.module||'')];if(!field)return json(res,400,{success:false,error:'Invalid module'});s[field]=x.active!==false;save();const modules={};for(const f of Object.values(map))modules[f]=s[f]!==false;return json(res,200,{success:true,modules});}
 if(p==='/api/admin/settings'&&req.method==='GET'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'}); return json(res,200,{success:true,settings:CFG(),plans:PLANS()});
 }
 if(p==='/api/admin/settings'&&(req.method==='POST'||req.method==='PUT')){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'}); const x=await bodyJson(req); const patch=x.settings&&typeof x.settings==='object'?x.settings:x; mergeMissing(CFG(),patch); // overwrite only supplied paths
   function deepAssign(dst,src){for(const [k,v] of Object.entries(src)){if(v&&typeof v==='object'&&!Array.isArray(v)&&dst[k]&&typeof dst[k]==='object'&&!Array.isArray(dst[k]))deepAssign(dst[k],v);else dst[k]=v;}}
   deepAssign(CFG(),patch); db.events.push({id:id('evt'),type:'settings_updated',by:'admin',at:new Date().toISOString()}); save(); return json(res,200,{success:true,settings:CFG()});
 }
 if(p==='/api/admin/plans'&&req.method==='POST'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'}); const x=await bodyJson(req); const key=String(x.id||'').toLowerCase().replace(/[^a-z0-9_-]/g,''); if(!key)return json(res,400,{success:false,error:'Plan ID required'}); PLANS()[key]={...(PLANS()[key]||{}),name:String(x.name||key),price:Number(x.price||0),durationDays:Number(x.durationDays||0),printLimit:x.printLimit===null||x.printLimit===''?null:Number(x.printLimit),advance:!!x.advance};save();return json(res,200,{success:true,plans:PLANS()});
 }
 if(p==='/api/admin/jobs'&&req.method==='GET'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'}); const sid=String(u.searchParams.get('shopId')||'').toUpperCase(); const status=String(u.searchParams.get('status')||'').toLowerCase(); const limit=Math.min(500,Math.max(1,Number(u.searchParams.get('limit')||100))); let jobs=db.jobs.slice(); if(sid)jobs=jobs.filter(j=>String(j.shopId).toUpperCase()===sid); if(status)jobs=jobs.filter(j=>String(j.status).toLowerCase()===status); jobs.sort((a,b)=>String(b.created_at).localeCompare(String(a.created_at)));return json(res,200,{success:true,jobs:jobs.slice(0,limit)});
 }
 if(p==='/api/admin/job/cancel'&&req.method==='POST'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'}); const x=await bodyJson(req); const j=db.jobs.find(q=>q.id===x.jobId);if(!j)return json(res,404,{success:false,error:'Job not found'});if(['completed','failed'].includes(j.status))return json(res,400,{success:false,error:'Completed/failed job cannot be cancelled'});j.status='cancelled';j.error='Cancelled by super admin';j.cancelled_at=new Date().toISOString();db.events.push({id:id('evt'),type:'job_cancelled',jobId:j.id,shopId:j.shopId,by:'admin',at:new Date().toISOString()});save();return json(res,200,{success:true});
 }
 if(p==='/api/admin/events'&&req.method==='GET'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'}); const limit=Math.min(500,Math.max(1,Number(u.searchParams.get('limit')||100)));return json(res,200,{success:true,events:(db.events||[]).slice(-limit).reverse()});
 }
 if(p==='/api/admin/broadcast'&&req.method==='POST'){
   if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'}); const x=await bodyJson(req);const msg=String(x.message||'').trim();if(!msg)return json(res,400,{success:false,error:'Message required'});const targets=Array.isArray(x.shopIds)&&x.shopIds.length?x.shopIds.map(v=>String(v).toUpperCase()):db.shops.map(s=>s.id);let n=0;for(const sid of targets){const s=db.shops.find(q=>String(q.id).toUpperCase()===sid);if(s){s.notice=msg;s.shop_notice=msg;n++;}}db.events.push({id:id('evt'),type:'broadcast',targets:targets.length,by:'admin',at:new Date().toISOString()});save();return json(res,200,{success:true,updated:n});
 }
 if(p==='/api/admin/export-data'&&req.method==='POST'){if(!adminAuth(req))return json(res,401,{success:false,error:'Unauthorized'});return json(res,200,{success:true,data:db});}
 if(p==='/api/admin/delete-account'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false,error:'Unauthorized'});db.jobs=db.jobs.filter(j=>j.shopId!==s.id);db.shops=db.shops.filter(q=>q.id!==s.id);for(const [t,v] of Object.entries(db.tokens||{}))if(v===s.id)delete db.tokens[t];save();return json(res,200,{success:true});}
 return json(res,404,{error:'API route not implemented in local edition',path:p});
}

const server=http.createServer(async(req,res)=>{try{
 const u=new URL(req.url,'http://localhost');
 if(req.method==='OPTIONS'){res.writeHead(204,{'Access-Control-Allow-Origin':'*','Access-Control-Allow-Headers':'Content-Type, Authorization','Access-Control-Allow-Methods':'GET,POST,PUT,OPTIONS'});return res.end();}
 if(u.pathname.startsWith('/api/')) return await api(req,res,u);
 if(u.pathname.startsWith('/files/')){const name=path.basename(u.pathname);const f=path.join(UPLOAD_DIR,name);if(!fs.existsSync(f))return text(res,404,'Not found');res.writeHead(200,{'Content-Type':mime(f),'Content-Disposition':'inline'});return fs.createReadStream(f).pipe(res);}
 if(u.pathname==='/'||u.pathname==='/index.html')return routePage(res,'index.html');
 if(u.pathname==='/admin')return routePage(res,'admin.html');
 if(u.pathname==='/dashboard'||u.pathname==='/login')return routePage(res,'dashboard.html');
 if(u.pathname.startsWith('/print/'))return routePage(res,'print.html');
 if(u.pathname==='/print')return routePage(res,'print.html');
 if(u.pathname==='/order')return routePage(res,'order.html');
 if(u.pathname==='/settings')return routePage(res,'settings.html');
 if(u.pathname==='/register')return routePage(res,'register.html');
 if(u.pathname==='/superadmin')return routePage(res,'admin.html');
 if(u.pathname==='/qr-downloads')return routePage(res,'qr-downloads.html');
 if(u.pathname.startsWith('/resume/'))return routePage(res,'resume.html');
 if(u.pathname==='/resume')return routePage(res,'resume.html');
 const fp=path.join(WEB,path.normalize(u.pathname).replace(/^[/\\]+/,''));if(fp.startsWith(WEB)&&fs.existsSync(fp)&&fs.statSync(fp).isFile()){res.writeHead(200,{'Content-Type':mime(fp)});return fs.createReadStream(fp).pipe(res);}text(res,404,'Not found');
}catch(e){console.error(e);json(res,500,{error:'Internal server error',message:e.message});}});
function mime(f){const e=path.extname(f).toLowerCase();return ({'.html':'text/html; charset=utf-8','.js':'application/javascript; charset=utf-8','.css':'text/css','.json':'application/json','.png':'image/png','.jpg':'image/jpeg','.jpeg':'image/jpeg','.svg':'image/svg+xml','.webmanifest':'application/manifest+json','.pdf':'application/pdf'})[e]||'application/octet-stream';}
server.listen(PORT,()=>console.log(`QR Se Print local server: http://localhost:${PORT}`));

setInterval(()=>{
  const cutoff=Date.now()-Number(CFG().uploads.fileMaxAgeMinutes||90)*60*1000;
  try{for(const f of fs.readdirSync(UPLOAD_DIR)){const fp=path.join(UPLOAD_DIR,f);if(fs.statSync(fp).mtimeMs<cutoff)fs.unlinkSync(fp);}}catch(e){}
},Number(CFG().uploads.cleanupMinutes||10)*60*1000);
