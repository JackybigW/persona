import { SymbolGlyphModifier } from '@kit.ArkUI';
import { i18n } from '@kit.LocalizationKit';
import { relativeActivityDate } from '../core/PhoneDate';
import { Colors } from '../design/Colors';
import { PhoneStyle } from '../design/PhoneStyle';
import { tr } from '../design/Strings';
import { Tokens } from '../design/Tokens';
import { ClientModel } from '../models/ClientModel';
import { HealthWorkspace } from '../models/HealthWorkspace';
import { BoundRow } from '../models/RowBinding';
import { ResourceRow, Wire } from '../models/Wire';
import { NativeEnvironment, nativeEnvironment } from '../platform/NativeEnvironment';

@ComponentV2
export struct HealthConversationHistory {
  @Require @Param model: ClientModel;
  @Require @Param workspace: HealthWorkspace;
  @Require @Param rows: BoundRow<ResourceRow>[];
  @Event onOpen: (row: ResourceRow) => void = (_row: ResourceRow) => {};
  @Local private archivingBinding: string = '';
  @Local private env: NativeEnvironment = nativeEnvironment;
  @Local private now: number = Date.now();
  private mounted: boolean = false;
  private timer: number = -1;

  aboutToAppear(): void { this.mounted=true; this.updateClock(); }
  aboutToDisappear(): void { this.mounted=false; this.updateClock(); }

  @Monitor('env.foreground')
  private updateClock(): void {
    if (this.timer !== -1) { clearInterval(this.timer); this.timer=-1; }
    if (!this.mounted || !this.env.foreground) return;
    this.now=Date.now();
    this.timer=setInterval(() => { this.now=Date.now(); }, 60000);
  }

  private scaledLength(regular: number, largest: number): number {
    return PhoneStyle.formSize(regular, largest, this.env.fontScale);
  }
  private title(row: ResourceRow): string {
    return row.title && row.title !== 'Claude Code' ? row.title : tr('phone_new_chat_title');
  }
  private preview(row: ResourceRow): string {
    return Wire.text(row.data(), 'last_text', row.detail);
  }
  private activity(row: ResourceRow): string {
    const locale = this.model.language === 'default' ? i18n.System.getSystemLanguage() : this.model.language;
    return relativeActivityDate(Date.parse(Wire.text(row.data(), 'updated_at')), this.now, locale,
      tr('agent_time_just_now'), tr('kb_minutes_ago'), tr('kb_yesterday'), i18n.System.is24HourClock() ? '24' : '12');
  }
  private rowHeight(row: ResourceRow): number {
    // The two text lines have a 2vp gap. A single line is shorter than the heart tile.
    const content = this.preview(row) ? this.scaledLength(20.3,27.7)+2+this.scaledLength(18,25.3) : this.scaledLength(32,42);
    return 30 + content;
  }
  @Computed get visibleRows(): BoundRow<ResourceRow>[] {
    return this.archivingBinding ? this.rows.filter((binding: BoundRow<ResourceRow>) => binding.key !== this.archivingBinding) : this.rows;
  }
  @Computed get listHeight(): number {
    return this.visibleRows.reduce((sum: number, binding: BoundRow<ResourceRow>) => sum+this.rowHeight(binding.value), 0)
      + Math.max(0,this.visibleRows.length-1)*this.getUIContext().px2vp(1);
  }
  private get titleInk(): ResourceColor {
    return Colors.systemControls && Colors.styleID === 'glass' ? (this.env.dark ? Color.White : Color.Black) : Colors.ink;
  }
  private get secondaryInk(): ResourceColor {
    return Colors.systemControls && Colors.styleID === 'glass' ? (this.env.dark ? '#80FFFFFF' : '#80000000') : Colors.secondary;
  }
  private get tint(): string { return this.env.dark ? '#FF6B8B' : '#D62E5C'; }
  private archiveAllowed(binding: BoundRow<ResourceRow>): boolean {
    return this.model.connected && !this.model.demo && !this.archivingBinding && !this.workspace.busy &&
      this.rows.includes(binding) && this.workspace.canArchiveConversation(binding.value);
  }
  private async archive(binding: BoundRow<ResourceRow>): Promise<void> {
    if (!this.mounted || !this.archiveAllowed(binding)) return;
    const client = this.model, owner = client.scope(), generation = client.connectionGeneration();
    this.archivingBinding = binding.key;
    try {
      const archived = await this.workspace.archive(binding.value);
      if (!this.mounted || this.model !== client || !client.owns(generation, owner)) return;
      const notice = archived ? 'phone_chat_archived' : this.workspace.error;
      if (notice) this.getUIContext().getPromptAction().showToast({message:tr(notice)});
    } finally {
      if (this.archivingBinding === binding.key) this.archivingBinding = '';
    }
  }
  private open(row: ResourceRow): void {
    const current = this.workspace.listedConversation(row);
    if (current) this.onOpen(current);
  }

