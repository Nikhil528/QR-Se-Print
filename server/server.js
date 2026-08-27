const http = require('http');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { URL } = require('url');

const ROOT = path.resolve(__dirname, '..');
const WEB = path.join(ROOT, 'web');
const DATA_DIR = path.join(ROOT, 'data');
const UPLOAD_DIR = path.join(ROOT, 'uploads');
fs.mkdirSync(DATA_DIR,{recursive:true}); fs.mkdirSync(UPLOAD_DIR,{recursive:true});
const DB = path.join(DATA_DIR,'db.json');
const PORT = Number(process.env.PORT || 8080);
const MAX_UPLOAD = Number(process.env.MAX_UPLOAD_BYTES || 25*1024*1024);

function load(){ try{return JSON.parse(fs.readFileSync(DB,'utf8'));}catch(e){return {shops:[],jobs:[],tokens:{},settings:{},events:[]};} }
let db=load();
function save(){fs.writeFileSync(DB+'.tmp',JSON.stringify(db,null,2));fs.renameSync(DB+'.tmp',DB);}
function id(prefix){return prefix+'_'+crypto.randomBytes(8).toString('hex');}
function sha(s){return crypto.createHash('sha256').update(s).digest('hex');}
function json(res,code,obj){const b=Buffer.from(JSON.stringify(obj));res.writeHead(code,{'Content-Type':'application/json; charset=utf-8','Content-Length':b.length,'Cache-Control':'no-store','Access-Control-Allow-Origin':'*','Access-Control-Allow-Headers':'Content-Type, Authorization','Access-Control-Allow-Methods':'GET,POST,PUT,OPTIONS'});res.end(b);}
function text(res,code,s,type='text/plain'){const b=Buffer.from(s);res.writeHead(code,{'Content-Type':type,'Content-Length':b.length});res.end(b);}
function readBody(req,limit=2*1024*1024){return new Promise((resolve,reject)=>{let a=[],n=0;req.on('data',c=>{n+=c.length;if(n>limit){reject(new Error('BODY_TOO_LARGE'));req.destroy();return;}a.push(c);});req.on('end',()=>resolve(Buffer.concat(a)));req.on('error',reject);});}
async function bodyJson(req){const b=await readBody(req);try{return JSON.parse(b.toString('utf8')||'{}')}catch(e){throw new Error('INVALID_JSON')}}
function bearer(req){const h=req.headers.authorization||'';return h.startsWith('Bearer ')?h.slice(7):'';}
function auth(req){const t=bearer(req);const shopId=db.tokens[t];return shopId?db.shops.find(s=>s.id===shopId):null;}
function seed(){if(db.shops.length)return;db.shops=[{id:'DEMO',name:'QR Se Print Demo Shop',address:'',phone:'',email:'',passwordHash:sha('1234'),paused:false,supply_warning:false,subscription_expired:false,shop_payment_mode:'both',online_allowed:false,color_price:5,bw_price:3,agent_token:'agent_demo',agent_id:null,agent_last_heartbeat:null,agent_printers:[],created_at:new Date().toISOString()}];save();}
seed();

