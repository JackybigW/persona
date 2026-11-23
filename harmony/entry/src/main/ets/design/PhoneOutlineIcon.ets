import { Colors } from './Colors';

interface GlyphDrawing {
  name: string;
  width: number;
  height: number;
  path: string;
  pen: number;
  filled?: boolean;
}

const cube = 'M12 1.2 L21.5 6.6 L21.5 17.4 L12 22.8 L2.5 17.4 L2.5 6.6 Z M2.5 6.6 L12 12 L21.5 6.6 M12 12 L12 22.8';
const drawings: GlyphDrawing[] = [
  {name:'dispatchToAgent',width:18,height:18,pen:2.1,path:'M2.1 3.8 V6.1 Q2.1 10.0 6.2 10.0 H15.4 M10.5 5.4 L15.4 10.0 L10.5 14.6'},
  {name:'networkShield',width:25,height:25,pen:1.45,path:'M16 20.5 A9.8 9.8 0 1 1 20 12 M2 10.6 H20 M4 5.4 Q11 9 18 5.4 M4.4 16.7 Q8.2 14.6 12.8 15 M10.7 1.5 Q2.7 11 11.3 21.1 M11 1.5 Q15.5 6.1 15.2 11.4 M11 1.9 V20.5 M18.5 12.5 Q20.9 14.1 23.6 14.2 V17.7 Q23 21.3 18.5 23.8 Q14 21.3 13.5 17.7 V14.2 Q16.1 14.1 18.5 12.5 Z M18.5 14.8 V21.4'},
  {name:'cube',width:24,height:24,path:cube,pen:1.5},
  {name:'package',width:24,height:24,path:cube+' M7.2 4 L16.8 9.4',pen:1.5},
  {name:'laptop',width:36,height:22,path:'M4.7 17.8 V4.4 Q4.7 3 6.2 3 H29.8 Q31.3 3 31.3 4.4 V17.8 Z M2.5 19 H33.5',pen:2.2},
  {name:'server',width:29,height:22,path:'M3.5 2 H25.5 Q26.5 2 26.5 3 V19 Q26.5 20 25.5 20 H3.5 Q2.5 20 2.5 19 V3 Q2.5 2 3.5 2 Z M2.5 8 H26.5 M2.5 14 H26.5 M21.5 5 H23 M21.5 11 H23 M21.5 17 H23',pen:2},
  {name:'desktop',width:30,height:24,path:'M3 1.8 H27 Q28 1.8 28 3 V18 H2 V3 Q2 1.8 3 1.8 Z M2 15 H28 M15 18 V21.8 M10 22 H20',pen:2},
  {name:'sparkles',width:20,height:22,path:'M12.4 4.5 Q13 10.7 18.7 12 Q13 13.3 12.4 20 Q11.7 13.3 6.1 12 Q11.7 10.7 12.4 4.5 Z M4.3 1 L5.1 4.3 L8.1 5.1 L5.1 6 L4.3 9.2 L3.4 6 L0.7 5.1 L3.4 4.3 Z',pen:0.4,filled:true},
  {name:'forward',width:9,height:14,path:'M1.7 1.7 L6.9 7 L1.7 12.3',pen:1.9},
  {name:'calendarAgenda',width:27,height:20,path:'M7 1.8 H25 Q26 1.8 26 3 V17 Q26 18.2 24.8 18.2 H7 Q5.8 18.2 5.8 17 V3 Q5.8 1.8 7 1.8 Z M6 8 H26 M1.6 3 H2.3 M1.6 9 H2.3 M1.6 15 H2.3',pen:1.6},
  {name:'books',width:25,height:24,path:'M1.5 5 H6 V21.5 H1.5 Z M8.1 1.7 H13.2 V21.5 H8.1 Z M8.5 6 H12.8 M1.9 17 H5.6 M15.1 3.9 L19.4 2.9 L23.6 20.4 L19.3 21.5 Z',pen:1.6},
  {name:'choices',width:11,height:15,path:'M1.6 5.4 L5.5 1.6 L9.4 5.4 M1.6 9.6 L5.5 13.4 L9.4 9.6',pen:1.8},
];

// The SVG path commands and their pen use physical pixels within the viewport. The outer frame follows the interface size.
@ComponentV2
export struct PhoneOutlineIcon {
  @Param name: string = 'cube';
  @Param iconScale: number = 1;
  @Param color: ResourceColor = Colors.link;

  @Computed get drawing(): GlyphDrawing { return drawings.find((entry: GlyphDrawing) => entry.name === this.name) || drawings[0]; }

  build() {
    Shape() {
      Path().commands(this.drawing.path).fill(this.drawing.filled ? this.color : Color.Transparent).stroke(this.color)
        .strokeWidth(String(this.drawing.pen)+'px').strokeLineCap(LineCapStyle.Round).strokeLineJoin(LineJoinStyle.Round)
    }.viewPort({x:0,y:0,width:String(this.drawing.width)+'px',height:String(this.drawing.height)+'px'})
      .width(this.drawing.width*this.iconScale).height(this.drawing.height*this.iconScale)
      .accessibilityLevel('no')
  }
}
