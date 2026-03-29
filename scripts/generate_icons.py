#!/usr/bin/env python3
"""Generate multiple resolution icons from a source image for web favicons and app icons."""

import os
from PIL import Image

SOURCE = os.path.join(os.path.dirname(__file__), "..", "internal", "static", "images", "flaticon", "icon.png")
OUTPUT_DIR = os.path.join(os.path.dirname(__file__), "..", "internal", "static", "images", "flaticon")

# Web favicon / app icon sizes
SIZES = {
    "favicon-16x16.png": (16, 16),
    "favicon-32x32.png": (32, 32),
    "favicon-48x48.png": (48, 48),
    "apple-touch-icon.png": (180, 180),
    "icon-192x192.png": (192, 192),
    "icon-512x512.png": (512, 512),
}

def generate_icons():
    source = os.path.abspath(SOURCE)
    output_dir = os.path.abspath(OUTPUT_DIR)
    os.makedirs(output_dir, exist_ok=True)

    img = Image.open(source).convert("RGBA")
    print(f"Source image: {source} ({img.width}x{img.height})")

    for filename, size in SIZES.items():
        resized = img.resize(size, Image.LANCZOS)
        out_path = os.path.join(output_dir, filename)
        resized.save(out_path, "PNG")
        print(f"  Created {filename} ({size[0]}x{size[1]})")

    # Generate ICO file with multiple sizes embedded
    ico_sizes = [(16, 16), (32, 32), (48, 48)]
    ico_images = [img.resize(s, Image.LANCZOS) for s in ico_sizes]
    ico_path = os.path.join(output_dir, "favicon.ico")
    ico_images[0].save(ico_path, format="ICO", sizes=ico_sizes, append_images=ico_images[1:])
    print(f"  Created favicon.ico (multi-size: {', '.join(f'{s[0]}x{s[1]}' for s in ico_sizes)})")

    print("\nDone! All icons generated.")

if __name__ == "__main__":
    generate_icons()
