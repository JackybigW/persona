import { Tokens } from '../design/Tokens';
import { nativeEnvironment } from '../platform/NativeEnvironment';
import { DictationPreference, dictationPreference } from '../platform/DictationPreference';
import { inputMethod } from '@kit.IMEKit';
import { i18n } from '@kit.LocalizationKit';
import { PhoneStyle, PhoneSectionTitle, PhoneSettingsRow } from '../design/PhoneStyle';
import { Colors } from '../design/Colors';
import { Symbols } from '../design/Symbols';
import { tr } from '../design/Strings';
import { supportedInterfaceLocale } from '../core/ClientLocale';
import { ClientModel } from '../models/ClientModel';
import { ResourceRow, Wire } from '../models/Wire';
import { MeetingPreferences, settingsMeetingRoleLabel } from '../models/MeetingPreferences';
import { machineOnline } from '../models/MachineDashboard';

const AGENT_OPTIONS:string[] = ['phone_agent_folders','phone_claude_config','phone_codex_config','phone_agy_config','triage'];
const SEARCH_ROUTES:string[] = ['phone_agent_followup','phone_agent_layout','launchers','engine','schedules'];
const LOOK_SEARCH_OPTIONS:string[] = ['phone_look_color','phone_look_style','phone_look_scheme'];
const ASSISTANT_OPTIONS:string[] = ['models','assistant','channels','connectors','phone_knowledge_selection','phone_meeting_options','phone_agent_options','machines'];
const DEVICE_OPTIONS:string[] = ['appearance','phone_language_options','notifications','translation','phone','privacy'];
const HUB_OPTIONS:string[] = ['managers','phone_look_avatar','extensions','memory','ssh','hub_update','node_update','ledger','diagnostics'];
const LANGUAGES:string[] = ['default','zh-CN','en-US','ja-JP'];

function languageName(tag:string):string {
  const locale = supportedInterfaceLocale(tag);
  return locale === 'zh-Hans' ? '简体中文' : locale === 'ja' ? '日本語' : 'English';
}

function currentLanguage(model:ClientModel):string {
  return model.language === 'default' ? tr('phone_settings_system_language').replace('{language}',languageName(i18n.getSystemLanguage())) : languageName(model.language);
}

function settingIcon(id:string):Resource {
  switch (id) {
    case 'models': return $r('sys.symbol.square_stack_3d');
    case 'assistant': return Symbols.persona;
    case 'channels': return $r('sys.symbol.ellipsis_bubble');
    case 'connectors': return $r('sys.symbol.link');
    case 'extensions': return Symbols.apps;
    case 'phone_knowledge_selection':
    case 'memory': return $r('sys.symbol.book');
    case 'phone_meeting_options': return Symbols.calendar;
    case 'phone_agent_folders':
    case 'phone_claude_config':
    case 'phone_codex_config':
    case 'phone_agy_config':
    case 'phone_agent_options':
    case 'triage': return $r('sys.symbol.slider_horizontal_2');
    case 'phone_agent_followup': return Symbols.inbox;
    case 'phone_agent_layout': return Symbols.spaces;
    case 'launchers': return $r('sys.symbol.play_fill');
    case 'engine': return $r('sys.symbol.computer');
    case 'machines': return $r('sys.symbol.desktop');
    case 'appearance': return Symbols.appearance;
    case 'phone_language_options': return $r('sys.symbol.textformat');
    case 'notifications': return $r('sys.symbol.bell');
    case 'translation': return $r('sys.symbol.translate');
    case 'phone': return $r('sys.symbol.phone_doc');
    case 'privacy': return $r('sys.symbol.checkmark_shield');
    case 'managers': return Symbols.agents;
    case 'ssh': return $r('sys.symbol.link');
    case 'ledger': return $r('sys.symbol.doc_plaintext');
    case 'hub_update':
    case 'node_update': return Symbols.refresh;
    case 'diagnostics': return $r('sys.symbol.info_circle');
    case 'schedules': return Symbols.calendar;
    default: return Symbols.gear;
  }
}

