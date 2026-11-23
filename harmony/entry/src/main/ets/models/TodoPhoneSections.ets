import type { TodoWorkspaceItem } from './TodoWorkspace';
import { ResourceRow, Wire } from './Wire';

/** The iPhone's ordered list sections. The task detail uses its independent, six-state status. */
export const TODO_PHONE_SECTIONS:string[]=['needs_you','working','pr','other','just_done','shelved','done'];
export interface TodoPhonePlacement { section:string; last:number; superseded:boolean; awaitingReview:boolean; }

export class TodoPhoneRowContext {
  lead: ResourceRow | undefined = undefined;
  activeAgents: number = 0;
  urgentState: string = '';
  summary: string = '';
  summaryKind: string = 'empty';
}

function time(value:string):number {
  const date=Date.parse(value); return Number.isFinite(date) ? date : Number.NEGATIVE_INFINITY;
}

export function phoneTodoTitle(section:string):string {
  switch (section) {
    case 'needs_you': return 'phone_todo_section_needs';
    case 'working': return 'phone_todo_section_working';
    case 'pr': return 'phone_todo_section_pr';
    case 'other': return 'phone_todo_section_open';
    case 'just_done': return 'phone_todo_section_recent';
    case 'shelved': return 'phone_todo_section_shelved';
    default: return 'phone_todo_section_done';
  }
}

export class TodoPhoneSections {
  private sessions:Map<string,ResourceRow>=new Map<string,ResourceRow>();
  private now:number;
  constructor(rows:ResourceRow[],now:number) {
    this.now=now;
    rows.forEach((row:ResourceRow)=>{
      if (row.kind==='session') this.sessions.set(JSON.stringify([row.manager,row.machine,row.session || row.id]),row);
    });
  }

  private session(manager: string, machine: string, id: string): ResourceRow | undefined {
    return this.sessions.get(JSON.stringify([manager, machine, id])) || this.sessions.get(JSON.stringify(['', machine, id]));
  }

  rowContext(todo: TodoWorkspaceItem, placement?: TodoPhonePlacement): TodoPhoneRowContext {
    const row = new TodoPhoneRowContext();
    let best = -1, leadMoved = Number.NEGATIVE_INFINITY;
    for (const agent of todo.activeAgents()) {
      row.activeAgents++;
      const candidate = this.session(todo.manager, agent.machineId, agent.sessionId);
      const data = candidate?.data();
      if (candidate && data) {
        const moved = Math.max(time(Wire.text(data, 'updated_at')), time(Wire.text(data, 'state_since')));
        const working = candidate.state === 'working', previousWorking = row.lead?.state === 'working';
        if (!row.lead || (working !== previousWorking ? working : moved > leadMoved)) {
          row.lead = candidate;
          leadMoved = moved;
        }
      }
      const state = agent.startError ? 'offline' : candidate?.state || 'offline';
      const urgency = agent.startError ? 1 : state === 'needs_you' ? 4 : state === 'unread' ? 3 : state === 'working' ? 2 : 0;
      if (urgency <= best) continue;
      best = urgency;
      row.urgentState = state;
      row.summary = agent.startError || (candidate && data ? Wire.text(data, 'last_text') || candidate.title : agent.title);
      row.summaryKind = agent.startError ? 'failure' : 'session';
    }
    const raw = todo.data(), closedBy = Wire.text(raw, 'done_by');
    const cancelled = !!Wire.text(raw, 'cancelled_at') || closedBy === 'cancelled' ||
      closedBy === 'review' && Wire.text(todo.supervision, 'state') === 'cancelled';
    const reason = cancelled ? (Wire.text(raw, 'cancelled_reason') || todo.doneReason).trim() :
      todo.isDone() && ['auto', 'review'].includes(closedBy) ? todo.doneReason.trim() : '';
    if (todo.startError && (!todo.isDone() || !placement || placement.section === 'needs_you')) {
      row.summaryKind = 'failure';
      row.summary = todo.startError;
    } else if (reason) {
      row.summaryKind = 'closed';
      row.summary = reason;
    } else if (!row.summary) {
      row.summaryKind = todo.notes ? 'notes' : 'empty';
      row.summary = todo.notes;
    }
    return row;
  }

