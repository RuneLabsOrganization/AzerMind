"""Rebuild site images from the approved reference screenshot and concept art.

Example:
    python scripts/build-assets.py --screenshot path/to/azermind-screenshot.png --mockup path/to/approved-concept.png

The original images are not shipped; this helper is optional and not needed to run the site.
"""
import argparse
from pathlib import Path
from PIL import Image, ImageDraw, ImageEnhance, ImageFilter
import numpy as np

parser = argparse.ArgumentParser()
parser.add_argument('--screenshot', required=True, type=Path, help='Original 1074x735 screenshot from AzerMind')
parser.add_argument('--mockup', required=True, type=Path, help='Approved 1024x1536 landing page concept')
parser.add_argument('--output', type=Path, default=Path(__file__).resolve().parents[1] / 'assets')
args = parser.parse_args()
args.output.mkdir(parents=True, exist_ok=True)
source = Image.open(args.screenshot).convert('RGB')
image = source.copy()
patch = source.crop((494, 401, 684, 427))
mask = Image.new('L', patch.size, 0)
draw = ImageDraw.Draw(mask)
width, height = patch.size
for x in range(width):
    strength = min(1, x/15, (width-1-x)/15)
    draw.line([(x,0),(x,height)], fill=int(255*max(0,strength)))
image.paste(patch, (696,401), mask)
image.save(args.output/'addon-screenshot.png', optimize=True)
image.save(args.output/'addon-screenshot.webp','WEBP',quality=88,method=6)
source.crop((28,29,109,108)).save(args.output/'emblem.webp','WEBP',quality=95)
ImageEnhance.Contrast(source.crop((42,263,255,474))).enhance(.85).save(args.output/'leather.webp','WEBP',quality=78)
source.crop((523,445,780,495)).save(args.output/'parchment.webp','WEBP',quality=82)
mock = Image.open(args.mockup).convert('RGB')
w,h=1800,790
sky=Image.new('RGB',(w,h),(15,11,12))
sky.paste(mock.crop((0,66,280,414)).resize((840,h),Image.Resampling.LANCZOS),(0,0))
sky.paste(mock.crop((795,66,1024,414)).resize((750,h),Image.Resampling.LANCZOS),(1050,0))
a=np.asarray(sky).astype(np.float32)
x=np.arange(w,dtype=np.float32)
center_dark=np.exp(-((x-w*.54)/(w*.30))**4)
alpha=np.clip(1-center_dark*.94,.07,1)
alpha=np.where(x>775,np.minimum(alpha,np.interp(x,[775,1035],[.12,0])),alpha)
alpha=np.where(x>1035,np.maximum(alpha,np.interp(x,[1035,1285],[0,.52])),alpha)[None,:,None]
shadow=np.array([16.,11.,15.],dtype=np.float32)[None,None,:]
scenery=Image.fromarray(np.clip(a*alpha+shadow*(1-alpha),0,255).astype('uint8')).filter(ImageFilter.GaussianBlur(1.5))
scenery.save(args.output/'azeroth-backdrop.webp','WEBP',quality=83,method=6)
mock.crop((0,1240,1024,1429)).resize((1600,330),Image.Resampling.LANCZOS).filter(ImageFilter.GaussianBlur(1.7)).save(args.output/'final-landscape.webp','WEBP',quality=76,method=6)
print(f'Prepared assets in {args.output}')