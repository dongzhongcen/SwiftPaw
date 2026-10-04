#!/usr/bin/env python3
"""根据 build/appicon.png 生成 Android 的启动图标（普通、圆形和自适应图标的前景）。

需要 Pillow：pip install pillow。在仓库根目录运行：python3 mobile/scripts/make-icons.py
图标改了以后重新运行一次，把生成的 png 一起提交。
"""
from pathlib import Path

from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parents[2]
SOURCE = ROOT / "build" / "appicon.png"
RES = ROOT / "mobile" / "android" / "app" / "src" / "main" / "res"

# 每种屏幕密度下的图标边长（像素）：启动图标 48dp，自适应图标的前景 108dp
DENSITIES = {"mdpi": 1, "hdpi": 1.5, "xhdpi": 2, "xxhdpi": 3, "xxxhdpi": 4}


def main():
    icon = Image.open(SOURCE).convert("RGB")
    for name, scale in DENSITIES.items():
        folder = RES / f"mipmap-{name}"
        folder.mkdir(parents=True, exist_ok=True)
        size = round(48 * scale)
        square = icon.resize((size, size), Image.LANCZOS)
        square.save(folder / "ic_launcher.png", optimize=True)

        # 圆形图标：圆以外透明
        mask = Image.new("L", (size * 4, size * 4), 0)
        ImageDraw.Draw(mask).ellipse((0, 0, size * 4 - 1, size * 4 - 1), fill=255)
        mask = mask.resize((size, size), Image.LANCZOS)
        round_icon = Image.new("RGBA", (size, size), (0, 0, 0, 0))
        round_icon.paste(square, (0, 0), mask)
        round_icon.save(folder / "ic_launcher_round.png", optimize=True)

        # 自适应图标的前景：108dp 的画布，图案放在中间 72dp 里（系统裁切时只保证中间 66dp 可见）
        canvas = round(108 * scale)
        inner = round(72 * scale)
        foreground = Image.new("RGB", (canvas, canvas), (0, 0, 0))
        offset = (canvas - inner) // 2
        foreground.paste(icon.resize((inner, inner), Image.LANCZOS), (offset, offset))
        foreground.save(folder / "ic_launcher_foreground.png", optimize=True)
    print("已生成启动图标")


if __name__ == "__main__":
    main()