function settingArtwork(id:string):Resource|undefined {
  switch (id) {
    case 'models': return $r('app.media.phone_settings_model');
    case 'assistant': return $r('app.media.phone_settings_profile');
    case 'channels': return $r('app.media.phone_settings_channels');
    case 'connectors': return $r('app.media.phone_settings_connectors');
    case 'phone_knowledge_selection': return $r('app.media.phone_settings_knowledge');
    case 'phone_meeting_options': return $r('app.media.phone_settings_meeting');
    case 'phone_language_options': return $r('app.media.phone_settings_language');
    case 'machines': return $r('app.media.phone_settings_machines');
    case 'phone_agent_options':
    case 'phone_agent_folders':
    case 'phone_claude_config':
    case 'phone_codex_config':
    case 'phone_agy_config': return $r('app.media.phone_settings_options');
    case 'triage': return $r('app.media.phone_settings_triage');
    case 'phone_agent_followup': return $r('app.media.phone_settings_followup');
    case 'phone_agent_layout': return $r('app.media.phone_settings_list');
    default: return undefined;
  }
}

function settingLabel(id:string, label:(page:string)=>string):string {
  if (id === 'phone_agent_options') return tr('settings_agent_options_title');
  if (id === 'phone_language_options') return tr('phone_settings_language_dictation');
  if (id === 'models') return tr('phone_settings_models');
  if (id === 'assistant') return tr('phone_settings_profile');
  if (id === 'appearance') return tr('phone_settings_display');
  if (id === 'phone_meeting_options') return tr('meetings');
  if (id === 'machines') return tr('phone_machines');
  if (id==='phone_knowledge_selection') return tr(id);
  if (AGENT_OPTIONS.includes(id) && id!=='triage' || id==='phone_agent_followup' || id==='phone_agent_layout') return tr(id);
  return tr(label(id));
}

const ALIASES:Record<string,string> = {
  'phone_look_avatar':'phone_look_avatar_alias','phone_look_color':'phone_look_color_alias','phone_look_style':'phone_look_style_alias','phone_look_scheme':'phone_look_scheme_alias',
  'models':'settings_alias_models', 'appearance':'settings_alias_appearance', 'phone_language_options':'settings_alias_language',
  'machines':'settings_alias_machines', 'phone_agent_options':'settings_alias_agents'
};

@ComponentV2
export struct SettingsDirectory {
  @Param elevated: boolean = false;
  @Require @Param model: ClientModel;
  @Param ownerName: string = '';
  @Param avatar: string = '';
  @Param initialQuery: string = '';
  @Param showDismissSearch: boolean = false;
  @Event onDismissSearch: () => void = () => {};
  @Local query: string = '';
  @Local meeting:MeetingPreferences=new MeetingPreferences();
  @Require @Param label: (page: string) => string;
  @Event onOpen: (page: string) => void = () => {};
  @Event onQueryChange: (query: string) => void = () => {};
  private field: TextInputController = new TextInputController();
  private mounted: boolean = false;

  aboutToAppear(): void { this.mounted=true; this.query=this.initialQuery; this.meeting.open(this.model); }
  aboutToDisappear():void { this.mounted=false; this.meeting.close(); }
  @Monitor('model.profile','model.features','model.connected') meetingOwnerChanged():void { this.meeting.open(this.model); }