function parseMultipart(buf,ctype){
 const m=/boundary=(?:"([^"]+)"|([^;]+))/i.exec(ctype||''); if(!m) throw new Error('MULTIPART_BOUNDARY_MISSING');
 const boundary=Buffer.from('--'+(m[1]||m[2])); const out={fields:{},file:null}; let p=0;
 while((p=buf.indexOf(boundary,p))!==-1){p+=boundary.length;if(buf[p]===45&&buf[p+1]===45)break;if(buf[p]===13&&buf[p+1]===10)p+=2;const end=buf.indexOf(Buffer.from('\r\n\r\n'),p);if(end<0)break;const hs=buf.slice(p,end).toString();let next=buf.indexOf(boundary,end+4);if(next<0)break;let dataEnd=next-2;if(dataEnd<end+4)dataEnd=end+4;const data=buf.slice(end+4,dataEnd);const nm=/name="([^"]+)"/i.exec(hs);if(!nm)continue;const fn=/filename="([^"]*)"/i.exec(hs);if(fn)out.file={name:fn[1],data};else out.fields[nm[1]]=data.toString();p=next;}
 return out;
}
function routePage(res,file){const p=path.join(WEB,file);if(!fs.existsSync(p))return text(res,404,'Not found');res.writeHead(200,{'Content-Type':'text/html; charset=utf-8'});fs.createReadStream(p).pipe(res);}
function publicShop(id0){const s=db.shops.find(x=>x.id.toUpperCase()===String(id0).toUpperCase());if(!s)return null;const out={...s,passwordHash:undefined,agent_token:undefined};out.price_bw=Number(s.price_bw??s.bw_price??3);out.price_color=Number(s.price_color??s.color_price??5);out.bw_price=Number(s.bw_price??out.price_bw);out.color_price=Number(s.color_price??out.price_color);out.payment_mode=s.payment_mode??s.shop_payment_mode??'both';out.has_razorpay_secret=!!s.razorpay_key_secret;out.has_cashfree_secret=!!s.cashfree_secret_key;return out;}
function calcAmount(x){const pages=Math.max(1,Number(x.totalPages||1));const copies=Math.max(1,Number(x.copies||1));const price=(String(x.colorMode||'bw').toLowerCase().includes('color'))?Number(db.shops.find(s=>s.id===x.shopId)?.color_price||5):Number(db.shops.find(s=>s.id===x.shopId)?.bw_price||3);return pages*copies*price;}

