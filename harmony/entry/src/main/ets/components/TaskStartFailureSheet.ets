import { Colors } from '../design/Colors';
import { ReadinessStyle } from '../design/PhoneStyle';
import { Symbols } from '../design/Symbols';
import { tr } from '../design/Strings';
import { Tokens } from '../design/Tokens';
import { TaskStartupQuestionCard } from './TaskStartupQuestionCard';
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
        .fontColor(prominent ? Color.White : Colors.link).maxLines(1)
    }.padding({left:12,right:12}).constraintSize({minWidth:ReadinessStyle.metric(50,62),minHeight:ReadinessStyle.metric(30,37)})
      .backgroundColor(prominent ? Colors.action : Colors.readinessControl)
      .border({width:prominent ? 0 : Tokens.divider,color:Colors.readinessEdge})
      .enabled(this.recovery.canAct(key==='remove'?'remove':'retry')).id('task.'+key)
      .onClick(()=>{if(key==='remove') this.remove(); else this.recovery.act('retry',key==='retry-launcher'?this.recovery.alternative:'');})
  }
  build() {
    Column() {
      Row().width(36).height(5).borderRadius(3).backgroundColor(Colors.tertiary).margin({top:5,bottom:6}).accessibilityLevel('no')
      Row({space:Tokens.gap}) {
        Button({type:ButtonType.Normal}) {
          Stack() { SymbolGlyph($r('sys.symbol.xmark')).fontSize(ReadinessStyle.font(15)).fontColor([Colors.link]) }
            .width(ReadinessStyle.metric(46,56)).height(ReadinessStyle.metric(46,56))
            .borderRadius(Tokens.capsule*2).backgroundColor(Colors.readinessControl)
        }.width(ReadinessStyle.metric(56,66)).height(ReadinessStyle.metric(46,56)).padding(0).backgroundColor(Color.Transparent)
          .onClick(()=>this.onClose()).accessibilityText(tr('close')).id('task.failure.close')
        SymbolGlyph($r('sys.symbol.exclamationmark_triangle_fill')).fontSize(ReadinessStyle.font(ReadinessStyle.label))
          .width(ReadinessStyle.metric(16,21)).fontColor([Colors.danger])
        Text(this.recovery.title || tr('task')).fontSize(ReadinessStyle.font(Tokens.body))
          .lineHeight(ReadinessStyle.line(21.2,28.3)).fontWeight(FontWeight.Medium).fontColor(Colors.ink)
          .maxLines(2).textOverflow({overflow:TextOverflow.Ellipsis}).layoutWeight(1).id('task.failure.title')
        Blank().width(Tokens.gap)
        Button({type:ButtonType.Normal}) {
          Stack() { Image(Symbols.more).width(ReadinessStyle.metric(17,23)).height(ReadinessStyle.metric(17,23)).fillColor(Colors.link) }
            .width(ReadinessStyle.metric(46,56)).height(ReadinessStyle.metric(46,56))
            .borderRadius(Tokens.capsule*2).backgroundColor(Colors.readinessControl)
        }.width(ReadinessStyle.metric(56,66)).height(ReadinessStyle.metric(46,56)).padding(0).backgroundColor(Color.Transparent)
          .accessibilityText(tr('more')).bindMenu(this.options()).id('task.failure.more')
      }.width('100%').height(ReadinessStyle.metric(59,69.7)).padding({left:Tokens.page,right:Tokens.page,bottom:10})
      Divider().strokeWidth(0.5).color(Colors.divider)
      Scroll() {
        Column({space:12}) {
          if (this.recovery.state==='starting') {
            Text(tr('task_recovery_starting')).fontSize(ReadinessStyle.font(Tokens.body)).fontWeight(FontWeight.Medium).fontColor(Colors.ink).id('task.failure.starting')
            Text(tr('task_recovery_starting_detail')).fontSize(ReadinessStyle.font(ReadinessStyle.body)).fontColor(Colors.secondary)
          } else if (this.recovery.state==='gone') {
            Text(tr('task_recovery_gone')).fontSize(ReadinessStyle.font(Tokens.body)).fontWeight(FontWeight.Medium).fontColor(Colors.ink)
          } else {
            Row({space:8}) {
              SymbolGlyph($r('sys.symbol.exclamationmark_triangle_fill')).fontSize(ReadinessStyle.font(ReadinessStyle.label))
                .width(ReadinessStyle.metric(15.3,22.3)).fontColor([Colors.danger])
              Column({space:Tokens.fine}) {
                Text(tr('task_recovery_heading')).fontSize(ReadinessStyle.font(ReadinessStyle.label)).lineHeight(ReadinessStyle.line(14.3,21.7)).fontWeight(FontWeight.Medium).fontColor(Colors.ink)
                Text(this.summary()).fontSize(ReadinessStyle.font(ReadinessStyle.body)).lineHeight(ReadinessStyle.line(18,25.7)).fontColor(Colors.secondary).id('task.failure.reason')
                if (this.summary()!==this.recovery.failure && this.recovery.failure) {
                  Button(this.details?tr('collapse'):tr('details')).fontSize(ReadinessStyle.font(ReadinessStyle.label)).fontColor(Colors.link).backgroundColor(Color.Transparent).onClick(()=>this.details=!this.details)
                  if (this.details) Text(this.recovery.failure).fontSize(ReadinessStyle.font(Tokens.meta)).fontColor(Colors.secondary)
                }
              }.layoutWeight(1).alignItems(HorizontalAlign.Start)
            }.width('100%').constraintSize({minHeight:60}).padding(12).alignItems(VerticalAlign.Top)
              .backgroundColor(Colors.readinessSurface).borderRadius(Tokens.inset)
              .border({width:Tokens.divider,color:Colors.readinessEdge}).shadow({radius:5,color:'#12000000',offsetY:2}).id('task.failure.card')
            if (this.recovery.question) TaskStartupQuestionCard({recovery:this.recovery,question:this.recovery.question!})
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
    }.width('100%').height('100%').backgroundColor(Colors.surface).id('task.failure.sheet')
  }
}
