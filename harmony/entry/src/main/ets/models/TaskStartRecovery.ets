import type { AssistantFeedModel } from './AssistantFeed';
import type { ClientModel } from './ClientModel';
import type { HubRepository } from './HubRepository';
import { ResourceRow, Wire } from './Wire';

type Fields = Record<string,Object>;

function matchingTask(plan:Fields, id:string):Fields | undefined {
  return Wire.rows(plan,'tasks').find((task:Fields) => Wire.text(task,'todo_id')===id);
}

function ownedManager(data:Fields, owner:string):boolean {
  return !Wire.has(data,'manager') || typeof data['manager']==='string' && data['manager']===owner;
}

function cannotWrite(row:Fields):boolean {
  return Wire.bool(row,'read_only',Wire.bool(row,'readonly')) || Wire.has(row,'can_write') && !Wire.bool(row,'can_write');
}

/** A review ticket is scoped to the failed attempt. Read markers and rendered timestamps are independent. */
function attemptSignature(todo:Fields, plan:Fields, task:Fields):string {
  return JSON.stringify([Wire.text(todo,'id'), Wire.text(todo,'origin'), Wire.text(todo,'plan_id'), Wire.text(todo,'title'),
    Wire.text(todo,'start_error'), Wire.text(todo,'done_at'), Wire.text(todo,'reopened_at'), Wire.text(plan,'id'), Wire.text(plan,'conversation'),
    Wire.text(task,'machine'), Wire.text(task,'cwd'), Wire.text(task,'launcher'), Wire.text(task,'error'), Wire.text(task,'error_code'),
    Wire.object(task,'error_args'), Wire.strings(task,'retry_with'),
    Wire.rows(todo,'agents').map((a:Fields)=>[Wire.text(a,'machine_id'),Wire.text(a,'session_id'),Wire.text(a,'linked_at'),Wire.text(a,'released_at'),Wire.text(a,'dismissed_at')])]);
}

@ObservedV2
export class TaskStartRecovery {
  @Trace id:string='';
  @Trace title:string='';
  @Trace failure:string='';
  @Trace state:string='';
  @Trace error:string='';
  @Trace busy:boolean=false;
  @Trace readOnly:boolean=false;
  @Trace confirmRemoval:boolean=false;
  @Trace alternative:string='';
  @Trace reasonCode:string='';
  @Trace reasonLauncher:string='';
  private client:ClientModel | undefined;
  private feed:AssistantFeedModel | undefined;
  private repository:HubRepository | undefined;
  private generation:number=0;
  private clientGeneration:number=0;
  private group:string='';
  private conversation:string='';
  private manager:string='';
  private scope:string='';
  private planID:string='';
  private version:string='';
  private active:()=>boolean=():boolean=>false;
  private uncertain:boolean=false;

  close():void {
    this.generation++; this.id=''; this.title=''; this.failure=''; this.state=''; this.error=''; this.busy=false;
    this.alternative=''; this.reasonCode=''; this.reasonLauncher=''; this.version=''; this.uncertain=false;this.readOnly=false;this.confirmRemoval=false;
    this.client=undefined; this.feed=undefined; this.repository=undefined; this.active=():boolean=>false;
  }

  open(client:ClientModel,feed:AssistantFeedModel,row:ResourceRow,active:()=>boolean):void {
    this.close();
    const item=feed.todo(row.id), origin=Wire.text(item,'origin');
    if (!row.id || !origin || origin!==feed.conversationId || row.group!==client.profile.id || row.manager!==client.profile.manager ||
      feed.owner!==client.profile.manager || !ownedManager(item,feed.owner)) return;
    this.client=client; this.feed=feed; this.repository=client.repo(); this.clientGeneration=client.connectionGeneration(); this.scope=client.scope();
    this.group=row.group; this.conversation=origin; this.manager=row.manager; this.id=row.id; this.title=Wire.text(item,'title'); this.active=active;
    let planID=Wire.text(item,'plan_id'), plan=Wire.record(feed.plans[planID]);
    if (!matchingTask(plan,this.id)) {
      const matches=Object.keys(feed.plans).filter((key:string)=>{
        const p=Wire.record(feed.plans[key]); return Wire.text(p,'conversation')===origin && Wire.text(p,'state')!=='dismissed' && !!matchingTask(p,this.id);
      });
      if(matches.length===1) { planID=matches[0]; plan=Wire.record(feed.plans[planID]); }
    }
    this.planID=planID; this.readOnly=cannotWrite(item);
    const task=matchingTask(plan,this.id) || {};
    this.failure=Wire.text(item,'start_error') || Wire.text(task,'error');
    this.version=this.failure && Wire.text(plan,'id')===planID && Wire.text(plan,'conversation')===origin ? attemptSignature(item,plan,task) : '';
    this.state='loading'; this.refresh();
  }