async function api(req,res,u){
 const p=u.pathname;
 // Render/monitoring health endpoints: never redirect, always return JSON.
 if((p==='/api/health'||p==='/api/health/') && req.method==='GET'){
   return json(res,200,{success:true,app:'QR Se Print',api_version:'complete-2026.08.26',status:'ok',storage:'local-json',db:false,time:new Date().toISOString()});
 }
 if(p==='/api/index.php' && req.method==='GET' && (u.searchParams.get('route')||'')==='health'){
   return json(res,200,{success:true,app:'QR Se Print',api_version:'complete-2026.08.26',db:false,time:new Date().toISOString()});
 }
 if(p==='/api/config' && req.method==='GET'){
   return json(res,200,{success:true,app:'QR Se Print',api_version:'complete-2026.08.26',apiBase:'/api',uploadMaxBytes:MAX_UPLOAD});
 }
 if(p==='/api/shop/login'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id.toUpperCase()===String(x.shopId||'').toUpperCase());if(!s||s.passwordHash!==sha(String(x.password||'')))return json(res,401,{success:false,error:'Invalid Shop ID or Password'});const t=id('tok');db.tokens[t]=s.id;save();return json(res,200,{success:true,token:t,shop:publicShop(s.id)});}
 if(p==='/api/demo/request'&&req.method==='POST'){
   const x=await bodyJson(req); let sid='DEMO'+Math.floor(Math.random()*9000+1000); while(db.shops.some(s=>s.id===sid)) sid='DEMO'+Math.floor(Math.random()*9000+1000);
   const s={id:sid,name:String(x.name||'Demo Shop').slice(0,100),address:'',phone:String(x.mobile||x.phone||''),email:String(x.email||''),passwordHash:sha('1234'),paused:false,supply_warning:false,subscription_expired:false,shop_payment_mode:'both',online_allowed:false,color_price:10,bw_price:5,agent_token:id('agent'),agent_id:null,created_at:new Date().toISOString()}; db.shops.push(s); save(); return json(res,200,{success:true,shopId:sid,password:'1234',shop:publicShop(sid)});
 }
 if(p==='/api/shop/set-password'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id.toUpperCase()===String(x.shopId||'').toUpperCase());if(!s)return json(res,404,{success:false,error:'Shop not found'});if(String(x.newPassword||'').length<4)return json(res,400,{success:false,error:'Password too short'});s.passwordHash=sha(x.newPassword);save();return json(res,200,{success:true});}
 const sm=p.match(/^\/api\/shop\/([^/]+)$/);if(sm&&req.method==='GET'){const s=publicShop(decodeURIComponent(sm[1]));if(!s)return json(res,404,{error:'Shop Nahi Mila'});return json(res,200,s);}
 const as=p.match(/^\/api\/shop\/([^/]+)\/agent-status$/);if(as&&req.method==='GET'){const sid=decodeURIComponent(as[1]);const s=db.shops.find(q=>String(q.id).toUpperCase()===String(sid).toUpperCase());if(!s)return json(res,404,{success:false,error:'Shop not found'});const hb=s.agent_last_heartbeat?Date.parse(s.agent_last_heartbeat):NaN;const online=Number.isFinite(hb)&&(Date.now()-hb<45000);return json(res,200,{success:true,online,seconds_ago:Number.isFinite(hb)?Math.max(0,Math.floor((Date.now()-hb)/1000)):null,last_heartbeat:s.agent_last_heartbeat||null});}
 if(p==='/api/admin/profile'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{error:'Unauthorized'});return json(res,200,publicShop(s.id));}
 if(p==='/api/settings'&&req.method==='GET')return json(res,200,{success:true});
 if(p==='/api/printer-models'&&req.method==='GET')return json(res,200,{success:true,printers:['Default Printer','Microsoft Print to PDF']});
 if(p==='/api/track'&&req.method==='POST'){try{const x=await bodyJson(req);db.events.push({at:new Date().toISOString(),...x});db.events=db.events.slice(-5000);save();}catch(e){}return json(res,200,{success:true});}
 if(p==='/api/upload/sign'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id===x.shopId);if(!s)return json(res,404,{success:false,error:'Shop not found'});if(Number(x.fileSize)>MAX_UPLOAD)return json(res,413,{success:false,error:'File too large'});const publicId=id('file');return json(res,200,{success:true,publicId,apiKey:'local',timestamp:Math.floor(Date.now()/1000),signature:'local',uploadUrl:'/api/upload',uploadToken:id('up')});}
 if(p==='/api/upload'&&req.method==='POST'){const b=await readBody(req,MAX_UPLOAD+5*1024*1024);const mp=parseMultipart(b,req.headers['content-type']);if(!mp.file)return json(res,400,{success:false,error:'File missing'});const fid=id('file');const ext=path.extname(mp.file.name||'').toLowerCase()||'.bin';const safe=fid+ext;fs.writeFileSync(path.join(UPLOAD_DIR,safe),mp.file.data);return json(res,200,{success:true,public_id:fid,secure_url:'/files/'+safe,url:'/files/'+safe,uploadToken:mp.fields.uploadToken||id('up')});}
 if(p==='/api/upload/confirm'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id===x.shopId);if(!s)return json(res,404,{success:false,error:'Shop not found'});const job={id:'JOB-'+Date.now().toString(36).toUpperCase(),shopId:s.id,status:'waiting',payment_status:'unpaid',file_url:x.secureUrl||'',file_name:x.fileName||'document',publicId:x.publicId||'',copies:Number(x.copies||1),colorMode:x.colorMode||'bw',totalPages:Number(x.totalPages||1),paperSize:x.paperSize||'A4',orientation:x.orientation||'portrait',duplex:!!x.duplex,amount:calcAmount(x),created_at:new Date().toISOString(),accepted_at:null,completed_at:null};db.jobs.push(job);save();return json(res,200,{success:true,jobId:job.id,amount:job.amount});}
 const jobm=p.match(/^\/api\/jobs\/([^/]+)\/feedback$/);if(jobm&&req.method==='POST'){return json(res,200,{success:true});}
 if(p==='/api/payment/counter'&&req.method==='POST'){const x=await bodyJson(req);const j=db.jobs.find(q=>q.id===x.jobId);if(!j)return json(res,404,{success:false,error:'Job not found'});j.status='waiting';j.payment_status='counter';j.amount=calcAmount({...x,shopId:j.shopId,totalPages:x.totalPages,copies:x.copies,colorMode:x.colorMode});save();return json(res,200,{success:true,amount:j.amount});}
 if(p==='/api/payment/online/create'&&req.method==='POST'){const x=await bodyJson(req);const j=db.jobs.find(q=>q.id===x.jobId);if(!j)return json(res,404,{success:false,error:'Job not found'});if(!process.env.RAZORPAY_KEY_ID||!process.env.RAZORPAY_KEY_SECRET)return json(res,503,{success:false,error:'Online payment is not configured. Use Counter Payment or add Razorpay credentials.'});return json(res,501,{success:false,error:'Payment gateway adapter requires production credentials/configuration.'});}
 if(p==='/api/payment/razorpay/verify'&&req.method==='POST')return json(res,501,{success:false,error:'Configure Razorpay server verification first'});
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
 if(p==='/api/agent/login'&&req.method==='POST'){const x=await bodyJson(req);const sid=String(x.shopId||'').toUpperCase();const s=db.shops.find(q=>String(q.id).toUpperCase()===sid);if(!s||s.passwordHash!==sha(String(x.password||'')))return json(res,401,{success:false,error:'Invalid Shop ID or Password'});s.agent_id=s.agent_id||id('agent');s.agent_last_heartbeat=new Date().toISOString();s.agent_printers=s.agent_printers||[];save();return json(res,200,{success:true,shopId:s.id,agentToken:s.agent_token||'',agentId:s.agent_id});}
 if(p==='/api/agent/heartbeat'&&req.method==='POST'){const x=await bodyJson(req);const sid=String(x.shopId||'').toUpperCase();const s=db.shops.find(q=>String(q.id).toUpperCase()===sid&&q.agent_token===x.token);if(!s)return json(res,401,{success:false,error:'Bad agent credentials'});s.agent_id=String(x.agentId||s.agent_id||id('agent'));s.agent_last_heartbeat=new Date().toISOString();s.agent_version=x.version||s.agent_version||'';save();return json(res,200,{success:true,online:true,serverTime:new Date().toISOString()});}
 if(p==='/api/agent/printers'&&req.method==='POST'){const x=await bodyJson(req);const sid=String(x.shopId||'').toUpperCase();const s=db.shops.find(q=>String(q.id).toUpperCase()===sid&&q.agent_token===x.token);if(!s)return json(res,401,{success:false,error:'Bad agent credentials'});s.agent_printers=Array.isArray(x.printers)?x.printers:[];s.agent_last_heartbeat=new Date().toISOString();save();return json(res,200,{success:true,printers:s.agent_printers});}
 if(p==='/api/admin/printers'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{success:false,error:'Unauthorized'});const names=[];for(const p0 of (s.agent_printers||[])){const n=typeof p0==='string'?p0:p0?.name;if(n&&!names.includes(n))names.push(n)};for(const k of ['printer_name_bw','printer_name_color','printer_name_4x6','printer_name_a3','printer_name_duplex']){if(s[k]&&!names.includes(s[k]))names.push(s[k]);}return json(res,200,{success:true,printers:names,online:!!s.agent_last_heartbeat&&Date.now()-Date.parse(s.agent_last_heartbeat)<45000});}
 if(p==='/api/agent/status'&&req.method==='GET'){const s=auth(req);if(!s)return json(res,401,{success:false});const hb=s.agent_last_heartbeat?Date.parse(s.agent_last_heartbeat):NaN;const online=Number.isFinite(hb)&&(Date.now()-hb<45000);const active=db.jobs.some(j=>j.shopId===s.id&&j.status==='printing');return json(res,200,{success:true,online,printing:active,seconds_ago:Number.isFinite(hb)?Math.max(0,Math.floor((Date.now()-hb)/1000)):null,last_heartbeat:s.agent_last_heartbeat||null});}
 if(p==='/api/agent/join'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false});s.agent_id=id('agent');save();return json(res,200,{success:true,agentId:s.agent_id,agentToken:s.agent_token||'agent_demo'});}
 const poll=p.match(/^\/api\/agent\/poll$/);if(poll&&req.method==='GET'){const sid=u.searchParams.get('shopId');const token=u.searchParams.get('token');const s=db.shops.find(q=>q.id===sid&&q.agent_token===token);if(!s)return json(res,401,{success:false,error:'Bad agent credentials'});s.agent_id=s.agent_id||id('agent');const j=db.jobs.find(q=>q.shopId===sid&&q.status==='waiting'&&(['counter','paid'].includes(q.payment_status)));return json(res,200,{success:true,job:j||null});}
 if(p==='/api/agent/claim'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id===x.shopId&&q.agent_token===x.token);const j=db.jobs.find(q=>q.id===x.jobId&&q.shopId===x.shopId);if(!s||!j)return json(res,401,{success:false,error:'Invalid claim'});if(j.status!=='waiting')return json(res,409,{success:false,error:'Job is not waiting'});j.status='printing';j.accepted_at=new Date().toISOString();save();return json(res,200,{success:true,fileUrl:j.file_url});}
 if(p==='/api/agent/complete'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id===x.shopId&&q.agent_token===x.token);const j=db.jobs.find(q=>q.id===x.jobId&&q.shopId===x.shopId);if(!s||!j)return json(res,401,{success:false});j.status=x.success?'completed':'failed';j.completed_at=new Date().toISOString();j.error=x.error||null;save();return json(res,200,{success:true});}
 if(p==='/api/agent/reject'&&req.method==='POST'){const x=await bodyJson(req);const s=db.shops.find(q=>q.id===x.shopId&&q.agent_token===x.token);const j=db.jobs.find(q=>q.id===x.jobId&&q.shopId===x.shopId);if(!s||!j)return json(res,401,{success:false});j.status='rejected';j.error=x.error||'Rejected by operator';save();return json(res,200,{success:true});}
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
 if(p==='/api/admin/change-password'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false});const x=await bodyJson(req);s.passwordHash=sha(String(x.newPassword||''));save();return json(res,200,{success:true});}
 if(p==='/api/admin/agent/disconnect'&&req.method==='POST'){const s=auth(req);if(!s)return json(res,401,{success:false});s.agent_id=null;save();return json(res,200,{success:true});}
 return json(res,404,{error:'API route not implemented in local edition',path:p});
}

