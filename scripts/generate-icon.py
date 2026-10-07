#!/usr/bin/env python3
"""Generate opaque 180px calendar artwork using only the standard library."""
from pathlib import Path
import struct,zlib,math
size=180
bg=(40,99,78);fg=(247,249,239);accent=(201,221,169)
canvas=[[bg for _ in range(size)] for _ in range(size)]
def line(x1,y1,x2,y2,width,color):
 for y in range(size):
  for x in range(size):
   t=max(0,min(1,((x-x1)*(x2-x1)+(y-y1)*(y2-y1))/max(1,(x2-x1)**2+(y2-y1)**2)))
   if math.hypot(x-(x1+t*(x2-x1)),y-(y1+t*(y2-y1)))<=width/2:canvas[y][x]=color
for a in [(49,46,131,46),(140,55,140,133),(49,142,131,142),(40,55,40,133),(43,75,137,75),(66,36,66,61),(114,36,114,61)]:line(*a,9,fg)
for cx,cy,start in [(49,55,180),(131,55,270),(131,133,0),(49,133,90)]:
 for degrees in range(start,start+91):
  t=math.radians(degrees);x=cx+9*math.cos(t);y=cy+9*math.sin(t);line(x,y,x,y,9,fg)
line(68,109,82,122,10,accent);line(82,122,111,93,10,accent)
def chunk(kind,data):return struct.pack('!I',len(data))+kind+data+struct.pack('!I',zlib.crc32(kind+data)&0xffffffff)
raw=b''.join(b'\x00'+bytes(c for pixel in row for c in pixel) for row in canvas)
png=b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('!2I5B',size,size,8,2,0,0,0))+chunk(b'IDAT',zlib.compress(raw,9))+chunk(b'IEND',b'')
(Path(__file__).resolve().parents[1]/'web/public/apple-touch-icon.png').write_bytes(png)
