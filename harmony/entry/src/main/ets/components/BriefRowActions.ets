import { LengthMetrics } from '@kit.ArkUI';
import { AssistantFeedModel } from '../models/AssistantFeed';
import type { BriefAction } from '../models/BriefRowActions';
import { briefTake } from '../models/BriefRowActions';
import { Wire } from '../models/Wire';
import { Colors } from '../design/Colors';
import { Tokens } from '../design/Tokens';
import { tr } from '../design/Strings';

@ComponentV2
export struct BriefRowActions {
  @Require @Param feed: AssistantFeedModel;
  @Require @Param item: Record<string,Object>;
  @Require @Param row: Record<string,Object>;
  @Require @Param section: string;
  @Param active: boolean = false;

  @Computed get outcome(): string { return this.feed.briefOutcome(this.section,this.row); }
  @Computed get error(): string { return this.feed.briefError(this.section,this.row); }

  @Builder action(kind: BriefAction, key: string, prominent: boolean) {
    Button() {
      if (this.outcome==='busy') {
        LoadingProgress().width(18).height(18).color(prominent ? Color.White : Colors.action)
      } else {
        Text(tr('brief_row_'+key)).fontSize(14).maxFontScale(Tokens.interfaceScaleFor(14)).fontWeight(FontWeight.Medium)
          .fontColor(prominent ? Color.White : Colors.action).textAlign(TextAlign.Center)
      }
    }.type(ButtonType.Capsule).constraintSize({minHeight:36}).padding({left:12,right:12,top:5,bottom:5})
      .backgroundColor(prominent ? Colors.action : Colors.card(true)).enabled(this.active && this.feed.canBriefAction(this.item,this.section,this.row,kind))
      .accessibilityText(tr('brief_row_'+key)+' '+Wire.text(this.row,'title'))
      .id('brief.'+(kind==='handled'?'later':kind)+'.'+Wire.text(this.row,'id'))
      .onClick(() => {
        const item=this.item, row=this.row, section=this.section;
        this.feed.performBriefAction(item,section,row,kind,() => this.active && this.section===section &&
          Wire.text(this.item,'id')===Wire.text(item,'id') && Wire.text(this.row,'id')===Wire.text(row,'id') &&
          Wire.text(this.row,'asked_at')===Wire.text(row,'asked_at'));
      })
  }

  build() {
    Column({space:5}) {
      if (this.outcome && this.outcome!=='busy') {
        if (this.outcome!=='completed') {
          Text(tr('brief_row_'+this.outcome)).fontSize(14).maxFontScale(Tokens.interfaceScaleFor(14))
            .fontColor(Colors.secondary).id('brief.outcome.'+Wire.text(this.row,'id'))
        }
      } else {
        Flex({direction:FlexDirection.Row,wrap:FlexWrap.Wrap,justifyContent:FlexAlign.End,space:{main:LengthMetrics.vp(6),cross:LengthMetrics.vp(5)}}) {
          if (this.section==='accept' && Wire.text(this.row,'outcome').trim()==='decide') this.action('complete','complete',false)
          if (this.section==='accept' && briefTake(this.row)) this.action('accept',briefTake(this.row),true)
          if (this.section==='later') this.action('handled','handle',false)
        }.width('100%')
        if (this.error) Text(tr(this.error)).fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13)).fontColor(Colors.secondary)
          .id('brief.error.'+Wire.text(this.row,'id'))
      }
    }.width('100%').alignItems(HorizontalAlign.End)
  }
}