const server=http.createServer(async(req,res)=>{try{
 const u=new URL(req.url,'http://localhost');
 if(req.method==='OPTIONS'){res.writeHead(204,{'Access-Control-Allow-Origin':'*','Access-Control-Allow-Headers':'Content-Type, Authorization','Access-Control-Allow-Methods':'GET,POST,PUT,OPTIONS'});return res.end();}
 if(u.pathname.startsWith('/api/')) return await api(req,res,u);
 if(u.pathname.startsWith('/files/')){const name=path.basename(u.pathname);const f=path.join(UPLOAD_DIR,name);if(!fs.existsSync(f))return text(res,404,'Not found');res.writeHead(200,{'Content-Type':mime(f),'Content-Disposition':'inline'});return fs.createReadStream(f).pipe(res);}
 if(u.pathname==='/'||u.pathname==='/index.html')return routePage(res,'index.html');
 if(u.pathname==='/admin'||u.pathname==='/dashboard'||u.pathname==='/login')return routePage(res,'dashboard.html');
 if(u.pathname.startsWith('/print/'))return routePage(res,'print.html');
 if(u.pathname==='/print')return routePage(res,'print.html');
 if(u.pathname==='/order')return routePage(res,'order.html');
 if(u.pathname==='/settings')return routePage(res,'settings.html');
 if(u.pathname==='/qr-downloads')return routePage(res,'qr-downloads.html');
 const fp=path.join(WEB,path.normalize(u.pathname).replace(/^[/\\]+/,''));if(fp.startsWith(WEB)&&fs.existsSync(fp)&&fs.statSync(fp).isFile()){res.writeHead(200,{'Content-Type':mime(fp)});return fs.createReadStream(fp).pipe(res);}text(res,404,'Not found');
}catch(e){console.error(e);json(res,500,{error:'Internal server error',message:e.message});}});
function mime(f){const e=path.extname(f).toLowerCase();return ({'.html':'text/html; charset=utf-8','.js':'application/javascript; charset=utf-8','.css':'text/css','.json':'application/json','.png':'image/png','.jpg':'image/jpeg','.jpeg':'image/jpeg','.svg':'image/svg+xml','.webmanifest':'application/manifest+json','.pdf':'application/pdf'})[e]||'application/octet-stream';}
server.listen(PORT,()=>console.log(`QR Se Print local server: http://localhost:${PORT}`));

setInterval(()=>{
  const cutoff=Date.now()-90*60*1000;
  try{for(const f of fs.readdirSync(UPLOAD_DIR)){const fp=path.join(UPLOAD_DIR,f);if(fs.statSync(fp).mtimeMs<cutoff)fs.unlinkSync(fp);}}catch(e){}
},10*60*1000);