  @Computed get managers():ResourceRow[] {
    return this.model.listedManagers().filter((r:ResourceRow) => !r.isInstallingManager());
  }
  private currentManager():ResourceRow | undefined {
    return this.managers.find((r:ResourceRow) => r.id === this.model.profile.manager);
  }
  private account():string { return this.currentManager()?.title || this.ownerName || tr('persona'); }
  private subtitle(id:string):string {
    if (id==='phone_meeting_options') return this.meeting.ready ? tr(settingsMeetingRoleLabel(this.meeting.role)) : '';
    if (id === 'phone_language_options') return currentLanguage(this.model);
    if (id === 'machines') {
      const profile = this.model.profile;
      const roster = this.model.directoryMachines.filter((r:ResourceRow) => r.group === profile.id && (!profile.manager || r.manager === profile.manager));
      const machines = roster.length ? roster : this.model.machines;
      if (!machines.length && !this.model.connected) return '';
      return tr('machine_online_ratio').replace('{online}',String(machines.filter((r:ResourceRow) => machineOnline(r)).length))
        .replace('{total}',String(machines.length));
    }
    return '';
  }
  @Computed get results():string[] {
    const query = this.query.trim().toLowerCase();
    if (!query) return [];
    const words = query.split(/\s+/);
    const direct = ASSISTANT_OPTIONS.concat(SEARCH_ROUTES,DEVICE_OPTIONS,HUB_OPTIONS,LOOK_SEARCH_OPTIONS);
    const agentDestinations = AGENT_OPTIONS.filter((id:string) => id!=='phone_agent_folders');
    return direct.concat(agentDestinations).filter((id:string) => {
      const aliases = ALIASES[id] ? tr(ALIASES[id]) : '';
      const terms = [settingLabel(id,this.label), this.subtitle(id), aliases].join(' ').toLocaleLowerCase();
      return words.every((word:string) => terms.includes(word));
    });
  }
  private restoreSearchFocus(): void {
    const query=this.initialQuery, scope=this.model.scope();
    if (!this.elevated || !query.trim()) return;
    const ui=this.getUIContext();
    setTimeout(() => {
      if (!this.mounted || scope!==this.model.scope() || query!==this.query) return;
      ui.getFocusController().requestFocus('settings.search');
      this.field.caretPosition(this.query.length);
    },0);
  }
  private openEntry(id:string):void {
    this.field.stopEditing();
    this.onOpen(id);
  }
  @Builder managerMenu(scope:string) {
    Menu() {
      ForEach(this.managers,(m:ResourceRow) => {
        MenuItem({content:m.title || m.id}).selected(m.id === this.model.profile.manager).selectIcon(true)
          .onClick(() => this.model.selectListedManager(m.id,scope))
      },(m:ResourceRow) => m.id)
    }
  }
  @Builder identity() {
    Button({type:ButtonType.Normal}) { Row({space:9}) {
      Stack({alignContent:Alignment.BottomEnd}) {
        Text(this.account().slice(0,1)).fontSize('16vp').fontWeight(FontWeight.Bold).fontColor(Color.White)
          .width(32).height(32).textAlign(TextAlign.Center).backgroundColor(Colors.ai).borderRadius(16)
        Circle().width(9).height(9).fill(this.model.connected ? Colors.positive : Colors.secondary)
          .stroke(Colors.listGroup(this.elevated)).strokeWidth(1.5)
      }.width(32).height(32)
      Text(this.currentManager() ? tr('settings_assistant_named').replace('{name}',this.account()) : tr('phone_settings_assistant'))
        .fontSize(14).maxFontScale(Tokens.interfaceScaleFor(14)).fontWeight(FontWeight.Bold).fontColor(Colors.ink).layoutWeight(1)
      if (this.currentManager()) Text(tr('phone_settings_switch')).fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13)).fontColor(Colors.secondary)
    }.width('100%').padding({left:15,right:13,top:13,bottom:10})
      }.width('100%').constraintSize({minHeight:PhoneStyle.formSize(64,78,nativeEnvironment.fontScale)}).padding(0).backgroundColor(Color.Transparent).accessibilityText(tr('settings_this_assistant')+' · '+this.account())
      .bindMenu(this.managerMenu(this.model.scope())).id('settings.assistant_switch')
  }
  @Builder section(title:string, entries?:string[]) {
    ListItem() {
      Column() {
        if (title) PhoneSectionTitle({text:title})
        Column() {
          ForEach(entries ?? this.results,(id:string,ix:number) => {
            PhoneSettingsRow({elevated:this.elevated,label:settingLabel(id,this.label),detail:this.subtitle(id),symbol:settingIcon(id),artwork:settingArtwork(id),tint:Colors.iconAccent,
              identity:'settings.'+id,last:ix+1===(entries ?? this.results).length,action:() => this.openEntry(id)})
          },(id:string,ix:number) => JSON.stringify([id,ix,this.subtitle(id),(entries ?? this.results).length]))
          if (!(entries ?? this.results).length) Text(tr('no_results')).fontSize(16).maxFontScale(Tokens.interfaceScaleFor(16)).fontColor(Colors.secondary).width('100%').padding(15).id('settings.no_results')
        }.width('100%').clip(true).backgroundColor(Colors.card(this.elevated)).borderRadius(PhoneStyle.cardRadius)
      }.width('100%')
    }
  }
  build() {
    List() {
      ListItem() {
        Row({space:8}) {
          Row({space:7}) {
          SymbolGlyph($r('sys.symbol.magnifyingglass')).fontSize('16vp').fontColor([Colors.secondary])
          TextInput({text:this.query,placeholder:tr('phone_settings_search'),controller:this.field})
            .fontSize(Tokens.interfaceFontVp(16,nativeEnvironment.fontScale)).placeholderFont({size:Tokens.interfaceFontVp(16,nativeEnvironment.fontScale)})
            .height(PhoneStyle.formSize(44,58,nativeEnvironment.fontScale)).padding(0)
            .layoutWeight(1).backgroundColor(Color.Transparent).cancelButton({style:CancelButtonStyle.INPUT})
            .enterKeyType(EnterKeyType.Search).onSubmit(() => this.field.stopEditing())
            .enableKeyboardOnFocus(true).onAppear(() => this.restoreSearchFocus())
            .defaultFocus(this.elevated && this.initialQuery.trim().length>0)
            .onChange((v:string) => { this.query = v; this.onQueryChange(v); }).id('settings.search')
        }.padding({left:12,right:8}).layoutWeight(1).borderRadius(32).backgroundColor(this.elevated ? Colors.sheetControl : Colors.control)
          if (this.showDismissSearch && this.query.trim()) Button({type:ButtonType.Circle}) {
            SymbolGlyph($r('sys.symbol.xmark')).fontSize('18vp').fontColor([Colors.ink])
          }.width(PhoneStyle.formSize(44,58,nativeEnvironment.fontScale)).height(PhoneStyle.formSize(44,58,nativeEnvironment.fontScale)).padding(0).backgroundColor(Colors.card(this.elevated)).accessibilityText(tr('close'))
            .onClick(() => { this.field.stopEditing(); this.onDismissSearch(); }).id('settings.close')
        }.width('100%').margin({bottom:10})
      }
      if (this.query.trim()) {
        this.section(tr('settings_search_results'))
      } else {
        ListItem() { this.identity() }
        this.section('',ASSISTANT_OPTIONS)
        ListItem() {
          Text(tr('phone_settings_owner_note').replace('{name}',this.account())).fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13))
            .fontColor(Colors.secondary).width('100%').padding({left:13,right:12,top:6,bottom:5})
        }
        this.section(tr('phone_settings_device'),DEVICE_OPTIONS)
        this.section(tr('phone_settings_hub'),HUB_OPTIONS)
      }
    }.width('100%').height('100%').padding({left:PhoneStyle.page,right:PhoneStyle.page,top:this.showDismissSearch && this.query.trim() ? 16 : 0,bottom:24}).backgroundColor(Colors.listGroup(this.elevated))
  }
}

