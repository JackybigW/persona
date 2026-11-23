import { LengthMetrics } from '@kit.ArkUI';
import { Colors } from '../design/Colors';
import { Symbols } from '../design/Symbols';
import { tr } from '../design/Strings';
import { Tokens } from '../design/Tokens';
import { TaskStartRecovery } from '../models/TaskStartRecovery';
import { TaskStartupQuestion } from '../models/TaskStartupQuestion';

@ComponentV2
export struct TaskStartupQuestionCard {
  @Require @Param recovery:TaskStartRecovery;
  @Require @Param question:TaskStartupQuestion;
  @Local details:boolean=false;
  @Local expired:boolean=false;
  private timer:number=-1;
  aboutToAppear():void { this.watchQuestion(); }
  aboutToDisappear():void { clearTimeout(this.timer); }
  @Monitor('question') watchQuestion():void {
    clearTimeout(this.timer); this.details=false; this.expired=!this.question.open();
    if (!this.expired && Number.isFinite(this.question.until)) this.tick();
  }
  private tick():void {
    const remaining=this.question.until-Date.now();
    if (remaining<=0) {this.expired=true;return;}
    this.timer=setTimeout(()=>this.tick(),Math.min(remaining,2147483647));
  }
  private hint():string {
    const question=this.question;
    const text=tr(question.secret ? 'task_start_secret' : question.kind==='ssh_host_key' ? 'task_start_host' : 'task_start_generic');
    if (!Number.isFinite(question.until)) return text;
    const clock=new Date(question.until).toLocaleTimeString(undefined,{hour:'numeric',minute:'2-digit'});
    return text+' '+tr('task_start_until').replace('{value}',clock);
  }
  @Builder answer(choice:string, attempt:string) {
    Button({type:ButtonType.Capsule}) {
      Text(tr(this.question.label(choice)).replace('{value}',choice)).fontSize(15).maxFontScale(Tokens.interfaceScaleFor(15))
        .fontWeight(FontWeight.Medium).maxLines(1).fontColor(Colors.ink)
    }.padding({left:12,right:12,top:5,bottom:5}).constraintSize({minHeight:32}).backgroundColor(Colors.control)
      .enabled(this.recovery.canAnswer(choice,attempt)).onClick(()=>this.recovery.act('answer',choice,attempt))
      .id('readiness-choice-'+choice)
  }
  build() {
    if (!this.expired && this.question.open() && this.recovery.question?.attempt===this.question.attempt) {
      Row({space:8}) {
        Row({space:1}) {
          SymbolGlyph(Symbols.expand).fontSize('7vp').fontColor([Colors.secondary])
          Rect().width(3).height(1).fill(Colors.secondary).alignSelf(ItemAlign.End)
        }.width(16).height(13).padding(2).border({width:1,radius:2,color:Colors.secondary}).accessibilityLevel('no')
        Column({space:6}) {
          Text(tr('task_start_title').replace('{value}',this.question.line)).fontSize(15).maxFontScale(Tokens.interfaceScaleFor(15)).fontWeight(FontWeight.Medium).fontColor(Colors.ink)
          Text(this.hint()).fontSize(15).maxFontScale(Tokens.interfaceScaleFor(15)).fontColor(Colors.secondary)
          Flex({wrap:FlexWrap.Wrap,space:{main:LengthMetrics.vp(6),cross:LengthMetrics.vp(6)}}) {
            ForEach(this.question.choices,(choice:string)=>{this.answer(choice,this.question.attempt)},(choice:string)=>JSON.stringify([this.question.attempt,choice]))
          }
          if (this.question.screen) {
            Button(tr(this.details?'collapse':'details')).fontSize(13).fontColor(Colors.link).backgroundColor(Color.Transparent)
              .onClick(()=>this.details=!this.details).id('task.start.details')
            if (this.details) Text(this.question.screen).fontSize(13).fontFamily('monospace').fontColor(Colors.secondary)
          }
          if (this.recovery.error) Text(tr(this.recovery.error)).fontSize(13).fontColor(Colors.danger)
        }.layoutWeight(1).alignItems(HorizontalAlign.Start)
      }.width('100%').alignItems(VerticalAlign.Top).padding(12).backgroundColor(Colors.sheetCard).borderRadius(12).id('shell-question')
    }
  }
}
