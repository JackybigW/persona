// Semantic native colors; appearance resources own the light and dark variants.
@ObservedV2
class NativeColors {
  readonly surface: ResourceColor = $r('app.color.persona_surface');
  readonly codeCanvas: ResourceColor = $r('app.color.persona_code_canvas');
  readonly inlineCodeInk: ResourceColor = $r('app.color.persona_inline_code_ink');
  readonly codeHunk: ResourceColor = $r('app.color.persona_code_hunk');
  readonly diffAdded: ResourceColor = $r('app.color.persona_diff_added');
  readonly diffRemoved: ResourceColor = $r('app.color.persona_diff_removed');
  readonly diffModified: ResourceColor = $r('app.color.persona_diff_modified');
  readonly codeAddedFill: ResourceColor = $r('app.color.persona_code_added_fill');
  readonly codeRemovedFill: ResourceColor = $r('app.color.persona_code_removed_fill');
  readonly canvas: ResourceColor = $r('app.color.persona_canvas');
  readonly grouped: ResourceColor = $r('app.color.persona_grouped');
  readonly sheetBackground: ResourceColor = $r('app.color.persona_sheet_background');
  readonly sheetCard: ResourceColor = $r('app.color.persona_sheet_card');
  readonly sheetControl: ResourceColor = $r('app.color.persona_sheet_control');
  readonly listBackground: ResourceColor = $r('app.color.persona_list_background');
  card(elevated: boolean): ResourceColor { return elevated ? this.sheetCard : this.surface; }
  listGroup(elevated: boolean): ResourceColor { return elevated ? this.sheetBackground : this.grouped; }
  readonly ink: ResourceColor = $r('app.color.persona_ink');
  readonly secondary: ResourceColor = $r('app.color.persona_secondary');
  readonly tertiary: ResourceColor = $r('app.color.project_card_tertiary');
  @Trace action: ResourceColor = $r('sys.color.comp_background_emphasize');
  @Trace onAction: ResourceColor = $r('sys.color.font_on_primary');
  @Trace link: ResourceColor = $r('sys.color.font_emphasize');
  @Trace iconAccent: ResourceColor = $r('sys.color.icon_emphasize');
  readonly prominentRim: ResourceColor = $r('app.color.phone_prominent_rim');
  @Trace systemControls: boolean = false;
  @Trace selectedAccentInk: ResourceColor = $r('sys.color.font_emphasize');
  @Computed get selectedFilterFill(): ResourceColor { return this.systemControls ? $r('app.color.phone_filter_fill') : this.selection; }
  @Computed get selectedFilterInk(): ResourceColor { return this.systemControls ? $r('app.color.phone_filter_ink') : this.selectedAccentInk; }
  readonly control: ResourceColor = $r('app.color.persona_control');
  readonly selection: ResourceColor = $r('app.color.persona_selection');
  readonly question: ResourceColor = $r('app.color.persona_question');
  readonly needs: ResourceColor = $r('app.color.persona_needs');
  readonly working: ResourceColor = $r('app.color.persona_working');
  readonly unread: ResourceColor = $r('app.color.persona_unread');
  readonly needsMark: ResourceColor = $r('app.color.persona_needs_mark');
  readonly workingMark: ResourceColor = $r('app.color.persona_working_mark');
  readonly unreadMark: ResourceColor = $r('app.color.persona_unread_mark');
  readonly positive: ResourceColor = $r('app.color.persona_positive');
  readonly divider: ResourceColor = $r('app.color.persona_divider');
  readonly danger: ResourceColor = $r('app.color.persona_danger');
  readonly ai: ResourceColor = $r('app.color.persona_ai');
  readonly teal: ResourceColor = $r('app.color.persona_teal');
  readonly subtle: ResourceColor = $r('app.color.persona_subtle');
}

export const Colors: NativeColors = new NativeColors();
