import { PhoneSectionTitle, PhoneStyle } from '../design/PhoneStyle';
import { Colors } from '../design/Colors';
import { Tokens } from '../design/Tokens';
import { tr } from '../design/Strings';
import { conversationPermissionModes } from '../core/PhoneSettingsPermissions';
import { ClientModel } from '../models/ClientModel';
import { PhoneAgentSettings } from '../models/PhoneAgentSettings';
import { Wire } from '../models/Wire';

@ComponentV2
export struct PhoneDefaultPermissionSection {
  @Require @Param model:ClientModel;
  @Param elevated:boolean=false;
  @Local form:PhoneAgentSettings=new PhoneAgentSettings();
  aboutToAppear():void { this.open(); }
  aboutToDisappear():void { this.form.close(); }
  @Monitor('model.profile','model.connected') ownerChanged():void { this.open(); }
  private open():void { this.form.open(this.model,'phone_default_permissions').catch(()=>undefined); }
  @Computed get mode():string { return Wire.text(this.form.data,'default_permission_mode'); }
  private name(value:string):string { return conversationPermissionModes.includes(value) ? tr('assistant_feed_permission_'+value) : value; }
  @Builder modeMenu() {
    Menu() {
      ForEach(conversationPermissionModes.includes(this.mode) ? conversationPermissionModes : conversationPermissionModes.concat([this.mode]),(value:string) => {
        MenuItem({content:this.name(value)}).selectIcon(true).selected(this.mode===value)
          .onClick(()=>{ this.form.setConversationPermission(value).catch(()=>undefined); }).id('settings.default_permission.'+value)
      },(value:string)=>value)
    }
  }
  build() {
    if (this.mode) { ListItem() {
      Column({space:6}) {
        PhoneSectionTitle({text:tr('phone_default_permissions')})
        Column({space:5}) {
          Button({type:ButtonType.Normal}) { Row({space:6}) {
            Text(tr('phone_new_conversation_permissions')).fontSize(17).maxFontScale(Tokens.interfaceFontMaxScale).fontColor(Colors.ink).layoutWeight(1)
            Text(this.name(this.mode)).fontSize(15).maxFontScale(Tokens.interfaceScaleFor(15)).fontColor(Colors.secondary).maxLines(2)
          }.width('100%') }.width('100%').backgroundColor(Color.Transparent).constraintSize({minHeight:44}).padding(0)
            .enabled(!this.form.saving && this.model.connected && !Wire.bool(this.form.data,'read_only')).bindMenu(this.modeMenu).id('settings.default_permission_mode')
          if (conversationPermissionModes.includes(this.mode)) Text(tr(this.mode==='bypassPermissions' ? 'header_permission_full_detail' : 'header_permission_detail_'+this.mode))
            .fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13)).fontColor(this.mode==='bypassPermissions' ? Colors.danger : Colors.secondary)
          if (this.form.notice) Text(tr(this.form.notice)).fontSize(13).fontColor(Colors.danger).id('settings.default_permission_error')
        }.width('100%').alignItems(HorizontalAlign.Start).padding({left:20,right:20,top:5,bottom:12}).backgroundColor(Colors.card(this.elevated)).borderRadius(PhoneStyle.cardRadius)
        Text(tr('phone_default_permission_footer')).fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13)).fontColor(Colors.secondary)
          .width('100%').padding({left:20,right:20,bottom:12})
      }.width('100%').alignItems(HorizontalAlign.Start)
    } } else if (this.form.loading || this.form.error) { ListItem() {
      if (this.form.loading) LoadingProgress().height(24).width(24)
      else Column({space:6}) {
        Text(tr(this.form.error)).fontSize(14).fontColor(Colors.danger)
        Button(tr('retry'),{buttonStyle:ButtonStyleMode.TEXTUAL}).onClick(()=>{ this.form.read().catch(()=>undefined); })
      }.width('100%')
    } }
  }
}