  private current(run:number):boolean {
    const c=this.client, f=this.feed;
    return this.generation===run && !!c && !!f && c.owns(this.clientGeneration,this.scope) && c.repo()===this.repository &&
      c.connected && c.profile.id===this.group && c.profile.manager===this.manager && f.owner===this.manager && f.conversationId===this.conversation && this.active();
  }
  private path(action:string):string {
    const tail=action==='plan' ? '/api/v1/task-plans/'+encodeURIComponent(this.planID) : '/api/v1/todos/'+encodeURIComponent(this.id)+(action==='retry'?'/retry':'');
    return this.manager ? tail+'?manager='+encodeURIComponent(this.manager) : tail;
  }
  private belongs(todo:Fields):boolean {
    return Wire.text(todo,'id')===this.id && Wire.text(todo,'origin')===this.conversation && ownedManager(todo,this.manager);
  }
  private async readAttempt(run:number):Promise<string> {
    const repository=this.repository!;
    if (!this.current(run)) return '';
    const result=await repository.call(this.path('todo'));
    if (!this.current(run)) return '';
    const todo=Wire.object(result,'todo');
    if (!this.belongs(todo)) return '';
    this.readOnly=cannotWrite(todo); this.confirmRemoval=Wire.rows(todo,'agents').length>0;
    if (!this.planID || (Wire.text(todo,'plan_id') && Wire.text(todo,'plan_id')!==this.planID)) return '';
    const response=await repository.call(this.path('plan'));
    if (!this.current(run)) return '';
    const plan=Wire.object(response,'plan'), task=matchingTask(plan,this.id);
    if (!task || Wire.text(plan,'id')!==this.planID || Wire.text(plan,'conversation')!==this.conversation ||
      !ownedManager(plan,this.manager) || Wire.text(plan,'state')==='dismissed') return '';
    const error=Wire.text(todo,'start_error') || (!Wire.rows(todo,'agents').length ? Wire.text(task,'error') : '');
    if (!error) return '';
    this.title=Wire.text(todo,'title'); this.failure=error;
    this.reasonCode=Wire.text(task,'error_code');
    this.reasonLauncher=Wire.text(Wire.object(task,'error_args'),'where')==='terminal' ? Wire.text(Wire.object(task,'error_args'),'launcher') : '';
    this.alternative=this.client!.features.includes('task_retry_launcher') ? Wire.strings(task,'retry_with')[0] || '' : '';
    return attemptSignature(todo,plan,task);
  }
  private fail(error:Error):void {
    const data=error as Object as Fields, status=Wire.number(data,'status');
    this.error=data['resultUnknown']===true || data['resultPending']===true ? 'result_unknown' :
      status===403 || status===401 ? 'permission_denied' : status===409 || status===410 ? 'request_updated' : status===404 ? 'task_recovery_gone' : 'failed';
    if (this.error==='result_unknown') this.uncertain=true;
    this.state=this.uncertain ? 'unknown' : this.error==='task_recovery_gone' ? 'gone' : 'stale';
  }
  async refresh():Promise<void> {
    if (this.busy || !this.id) return;
    const run=this.generation; this.busy=true;
    try {
      const result=await this.readAttempt(run);
      if (!this.current(run)) return;
      if (!result || (this.version && result!==this.version)) { this.state='stale'; this.error='request_updated'; return; }
      this.version=result; this.state=this.uncertain ? 'unknown':'ready'; this.error=this.uncertain?'result_unknown':'';
    } catch(e) { if (this.current(run)) this.fail(e as Error); }
    finally { if(this.generation===run) this.busy=false; }
  }
  statusMessage():string {
    if (this.error) return this.error;
    if (this.state!=='ready' || !this.current(this.generation)) return '';
    if (this.readOnly || !this.client!.canApprove()) return 'permission_denied';
    if (this.client!.operationBlocked(this.path('retry')) || this.client!.operationBlocked(this.path('remove'))) return 'result_unknown';
    return '';
  }
  canAct(action:string):boolean {
    return (action==='retry' || action==='remove') && this.current(this.generation) && this.state==='ready' && !this.readOnly && !this.uncertain && !this.busy &&
      !!this.version && !!this.client?.canApprove() && !this.client.operationBlocked(this.path('retry')) && !this.client.operationBlocked(this.path('remove'));
  }
  private localUnchanged():boolean {
    const f=this.feed;
    if (!f || !f.todosReady) return false;
    const todo=f.todo(this.id), plan=Wire.record(f.plans[this.planID]), task=matchingTask(plan,this.id);
    return this.belongs(todo) && !cannotWrite(todo) && !!task && attemptSignature(todo,plan,task)===this.version;
  }
  async act(action:string, launcher:string=''):Promise<boolean> {
    if (!this.canAct(action) || launcher && (action!=='retry' || launcher!==this.alternative || !this.client?.features.includes('task_retry_launcher'))) return false;
    const run=this.generation, version=this.version, owner=this.manager, id=this.id, origin=this.conversation, repo=this.repository!;
    this.busy=true; this.error='';
    try {
      const live=await this.readAttempt(run);
      if (!this.current(run)) return false;
      if (version!==live || !this.localUnchanged()) { this.state='stale'; this.error='request_updated'; return false; }
      if (this.readOnly || !this.client!.canApprove() || this.client!.operationBlocked(this.path('retry')) || this.client!.operationBlocked(this.path('remove'))) return false;
      const admit=():boolean=>this.current(run) && this.client!.canApprove() && !this.readOnly && this.version===version && this.localUnchanged() &&
        (!launcher || this.client!.features.includes('task_retry_launcher') && this.alternative===launcher);
      const acknowledged=(data:Fields):boolean=>{
        if(action==='remove') return Wire.bool(data,'ok');
        const t=Wire.object(data,'todo'); return Wire.text(t,'id')===id && Wire.text(t,'origin')===origin && ownedManager(t,owner);
      };
      const reply=await repo.call(this.path(action),action==='remove'?'DELETE':'POST',launcher ? Wire.payload(['launcher'],[launcher]) : {},'',admit,false,acknowledged,!launcher);
      if (!this.current(run)) return false;
      if (!acknowledged(reply)) { this.uncertain=true;this.state='unknown';this.error='result_unknown';return false; }
      this.state=action==='remove' ? 'gone' : 'starting'; this.failure=''; this.alternative=''; this.version='';
      await this.feed!.refresh();
      return this.current(run);
    } catch(e) { if(this.current(run)) this.fail(e as Error); return false; }
    finally { if(this.generation===run) this.busy=false; }
  }
}
