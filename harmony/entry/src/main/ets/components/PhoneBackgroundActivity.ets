import type { BackgroundRound } from '../core/BackgroundActivity';
import { activityFeedRound, splitBackgroundRounds } from '../core/BackgroundActivity';
import type { TranscriptItem } from '../core/Domain';
import { Colors } from '../design/Colors';
import { tr, trError } from '../design/Strings';
import { Symbols } from '../design/Symbols';
import { Tokens } from '../design/Tokens';
import { conversationFeedPath } from '../models/AssistantFeed';
import { ClientModel } from '../models/ClientModel';
import { TranscriptModel } from '../models/TranscriptModel';
import { ResourceRow, Wire } from '../models/Wire';

@ComponentV2
export struct PhoneBackgroundActivity {
  @Require @Param model: ClientModel;
  @Require @Param row: ResourceRow;
  @Event onClose: () => void = () => {};

  @Local rounds: BackgroundRound[] = [];
  @Local feed: Record<string, Object>[] = [];
  @Local opened: string[] = [];
  @Local more: boolean = false;
  @Local cursor: string = '';
  @Local busy: boolean = true;
  @Local error: string = '';

  private transcript: TranscriptModel | undefined = undefined;
  private ticket: number = 0;
  private generation: number = -1;
  private scope: string = '';
  private target: string = '';

  private current(ticket: number): boolean {
    const main = this.model.mainActivityRow();
    return this.ticket === ticket && this.model.owns(this.generation, this.scope) &&
      main !== undefined && this.model.target(main) === this.target;
  }

  @Monitor('model.profile', 'model.connected', 'model.mainId', 'model.mainMachineId')
  accountChanged(): void { if (!this.current(this.ticket)) this.onClose(); }

  aboutToAppear(): void { this.load(); }
  aboutToDisappear(): void { this.ticket++; this.transcript?.close(); this.transcript = undefined; }

  private async load(): Promise<void> {
    const ticket = ++this.ticket;
    this.transcript?.close(); this.transcript = undefined;
    this.generation = this.model.connectionGeneration(); this.scope = this.model.scope(); this.target = this.model.target(this.row);
    this.busy = true; this.error = ''; this.more = false; this.cursor = ''; this.feed = []; this.rounds = []; this.opened = [];
    try {
      this.transcript = this.model.readActivityTranscript(this.row, (items: readonly TranscriptItem[], more: boolean): void => {
        if (!this.current(ticket)) return;
        const actual = this.model.mainActivityRow();
        const running = actual?.state === 'working' || actual?.state === 'running' || Wire.bool(actual?.data() || {}, 'working');
        this.rounds = Array.from(splitBackgroundRounds(items, running).rounds);
        this.more = more;
      }, (error: Error): void => { if (this.current(ticket)) this.error = error.message; });
      await Promise.all([this.transcript?.open(), this.readFeed(ticket, '')]);
    } catch (error) {
      if (this.current(ticket)) this.error = (error as Error).message || 'failed';
    } finally { if (this.current(ticket)) this.busy = false; }
  }

  private async readFeed(ticket: number, cursor: string): Promise<void> {
    if (!this.model.connected || this.model.demo || !this.current(ticket)) return;
    const path = conversationFeedPath(this.row.id, this.row.manager || this.model.profile.manager, cursor);
    const page = await this.model.repo().call(path);
    if (!this.current(ticket)) return;
    const items = new Map<string, Record<string, Object>>();
    for (const item of this.feed.concat(Wire.rows(page, 'items'))) {
      const id = Wire.text(item, 'id');
      if (id) items.set(id, item);
    }
    this.feed = Array.from(items.values());
    const next = Wire.text(page, 'next_cursor');
    this.cursor = next !== cursor ? next : '';
  }

  private async older(): Promise<void> {
    if (this.busy) return;
    this.busy = true; this.error = '';
    const ticket = this.ticket;
    try {
      if (this.more) await this.transcript?.older();
      if (this.current(ticket) && this.cursor) await this.readFeed(ticket, this.cursor);
    } catch (error) { if (this.current(ticket)) this.error = (error as Error).message || 'failed'; }
    finally { if (this.current(ticket)) this.busy = false; }
  }

  private toggle(key: string): void {
    this.opened = this.opened.includes(key) ? this.opened.filter((id: string) => id !== key) : this.opened.concat([key]);
  }

  private roundFeed(round: BackgroundRound): Record<string, Object>[] {
    return this.feed.filter((item: Record<string, Object>): boolean =>
      activityFeedRound([round], Wire.text(item, 'kind'), Date.parse(Wire.text(item, 'at'))) === round.id)
      .sort((left: Record<string, Object>, right: Record<string, Object>): number =>
        Date.parse(Wire.text(left, 'at')) - Date.parse(Wire.text(right, 'at')));
  }

  private transcriptLabel(item: TranscriptItem): string {
    if (item.kind !== 'tool_use') return item.text || '';
    const input = Wire.object(Wire.record(item as Object), 'tool_input');
    return Wire.text(input, 'description') || Wire.text(input, 'text') || String(item.tool_name || '');
  }

  private changeFold(item: TranscriptItem): void {
    const ticket = this.ticket;
    this.transcript?.unfold(item.id).catch((error: Error): void => { if (this.current(ticket)) this.error = error.message; });
  }