  @Builder face(row: ResourceRow, preview: string, activity: string) {
    Row() {
      Stack() {
        SymbolGlyph($r('sys.symbol.heart_fill')).fontSize(String(this.scaledLength(17,23))+'vp').fontColor([this.tint])
      }.width(this.scaledLength(32,42)).height(this.scaledLength(32,42)).borderRadius(this.scaledLength(16,21))
        .backgroundColor('#24'+this.tint.slice(1)).accessibilityLevel('no')
      Column({space:2}) {
        Text(this.title(row)).fontSize(17).maxFontScale(Tokens.interfaceScaleFor(17)).lineHeight(String(this.scaledLength(20.3,27.7))+'vp')
          .fontColor(this.titleInk).maxLines(1).textOverflow({overflow:TextOverflow.Ellipsis}).id('health.history.title.'+row.id)
        if (preview) Text(preview).fontSize(15).maxFontScale(Tokens.interfaceScaleFor(15)).lineHeight(String(this.scaledLength(18,25.3))+'vp')
          .fontColor(this.secondaryInk).maxLines(1).textOverflow({overflow:TextOverflow.Ellipsis}).id('health.history.preview.'+row.id)
      }.alignItems(HorizontalAlign.Start).layoutWeight(1).margin({left:12})
      if (activity) Text(activity).fontSize(12).maxFontScale(Tokens.interfaceScaleFor(12)).lineHeight(String(this.scaledLength(14.3,21.7))+'vp')
        .fontColor(this.secondaryInk).maxLines(1).textOverflow({overflow:TextOverflow.Ellipsis}).margin({left:8}).id('health.history.activity.'+row.id)
    }.width('100%').height(this.rowHeight(row)).padding({left:20,right:20}).alignItems(VerticalAlign.Center)
  }
  @Builder rowButton(binding: BoundRow<ResourceRow>) {
    Button({type:ButtonType.Normal}) {
      this.face(binding.value, this.preview(binding.value), this.activity(binding.value))
    }.width('100%').height(this.rowHeight(binding.value)).padding(0).buttonStyle(ButtonStyleMode.TEXTUAL)
      .backgroundColor(Color.Transparent).onClick(() => this.open(binding.value)).id('health.chat.'+binding.value.id)
      .accessibilityText(this.title(binding.value))
      .accessibilityDescription([this.preview(binding.value),this.activity(binding.value)].filter((part: string) => part.length > 0).join(' · '))
      .accessibilityCustomActions(this.archiveAllowed(binding) ?
        [{name:tr('health_archive'),onAction:(): void => { this.archive(binding); }}] : undefined)
  }
  @Builder archiveMenu(binding: BoundRow<ResourceRow>) {
    Menu() {
      MenuItem({content:tr('health_archive'),symbolStartIcon:new SymbolGlyphModifier($r('sys.symbol.archivebox'))})
        .enabled(this.archiveAllowed(binding)).onClick(() => { this.archive(binding); }).id('health.history.menu.archive')
    }
  }
  @Builder archiveAction(binding: BoundRow<ResourceRow>) {
    Button({type:ButtonType.Normal}) {
      Column({space:2}) {
        SymbolGlyph($r('sys.symbol.archivebox')).fontSize('19vp').fontColor([Colors.onAction])
        Text(tr('health_archive')).fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13)).fontColor(Colors.onAction).maxLines(1)
      }
    }.width(80).height(this.rowHeight(binding.value)).padding(0).backgroundColor(Colors.action).accessibilityText(tr('health_archive'))
      .enabled(this.archiveAllowed(binding)).onClick(() => { this.archive(binding); }).id('health.chat.archive.'+binding.value.id)
  }
  @Builder entry(binding: BoundRow<ResourceRow>) {
    if (this.archiveAllowed(binding)) {
      ListItem() { this.rowButton(binding) }
        .swipeAction({end:{builder:() => { this.archiveAction(binding); },actionAreaDistance:80,onAction:() => { this.archive(binding); }}})
        .bindContextMenu(this.archiveMenu(binding),ResponseType.LongPress)
    } else {
      ListItem() { this.rowButton(binding) }
    }
  }
  build() {
    Column() {
      Text(tr('health_chats')).fontSize(15).maxFontScale(Tokens.interfaceScaleFor(15)).fontColor(this.secondaryInk)
        .height(this.scaledLength(40.3,47.7)).width('100%').padding({left:20,right:20})
      if (this.visibleRows.length) {
        List() {
          ForEach(this.visibleRows, (binding: BoundRow<ResourceRow>) => { this.entry(binding); },
            (binding: BoundRow<ResourceRow>) => binding.key)
        }.width('100%').height(this.listHeight).scrollBar(BarState.Off).edgeEffect(EdgeEffect.None)
          .nestedScroll({scrollForward:NestedScrollMode.PARENT_FIRST,scrollBackward:NestedScrollMode.PARENT_FIRST})
          .divider({strokeWidth:'1px',startMargin:20+this.scaledLength(32,42)+12,endMargin:20,color:Colors.divider})
          .backgroundColor(Colors.surface).borderRadius(26).clip(true)
      } else {
        Column() {
          if (this.workspace.loading) LoadingProgress().width(Tokens.touch).height(Tokens.touch)
          else Text(tr(this.workspace.error ? 'health_chats_unavailable' : 'health_no_chats'))
            .fontSize(15).maxFontScale(Tokens.interfaceScaleFor(15)).fontColor(this.secondaryInk).width('100%')
        }.width('100%').padding({left:20,right:20,top:15,bottom:15}).alignItems(HorizontalAlign.Start)
          .backgroundColor(Colors.surface).borderRadius(26)
      }
    }.width('100%').alignItems(HorizontalAlign.Start)
  }
}
