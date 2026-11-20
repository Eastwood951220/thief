#import <AppKit/AppKit.h>
#import <CoreText/CoreText.h>
#import "bridge.h"
long thief_fit(const char *utf8,double width,double height,double font,double spacing,int single){@autoreleasepool{
 NSString *text=[[NSString alloc] initWithUTF8String:utf8];if(!text||width<=0||height<=0)return 0;
 if(single)text=[text stringByReplacingOccurrencesOfString:@"\n" withString:@" "];
 NSMutableParagraphStyle *style=[NSMutableParagraphStyle new];style.lineSpacing=spacing;
 NSAttributedString *a=[[NSAttributedString alloc] initWithString:text attributes:@{NSFontAttributeName:[NSFont systemFontOfSize:font],NSParagraphStyleAttributeName:style}];
 CFIndex end=0;
 if(single){CTTypesetterRef t=CTTypesetterCreateWithAttributedString((__bridge CFAttributedStringRef)a);end=CTTypesetterSuggestClusterBreak(t,0,width);CFRelease(t);}
 else {CTFramesetterRef fs=CTFramesetterCreateWithAttributedString((__bridge CFAttributedStringRef)a);CGPathRef path=CGPathCreateWithRect(CGRectMake(0,0,width,height),NULL);CTFrameRef frame=CTFramesetterCreateFrame(fs,CFRangeMake(0,0),path,NULL);end=CTFrameGetVisibleStringRange(frame).length;CFRelease(frame);CGPathRelease(path);CFRelease(fs);}
 if(end>0&&end<text.length){NSRange r=[text rangeOfComposedCharacterSequenceAtIndex:end];if(r.location<end)end=r.location;}
 return end;
}}
