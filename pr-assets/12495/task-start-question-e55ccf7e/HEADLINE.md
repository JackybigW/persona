# 启动失败面板的标题

在 402 逻辑宽的两部模拟器上，浅色和深色都完整显示两行。最大字号下，iPhone 的第二行能显示到「work」，鸿蒙显示到「wor」。两端的原生控件树都保留完整标题「Prepare the demo workspace」。

| 状态 | iPhone 原图 | 鸿蒙原图 |
| --- | --- | --- |
| 浅色 | [查看](light-iphone.png) | [查看](light-after.png) |
| 深色 | [查看](dark-iphone.png) | [查看](dark-after.png) |
| 最大字号 | [查看](large-iphone.png) | [查看](large-after.png) |

最大字号的标题**实际可用宽度是 iPhone 176.9pt、鸿蒙 177vp**。iPhone 控件树里标出的文字框宽 171.3pt，是排字后的内容宽度。两种框相减会得出约 17 个物理像素的表面差。关闭和更多的横向点击范围相同。

用两套系统实际随附的字体，在相同的字号、字重和列宽下，排字结果复现了这个字母的差异。省略号在 iPhone 字体中宽 50.33px，在鸿蒙字体中宽 69px。「the demo work…」分别需要 515.52px 和 559.26px，而可用空间约 531px。这里保留系统字体及完整无障碍名称。[原生坐标、逐图与控件树哈希及字体测量](title-font-metrics.json)
