// Migration uses the actual persistence functions with an abortable IndexedDB
// transaction model. Commits publish both stores together, just as IDB does.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
const source=fs.readFileSync(new URL('../js/05-persistence.js',import.meta.url),'utf8');
const copy=x=>x===undefined?undefined:structuredClone(x);
const old={threads:[{id:'a',messages:[{role:'user',content:'old chat'}]}],characterCards:[{id:'g',name:'Goblin'}],personaCards:[{id:'p',name:'Reader'}],settings:{theme:'green'}};
function harness({meta,threads=[],legacy,legacyKey='state',abort=false,readFailure=false}={}) {
 const disk={meta:new Map(meta?[['app',copy(meta)]]:[]),threads:new Map(threads.map(t=>[t.id,copy(t)]))};
 const local=new Map(legacy===undefined?[]:[[legacyKey,typeof legacy==='string'?legacy:JSON.stringify(legacy)]]);
 const db={transaction(names,mode){
  const stores=Array.isArray(names)?names:[names];const staged=Object.fromEntries(stores.map(n=>[n,new Map(disk[n])]));
  let cancelled=false;
  const tx={error:null,abort(){cancelled=true;queueMicrotask(()=>tx.onabort?.());},objectStore(n){return {
   put(value,key){if(n==='threads'&&(typeof value.id!=='string'||!value.id))throw Error('DataError');staged[n].set(key??value.id,copy(value));return {};},
   clear(){staged[n].clear();return {};}
  };}};
  setTimeout(()=>{if(cancelled)return;if(abort){tx.error=Object.assign(Error('disk full'),{name:'QuotaExceededError'});tx.onabort?.();return;}
   for(const name of stores)disk[name]=staged[name];tx.oncomplete?.();},0);
  return tx;
 }};
 const ctx={console:{log(){},warn(){},error(){}},setTimeout,clearTimeout,structuredClone,
  STORAGE_KEY:'state',LEGACY_STORAGE_KEY:'legacy',
  localStorage:{getItem:k=>local.get(k)??null,setItem:(k,v)=>local.set(k,v),removeItem:k=>local.delete(k)},
  state:{threads:[],settings:{},characterCards:[]},
 };
 vm.createContext(ctx);vm.runInContext(source,ctx);
 ctx.idbOpen=async()=>db;
 ctx.idbGet=async(store,key)=>{if(readFailure)throw Error('read failed');return copy(disk[store].get(key));};
 ctx.idbGetAll=async store=>Array.from(disk[store].values(),copy);
 ctx.applyLoadedState=blob=>{if(!Array.isArray(blob.threads))return false;Object.assign(ctx.state,copy(blob));return true;};
 ctx.buildStateBlob=()=>copy(ctx.state);
 ctx.persistAdoptedThreadOrder=async()=>{};
 return {ctx,disk,local,get:s=>vm.runInContext(s,ctx)};
}
for(const legacyKey of ['state','legacy']) {
 const h=harness({legacy:old,legacyKey});await h.ctx.loadState();
 assert.equal(h.ctx.state.threads[0].messages[0].content,'old chat');
 assert.equal(h.disk.meta.get('app').characterCards[0].name,'Goblin');
 assert.equal(h.disk.meta.get('app')._storageLayout,1);
 assert.equal(h.disk.threads.size,1);assert.ok(h.local.has(legacyKey),'original retained');
}
{
 const h=harness({legacy:old,abort:true});await h.ctx.loadState();
 assert.equal(h.disk.meta.has('app'),false,'aborted migration must not publish meta');
 assert.equal(h.disk.threads.size,0);
 assert.equal(h.ctx.state.threads.length,1,'history remains readable in memory');
 assert.equal(h.get('STORAGE_BACKEND'),'localstorage');
 assert.equal(h.local.get('state'),JSON.stringify(old));
 const retry=harness({legacy:JSON.parse(h.local.get('state'))});await retry.ctx.loadState();
 assert.equal(retry.disk.threads.size,1,'next boot retries migration');
}
{
 const h=harness({meta:old,threads:[{id:'new',messages:[]}]});await h.ctx.loadState();
 assert.deepEqual(Array.from(h.ctx.state.threads,t=>t.id).sort(),['a','new']);
 assert.equal(h.disk.meta.get('app').threads,undefined,'merged document was split');
 assert.deepEqual(h.disk.meta.get('pre-split'),old,'original merged snapshot retained');
}
{
 const h=harness({threads:old.threads});await h.ctx.loadState();
 assert.equal(h.ctx.state.threads[0].id,'a','orphan records recovered without meta');
 assert.equal(h.disk.meta.get('app')._storageLayout,1);
}
{
 const h=harness({meta:{settings:{theme:'new'}},legacy:old});await h.ctx.loadState();
 assert.equal(h.ctx.state.threads.length,0,'do not guess whether empty means deletion');
 assert.equal(h.get('legacyBrowserRecovery.blob.threads.length'),1,'offer older copy in UI');
 assert.equal(h.disk.meta.get('pre-split').threads.length,1,'recovery remains available on subsequent boots');
}
{
 const h=harness({meta:{_storageLayout:1,settings:{}},legacy:old});await h.ctx.loadState();
 assert.equal(h.ctx.state.threads.length,0);
 assert.equal(h.get('legacyBrowserRecovery'),null,'intentional empty database does not resurrect stale legacy data');
}
{
 const a=harness({meta:{settings:{theme:'current'}},threads:old.threads,abort:true});await a.ctx.loadState();
 await assert.rejects(a.ctx.idbWriteSnapshot({threads:[{id:'replacement',messages:[]}],settings:{}},true));
 assert.equal(a.disk.threads.get('a').messages[0].content,'old chat','failed restore keeps old conversations');
 assert.equal(a.disk.meta.get('app').settings.theme,'current');
}
{
 const h=harness({legacy:'not json'});await h.ctx.loadState();
 assert.equal(h.get('storageLoadFailed'),true,'unreadable source pauses automatic saves');
 assert.equal(h.disk.meta.size,0);assert.equal(h.local.get('state'),'not json');
}
{
 const h=harness({meta:{settings:{}},threads:old.threads,readFailure:true});await h.ctx.loadState();
 assert.equal(h.get('storageLoadFailed'),true);assert.equal(h.disk.threads.size,1);
}
console.log('PASS: atomic migration/restore, retry, merged and orphan records, retained originals, safe recovery and read failures');