  place(todo:TodoWorkspaceItem,superseded:boolean=false):TodoPhonePlacement {
    const sup=todo.supervision,reviews=Wire.rows(sup,'reviews'),review:Record<string,Object>=reviews[reviews.length-1] || Wire.payload([],[]);
    const reviewAt=time(Wire.text(review,'at')),delivery=Wire.text(todo.delivery,'state'),deliveredAt=time(Wire.text(todo.delivery,'at'));
    const askTime=time(Wire.text(todo.ask,'asked_at')),doneAt=time(todo.doneAt),seenAt=time(Wire.text(todo.data(),'seen_at'));
    let last=Math.max(time(todo.updatedAt),time(todo.createdAt),time(todo.reopenedAt),doneAt,time(Wire.text(sup,'updated_at')),
      time(Wire.text(Wire.object(sup,'progress'),'at')),reviewAt,deliveredAt,askTime);
    let asks=false,interrupted=false,limit=false,unread=false,working=false;
    for (const a of todo.activeAgents()) {
      const row=this.session(todo.manager,a.machineId,a.sessionId);
      if (!row) continue;
      const data=row.data(),state=row.state || Wire.text(data,'state');
      last=Math.max(last,time(Wire.text(data,'updated_at')),time(Wire.text(data,'state_since')));
      if (a.startError) continue;
      const stopped=state==='idle' || state==='unread';
      if (state==='needs_you') asks=true;
      else if (stopped && Wire.text(data,'turn_error')) {
        if (Wire.text(data,'turn_error_kind')==='usage_limit') limit=true;
        else interrupted=true;
      } else if (state==='unread') unread=true;
      else if (state==='working' || Wire.bool(data,'starting')) working=true;
    }
    const current=(when:number):boolean=>this.now-when<86400000 && !(Number.isFinite(seenAt) && seenAt>=when);
    const concern=todo.kind==='concern' && !todo.isDone(),reviewState=Wire.text(sup,'state');
    const openAsk=!!Wire.text(todo.ask,'verdict') && Number.isFinite(askTime),outcome=Wire.text(todo.ask,'outcome');
    const need=reviewState==='needs_user' || reviewState==='user_check' || openAsk && (outcome==='decide' || outcome==='cannot') ||
      Wire.text(todo.delivery,'unblock')==='user' && ['pr','merged','off'].includes(delivery);
    const finished=!todo.isDone() && !limit && unread || reviewState==='pending_review' || openAsk && (!outcome || outcome==='done');
    let section='other';
    if (todo.startError) section=superseded ? 'done' :
      this.now-last<172800000 ? 'needs_you' : concern ? 'other' : 'done';
    else if (!todo.isDone() && (asks || interrupted || need)) section='needs_you';
    else if (!todo.isDone() && !limit && !unread && working) section='working';
    else if (todo.isDone()) section=todo.kind==='concern' && current(doneAt) ? 'just_done' : 'done';
    else if (!concern && reviewState==='accepted' && Wire.text(review,'verdict')==='accepted') section='done';
    else if (!concern && delivery==='live') section='just_done';
    else if (finished && reviewState!=='parked' && (!(Number.isFinite(seenAt) && seenAt>=last) ||
      reviewState==='pending_review' || !!Wire.text(sup,'supervisor'))) section='just_done';
    else if (delivery==='pr' || !!todo.prUrl && (!delivery || delivery==='none')) section='pr';
    else if (finished && !concern) section='done';
    else if (!concern && (todo.agents.length>0 || !!todo.origin) && !(todo.dueBy>=this.now) && this.now-last>=172800000) section='shelved';
    return {section:section,last:last,superseded:superseded,awaitingReview:section==='just_done' && !todo.isDone()};
  }
}