@ComponentV2
export struct PhoneSettingsOptions {
  @Param elevated: boolean = false;
  @Local dictation: DictationPreference = dictationPreference;
  @Require @Param model:ClientModel;
  @Require @Param page:string;
  @Require @Param label:(page:string)=>string;
  @Event onOpen:(page:string)=>void = () => {};
  private mounted:boolean = false;
  aboutToAppear():void { this.mounted = true; this.dictation.load(this.getUIContext().getHostContext()); }
  private dictationLabel(choice: string): string {
    return choice === 'zh-CN' ? languageName(choice) : tr(choice === 'keyboard' ? 'phone_system_input' : 'auto');
  }
  @Builder dictationMenu() {
    Menu() {
      ForEach(['auto','zh-CN','keyboard'], (choice: string) => {
        MenuItem({content: this.dictationLabel(choice)}).selected(this.dictation.choice === choice).selectIcon(true)
          .enabled(!this.dictation.saving).onClick(() => {
            this.dictation.choose(choice).then((saved: boolean) => {
              if (!saved && this.mounted) this.getUIContext().getPromptAction().showToast({message:tr('cache_unavailable')});
            });
          }).id('settings.dictation.' + choice)
      }, (choice: string) => choice)
      MenuItem({content: tr('phone_keyboard_settings')}).onClick(() => {
        inputMethod.getSetting().showOptionalInputMethods().catch(() => {
          if (this.mounted) this.getUIContext().getPromptAction().showToast({message:tr('failed')});
        });
      }).id('settings.dictation.system')
    }
  }
  aboutToDisappear():void { this.mounted = false; }
  private choose(language:string):void {
    const owner=this.model;
    if (language===owner.language || owner.languageChanging) return;
    const scope=owner.scope(), generation=owner.connectionGeneration();
    owner.setLanguage(language).then(() => {
      if (this.mounted && this.model===owner && owner.owns(generation,scope) && language==='default' && owner.language==='default')
        this.getUIContext().getPromptAction().showToast({message:tr('language_after_restart')});
    }).catch(() => {});
  }
  build() {
    List() {
      if (this.page === 'phone_agent_options') {
        ListItem() {
        Column() {
          ForEach(AGENT_OPTIONS,(id:string,ix:number) => {
            PhoneSettingsRow({elevated:this.elevated,label:settingLabel(id,this.label),symbol:settingIcon(id),artwork:settingArtwork(id),tint:Colors.iconAccent,
              identity:'settings.'+id,last:ix === AGENT_OPTIONS.length-1,action:() => this.onOpen(id)})
          },(id:string) => id)
        }.width('100%').clip(true).backgroundColor(Colors.card(this.elevated)).borderRadius(PhoneStyle.cardRadius)
        }
        ListItem() { Text(tr('phone_agent_tools_note')).fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13)).fontColor(Colors.secondary).width('100%').padding({left:7,right:7,top:7,bottom:10}) }
      } else {
        ListItem() {
        Column() {
          ForEach(LANGUAGES,(lang:string,ix:number) => {
            Row({space:7}) {
              Text(lang==='default' ? tr('phone_settings_system_language').replace('{language}',languageName(i18n.getSystemLanguage())) : languageName(lang))
                .fontSize(17).maxFontScale(Tokens.interfaceScaleFor(17)).fontColor(Colors.ink).layoutWeight(1)
              if (this.model.language===lang) SymbolGlyph(Symbols.check).fontSize('17vp').width(21).flexShrink(0).fontColor([Colors.link])
            }.width('100%').constraintSize({minHeight:PhoneStyle.groupedRowHeight(nativeEnvironment.fontScale)}).padding({left:20,right:20,top:6,bottom:6})
              .enabled(!this.model.languageChanging).accessibilitySelected(this.model.language===lang).accessibilityGroup(true)
              .onClick(() => this.choose(lang)).id('settings.language.'+lang)
            if (ix<LANGUAGES.length-1) Divider().strokeWidth(0.5).color(Colors.divider).margin({left:20})
          },(lang:string) => lang)
        }.width('100%').clip(true).backgroundColor(Colors.card(this.elevated)).borderRadius(PhoneStyle.cardRadius)
        }
        ListItem() { Text(tr('phone_language_system_note')).fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13)).fontColor(Colors.secondary).width('100%').padding({left:20,right:20,top:7,bottom:20}) }
        ListItem() {
          Row({space:8}) {
            Text(tr('phone_dictation')).fontSize(16).maxFontScale(Tokens.interfaceScaleFor(16)).fontColor(Colors.ink).layoutWeight(1)
            Text(this.dictationLabel(this.dictation.choice)).fontSize(15).maxFontScale(Tokens.interfaceScaleFor(15)).fontColor(Colors.link)
            SymbolGlyph(Symbols.expand).fontSize('12vp').fontColor([Colors.secondary])
          }.constraintSize({minHeight:PhoneStyle.groupedRowHeight(nativeEnvironment.fontScale)}).width('100%').padding({left:20,right:20,top:7,bottom:7})
            .backgroundColor(Colors.card(this.elevated)).borderRadius(PhoneStyle.cardRadius).accessibilityGroup(true)
            .bindMenu(this.dictationMenu()).accessibilityRole(AccessibilityRoleType.BUTTON).id('settings.dictation.choice')
        }
        ListItem() { Text(tr('phone_dictation_automatic_note')).fontSize(13).maxFontScale(Tokens.interfaceScaleFor(13)).fontColor(Colors.secondary).width('100%').padding({left:20,right:20,top:7,bottom:10}) }
      }
    }.width('100%').height('100%').padding({left:16,right:16,top:35,bottom:24}).backgroundColor(Colors.listGroup(this.elevated))
  }
}