  @Builder disclosure(label: string, key: string, identity: string) {
    Button({ type: ButtonType.Normal }) {
      Row({ space: 8 }) {
        SymbolGlyph(this.opened.includes(key) ? Symbols.down : Symbols.expand).fontSize(13).fontColor([Colors.secondary])
        Text(label).fontSize(15).fontColor(Colors.secondary).layoutWeight(1).textAlign(TextAlign.Start)
      }.width('100%').constraintSize({ minHeight: 44 })
    }.width('100%').padding({ left: 8, right: 8 }).backgroundColor(Color.Transparent)
      .accessibilityText(label).accessibilitySelected(this.opened.includes(key)).onClick(() => this.toggle(key)).id(identity)
  }

  @Builder feedRow(item: Record<string, Object>) {
    Column({ space: 6 }) {
      if (Wire.text(item, 'kind') === 'answered') {
        this.disclosure(tr('activity_answered'), Wire.text(item, 'id'), 'answered-line')
        if (this.opened.includes(Wire.text(item, 'id'))) {
          Column({ space: 6 }) {
            Text(Wire.text(Wire.object(item, 'body'), 'question')).fontColor(Colors.ink).fontSize(16)
            Text(Wire.text(Wire.object(item, 'body'), 'answer')).fontColor(Colors.secondary).fontSize(15)
          }.width('100%').alignItems(HorizontalAlign.Start).id('answered-detail')
        }
      } else {
        Text(Wire.text(Wire.object(item, 'body'), 'title') || Wire.text(Wire.object(item, 'body'), 'text') ||
          this.model.todos.find((todo: ResourceRow) => todo.id === Wire.text(item, 'ref'))?.title || tr('activity_task'))
          .fontColor(Colors.ink).fontSize(16).width('100%').textAlign(TextAlign.Start)
          .id('dispatch.' + Wire.text(item, 'kind') + '.' + Wire.text(item, 'ref'))
      }
    }.width('100%').alignItems(HorizontalAlign.Start).padding(12).backgroundColor(Colors.surface).borderRadius(14)
  }

  @Builder activity(round: BackgroundRound) {
    Column({ space: 6 }) {
      this.disclosure(tr('activity_record'), round.id, 'supervision-record')
      if (this.opened.includes(round.id)) {
        Text(round.items[0].text || '').width('100%').fontSize(16).fontColor(Colors.ink).id('supervision-input')
        this.disclosure(tr('work_summary'), 'work:' + round.id, 'work-head')
        if (this.opened.includes('work:' + round.id)) {
          ForEach(round.items.slice(1), (item: TranscriptItem) => {
            if (item.kind === 'fold') {
              Button(tr('activity_expand')).onClick(() => this.changeFold(item)).id('activity.fold.' + item.id)
            } else if (this.transcriptLabel(item)) {
              Text(this.transcriptLabel(item)).width('100%').fontSize(15).fontColor(Colors.ink).textAlign(TextAlign.Start)
            }
          }, (item: TranscriptItem) => item.id)
        }
      }
      ForEach(this.roundFeed(round), (item: Record<string, Object>) => { this.feedRow(item) }, (item: Record<string, Object>) => Wire.text(item, 'id'))
    }.width('100%').alignItems(HorizontalAlign.Start)
  }

  build() {
    Column() {
      Stack({ alignContent: Alignment.Center }) {
        Text(tr('background_activity')).fontSize(17).fontWeight(FontWeight.Medium).fontColor(Colors.ink)
        Row() {
          Button({ type: ButtonType.Circle }) {
            SymbolGlyph($r('sys.symbol.xmark')).fontSize(20).fontColor([Colors.ink])
          }.width(44).height(44).backgroundColor(Colors.surface).accessibilityText(tr('close')).onClick(() => this.onClose()).id('activity.close')
          Blank()
        }.width('100%')
      }.width('100%').height(56).padding({ left: 16, right: 16 })
      if (this.error) {
        Text(trError(this.error)).fontColor(Colors.secondary).padding(12)
        Button(tr('retry')).onClick(() => this.load()).id('activity.retry')
      }
      if (!this.rounds.length && this.busy) {
        Blank()
        LoadingProgress().width(32).height(32)
        Blank()
      } else if (!this.rounds.length && !this.more && !this.cursor && !this.error) {
        Blank()
        SymbolGlyph($r('sys.symbol.clock')).fontSize(38).fontColor([Colors.secondary])
        Text(tr('activity_empty')).fontSize(19).fontWeight(FontWeight.Medium).fontColor(Colors.ink).margin({ top: 10 })
        Text(tr('activity_empty_detail')).fontSize(15).fontColor(Colors.secondary).margin({ top: 6 }).id('background-activity-empty')
        Blank()
      } else {
        List({ space: 10 }) {
          if (this.more || this.cursor) ListItem() { Button(tr('load_more')).enabled(!this.busy).onClick(() => this.older()).id('activity.older') }
          ForEach(this.rounds, (round: BackgroundRound) => { ListItem() { this.activity(round) } }, (round: BackgroundRound) => round.id)
        }.width('100%').padding({ left: 16, right: 16, bottom: 12 }).layoutWeight(1)
      }
    }.width('100%').height('100%').backgroundColor(Colors.canvas).id('background-activity-page')
  }
}
