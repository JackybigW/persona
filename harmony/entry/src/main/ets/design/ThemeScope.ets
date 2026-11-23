import { Theme } from '@kit.ArkUI';
import { Colors } from './Colors';

/** Explicit attributes use resolved theme tokens rather than global system resource IDs. */
@ComponentV2
export struct ThemeScope {
  @Param systemControls: boolean = false;
  @Param selectedAccentInk: string = '';
  @BuilderParam content: () => void;
  aboutToAppear(): void { this.controlAppearanceChanged(); }
  @Monitor('systemControls', 'selectedAccentInk') controlAppearanceChanged(): void {
    Colors.systemControls = this.systemControls;
    Colors.selectedAccentInk = this.selectedAccentInk;
  }
  onWillApplyTheme(theme: Theme): void {
    Colors.action = theme.colors.compBackgroundEmphasize;
    Colors.onAction = theme.colors.fontOnPrimary;
    Colors.link = theme.colors.fontEmphasize;
    Colors.iconAccent = theme.colors.iconEmphasize;
  }
  build() { this.content() }
}
