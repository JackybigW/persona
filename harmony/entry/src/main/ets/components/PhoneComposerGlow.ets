import { accessibility } from '@kit.AccessibilityKit';
import { NativeEnvironment, nativeEnvironment } from '../platform/NativeEnvironment';

const LIGHT: string[] = ['#2E6FD0', '#5856D6', '#7A5AF8', '#D62E5C'];
const DARK: string[] = ['#5B9BFF', '#5E5CE6', '#9D8BFF', '#FF6B8B'];

/** A three-second angular sweep around the inline composer. Reduce Motion leaves a stationary colored rim. */
@ComponentV2
export struct PhoneComposerGlow {
  @Param barWidth: number = 0;
  @Param barHeight: number = 0;
  @Local env: NativeEnvironment = nativeEnvironment;
  private ctx: CanvasRenderingContext2D = new CanvasRenderingContext2D(new RenderingContextSettings(true));
  private ready: boolean = false;
  private visible: boolean = true;
  private reduced: boolean = false;
  private timer: number = -1;
  private frameWidth: number = 0;
  private frameHeight: number = 0;
  private readonly animationChanged = (value: boolean): void => {
    this.reduced = value;
    this.schedule();
  };

  aboutToAppear(): void {
    this.reduced = this.env.reduceMotion;
    try { accessibility.onAnimationReduceStateChange(this.animationChanged); } catch (_) {}
  }

  aboutToDisappear(): void {
    this.ready = false;
    this.schedule();
    try { accessibility.offAnimationReduceStateChange(this.animationChanged); } catch (_) {}
  }

  @Monitor('env.foreground', 'env.dark') refresh(): void { this.schedule(); }

  private schedule(): void {
    if (this.timer >= 0) {
      clearInterval(this.timer);
      this.timer = -1;
    }
    if (!this.ready || !this.visible || !this.env.foreground) return;
    this.paint();
    if (!this.reduced) this.timer = setInterval(() => this.paint(), 40);
  }

  private paint(): void {
    const ctx = this.ctx, w = this.frameWidth, h = this.frameHeight;
    if (!this.ready || w <= 40 || h <= 40) return;
    const inset = 20, r = Math.max(0, Math.min(25, (h - 2 * inset) / 2));
    const colors = this.env.dark ? DARK : LIGHT;
    const angle = this.reduced ? 0 : Date.now() % 3000 / 3000 * 2 * Math.PI;
    const color = ctx.createConicGradient(angle, w / 2, h / 2);
    colors.forEach((value: string, index: number) => color.addColorStop(index / 4, value));
    color.addColorStop(1, colors[0]);
    ctx.clearRect(0, 0, w, h);
    ctx.beginPath();
    ctx.moveTo(inset + r, inset);
    ctx.arcTo(w - inset, inset, w - inset, h - inset, r);
    ctx.arcTo(w - inset, h - inset, inset, h - inset, r);
    ctx.arcTo(inset, h - inset, inset, inset, r);
    ctx.arcTo(inset, inset, w - inset, inset, r);
    ctx.closePath();
    ctx.strokeStyle = color;
    ctx.globalAlpha = .035; ctx.lineWidth = 40; ctx.stroke();
    ctx.globalAlpha = .045; ctx.lineWidth = 30; ctx.stroke();
    ctx.globalAlpha = .11; ctx.lineWidth = 19; ctx.stroke();
    ctx.globalAlpha = .2; ctx.lineWidth = 11; ctx.stroke();
    ctx.globalAlpha = .31; ctx.lineWidth = 6; ctx.stroke();
    ctx.globalAlpha = 1; ctx.lineWidth = 2; ctx.stroke();
    if (!this.reduced) {
      const highlight = ctx.createConicGradient(angle, w / 2, h / 2);
      highlight.addColorStop(0, '#00FFFFFF');
      highlight.addColorStop(.42, '#00FFFFFF');
      highlight.addColorStop(.5, this.env.dark ? '#73FFFFFF' : '#D9FFFFFF');
      highlight.addColorStop(.58, '#00FFFFFF');
      highlight.addColorStop(1, '#00FFFFFF');
      ctx.strokeStyle = highlight;
      ctx.stroke();
    }
  }

  build() {
    Canvas(this.ctx).width(this.barWidth + 40).height(this.barHeight + 40).hitTestBehavior(HitTestMode.None).accessibilityLevel('no')
      .onReady(() => { this.ready = true; this.schedule(); })
      .onAreaChange((_: Area, area: Area) => {
        this.frameWidth = Number(area.width);
        this.frameHeight = Number(area.height);
        this.paint();
      })
      .onVisibleAreaChange([0], (visible: boolean, ratio: number) => {
        this.visible = visible || ratio > 0;
        this.schedule();
      }).id('chat.reply.glow')
  }
}
