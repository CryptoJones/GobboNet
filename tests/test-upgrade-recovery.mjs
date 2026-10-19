import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
const src=fs.readFileSync(new URL('../js/21-data.js',import.meta.url),'utf8');
const code=src.slice(src.indexOf('/* UPGRADE RECOVERY'));
function harness({agree=true,backupFailure=false,restoreFailure=false}={}) {
 const writes=[],storage=new Map();const elements={};
 const el=id=>elements[id]??=({textContent:'',style:{},value:'legacy-1234567890123456',appendChild(){}});
 const ctx={console,JSON,Date,document:{getElementById:el,createElement:()=>({})},
  state:{threads:[{id:'current',messages:[]}],settings:{}},STORAGE_BACKEND:'idb',storageLoadFailed:false,
  legacyBrowserRecovery:{raw:JSON.stringify({threads:[{id:'old',messages:[]}],settings:{},extensions:{enabled:true}})},
  storageRecoveryMessage:'',isGenerating:false,confirm:()=>agree,alert(){},
  syncTarget:{mode:'shared',profile:''},stateSync:{pushTimer:0,pendingPush:false},_idbFullSaveTimer:null,_idbPendingBlob:null,
  buildStateBlob:()=>structuredClone(ctx.state),
  idbGet:async(_,key)=>storage.get(key),
  idbPut:async(_,value,key)=>{if(backupFailure)throw Error('disk full');storage.set(key,structuredClone(value));},
  idbWriteSnapshot:async blob=>writes.push(structuredClone(blob)),
  validateServerSnapshot:blob=>assert.ok(Array.isArray(blob.threads)),
  neutralizeUntrustedCode:blob=>{blob.extensions.enabled=false;},
  applyLoadedState:blob=>{Object.assign(ctx.state,blob);return true;},
  persistSyncTarget(){},invalidateLedger(){},render(){},applyAvatarScale(){},applyActiveCardBackground(){},
  updateSyncIndicator(){},renderSyncTargetPanel(){},clearTimeout(){},
  restoreFromServer:async()=>!restoreFailure,downloadJSON(blob){writes.push(blob);},
 };
 vm.createContext(ctx);vm.runInContext(code,ctx);
 return {ctx,writes,storage,get:expr=>vm.runInContext(expr,ctx)};
}
{
 const h=harness({agree:false});await h.ctx.recoverUpgradeHistory('browser');
 assert.equal(h.storage.size,0);assert.equal(h.ctx.state.threads[0].id,'current');
}
{
 const h=harness({backupFailure:true});await h.ctx.recoverUpgradeHistory('browser');
 assert.equal(h.ctx.state.threads[0].id,'current');assert.equal(h.ctx.syncTarget.mode,'shared');
 assert.match(h.get('upgradeRecoveryStatus'),/could not finish/);
}
{
 const h=harness();await h.ctx.recoverUpgradeHistory('browser');
 assert.equal(h.storage.get('pre-recovery').threads[0].id,'current');
 assert.equal(h.ctx.state.threads[0].id,'old');assert.equal(h.ctx.state.extensions.enabled,false);
 assert.equal(h.ctx.syncTarget.mode,'off','browser recovery cannot overwrite shared server history');
 await h.ctx.downloadBeforeUpgradeRecovery();assert.equal(h.writes.at(-1).threads[0].id,'current');
}
{
 const h=harness({restoreFailure:true});vm.runInContext("upgradeServerProfiles = ['legacy-1234567890123456']",h.ctx);
 await h.ctx.recoverUpgradeHistory('server');assert.equal(h.ctx.syncTarget.mode,'shared');
 assert.equal(h.ctx.state.threads[0].id,'current');assert.equal(h.storage.get('pre-recovery').threads[0].id,'current');
}
console.log('PASS: recovery cancellation, safety copy, failed persistence, local-only restore, code quarantine and target rollback');
