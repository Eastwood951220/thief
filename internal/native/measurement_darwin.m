#import <AppKit/AppKit.h>
#import <CoreText/CoreText.h>
#import "bridge.h"
long thief_fit(const char *utf8,long length,double width,double height,double font,double spacing,int single){@autoreleasepool{
 NSString *text=[[NSString alloc] initWithBytes:utf8 length:length encoding:NSUTF8StringEncoding];if(!text||width<=0||height<=0)return 0;
 if(single)text=[text stringByReplacingOccurrencesOfString:@"\n" withString:@" "];
 NSMutableParagraphStyle *style=[NSMutableParagraphStyle new];style.lineSpacing=spacing;
 NSAttributedString *a=[[NSAttributedString alloc] initWithString:text attributes:@{NSFontAttributeName:[NSFont systemFontOfSize:font],NSParagraphStyleAttributeName:style}];
 CFIndex end=0;
 if(single){CTTypesetterRef t=CTTypesetterCreateWithAttributedString((__bridge CFAttributedStringRef)a);end=CTTypesetterSuggestClusterBreak(t,0,width);CFRelease(t);}
 else {NSTextStorage *storage=[[NSTextStorage alloc] initWithAttributedString:a];NSLayoutManager *layout=[NSLayoutManager new];NSTextContainer *container=[[NSTextContainer alloc] initWithContainerSize:NSMakeSize(width,height)];container.lineFragmentPadding=0;[layout addTextContainer:container];[storage addLayoutManager:layout];NSRange glyphs=[layout glyphRangeForTextContainer:container];__block NSUInteger visible=0;[layout enumerateLineFragmentsForGlyphRange:glyphs usingBlock:^(NSRect rect,NSRect used,NSTextContainer *c,NSRange range,BOOL *stop){if(NSMaxY(used)>height+.01){*stop=YES;return;}visible=NSMaxRange(range);}];end=NSMaxRange([layout characterRangeForGlyphRange:NSMakeRange(0,visible) actualGlyphRange:NULL]);}
 if(end>0&&end<text.length){NSRange r=[text rangeOfComposedCharacterSequenceAtIndex:end];if(r.location<end)end=r.location;}
 return end;
}}
