import { Colors } from '../design/Colors';
import { Symbols } from '../design/Symbols';
import { tr } from '../design/Strings';
import { Tokens } from '../design/Tokens';
import { TaskStartRecovery } from '../models/TaskStartRecovery';

@ComponentV2
export struct TaskStartFailureSheet {
  @Require @Param recovery:TaskStartRecovery;
  @Event onClose:()=>void=()=>{};
  @Event onEdit:()=>void=()=>{};
  @Local details:boolean=false;
  private remove():void {
    if (!this.recovery.canAct('remove')) return;
    const id=this.recovery.id;
    const remove=async ():Promise<void>=>{
      if (this.recovery.id===id && await this.recovery.act('remove') && this.recovery.id===id) this.onClose();
    };
    if (!this.recovery.confirmRemoval) { remove(); return; }
    this.getUIContext().showAlertDialog({title:tr('task_recovery_remove_title'), message:this.recovery.title,
      primaryButton:{value:tr('cancel'),action:()=>{}}, secondaryButton:{value:tr('remove'),action:()=>{remove();}}});
  }
  private summary():string {
    if (this.recovery.reasonCode==='worktree_path_taken') return tr('task_recovery_workspace_taken');
    if (this.recovery.reasonCode==='launcher_missing' && this.recovery.reasonLauncher)
      return tr('task_recovery_launcher_missing').replace('{launcher}',this.recovery.reasonLauncher);
    return this.recovery.failure;
  }
  @Builder options() {
    Menu() {
      MenuItem({content:tr('edit')}).onClick(()=>this.onEdit())
      if (this.recovery.canAct('remove')) MenuItem({content:tr('remove')}).onClick(()=>this.remove())
    }
  }
  @Builder action(key:string, label:string, prominent:boolean) {
    Button({type:ButtonType.Capsule}) {
      Text(label).fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13)).fontWeight(FontWeight.Medium)
        .fontColor(prominent ? Color.White : Colors.ink).maxLines(2)
    }.padding({left:12,right:12,top:6,bottom:6}).constraintSize({minWidth:50,minHeight:30})
      .backgroundColor(prominent ? Colors.link : Colors.control)
      .enabled(this.recovery.canAct(key==='remove'?'remove':'retry')).id('task.'+key)
      .onClick(()=>{if(key==='remove') this.remove(); else this.recovery.act('retry',key==='retry-launcher'?this.recovery.alternative:'');})
  }
  build() {
    Column() {
      Row().width(36).height(5).borderRadius(3).backgroundColor(Colors.tertiary).margin({top:5,bottom:6}).accessibilityLevel('no')
      Row({space:9}) {
        Button({ type:ButtonType.Capsule }) { SymbolGlyph($r('sys.symbol.xmark')).fontSize(16).fontColor([Colors.ink]) }
          .width(56).height(46).backgroundColor(Colors.sheetControl).onClick(()=>this.onClose()).accessibilityText(tr('close')).id('task.failure.close')
        SymbolGlyph($r('sys.symbol.exclamationmark_circle_fill')).fontSize(13).fontColor([Colors.danger])
        Text(this.recovery.title || tr('task')).fontSize(17).maxFontScale(Tokens.interfaceScaleFor(17)).fontWeight(FontWeight.Medium)
          .fontColor(Colors.ink).maxLines(1).textOverflow({overflow:TextOverflow.Ellipsis}).layoutWeight(1).id('task.failure.title')
        Button({ type:ButtonType.Capsule }) { Image(Symbols.more).width(18).height(18).fillColor(Colors.ink) }
          .width(56).height(46).backgroundColor(Colors.sheetControl).accessibilityText(tr('more')).bindMenu(this.options()).id('task.failure.more')
      }.width('100%').height(58).padding({left:16,right:16})
      Divider().strokeWidth(0.5).color(Colors.divider)
      Scroll() {
        Column({space:12}) {
          if (this.recovery.state==='starting') {
            Text(tr('task_recovery_starting')).fontSize(17).fontWeight(FontWeight.Medium).fontColor(Colors.ink).id('task.failure.starting')
            Text(tr('task_recovery_starting_detail')).fontSize(15).fontColor(Colors.secondary)
          } else if (this.recovery.state==='gone') {
            Text(tr('task_recovery_gone')).fontSize(17).fontWeight(FontWeight.Medium).fontColor(Colors.ink)
          } else {
            Row({space:8}) {
              SymbolGlyph($r('sys.symbol.exclamationmark_triangle_fill')).fontSize(14).fontColor([Colors.danger])
              Column({space:3}) {
                Text(tr('task_recovery_heading')).fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13)).fontWeight(FontWeight.Medium).fontColor(Colors.ink)
                Text(this.summary()).fontSize(15).fontColor(Colors.secondary).id('task.failure.reason')
                if (this.summary()!==this.recovery.failure && this.recovery.failure) {
                  Button(this.details?tr('collapse'):tr('details')).fontSize(13).fontColor(Colors.link).backgroundColor(Color.Transparent).onClick(()=>this.details=!this.details)
                  if (this.details) Text(this.recovery.failure).fontSize(14).fontColor(Colors.secondary)
                }
              }.layoutWeight(1).alignItems(HorizontalAlign.Start)
            }.width('100%').constraintSize({minHeight:60}).padding(12).alignItems(VerticalAlign.Top)
              .backgroundColor(Colors.sheetCard).borderRadius(12).id('task.failure.card')
            if (this.recovery.alternative) {
              Row() { this.action('retry-launcher',tr('task_recovery_instead').replace('{launcher}',this.recovery.alternative),true) }.width('100%')
            }
            Row({space:8}) {
              Row() { this.action('retry',tr('retry'),!this.recovery.alternative) }.layoutWeight(1)
              Row() { this.action('remove',tr('remove'),false) }.layoutWeight(1)
            }.width('100%')
            if (this.recovery.statusMessage()) Text(tr(this.recovery.statusMessage())).fontSize(13).fontColor(Colors.secondary).id('task.failure.error')
            if (this.recovery.state==='stale' || this.recovery.state==='unknown')
              Button(tr('refresh')).fontColor(Colors.link).backgroundColor(Color.Transparent).enabled(!this.recovery.busy).onClick(()=>this.recovery.refresh()).id('task.failure.refresh')
          }
        }.width('100%').alignItems(HorizontalAlign.Start).padding(16)
      }.layoutWeight(1).width('100%').align(Alignment.Top)
    }.width('100%').height('100%').backgroundColor(Colors.sheetBackground).id('task.failure.sheet')
  }
}
