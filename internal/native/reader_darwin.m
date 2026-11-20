#import "ui.h"
@interface ReaderPanel:NSPanel @end
@implementation ReaderPanel
- (BOOL)canBecomeKeyWindow{return NO;}
- (BOOL)canBecomeMainWindow{return NO;}
@end
@interface ReaderText:NSTextView @end
@implementation ReaderText
- (NSView *)hitTest:(NSPoint)point{return nil;}
@end
@interface ReaderSurface:NSView
@property(nonatomic,strong) NSTrackingArea *tracking;
@property(nonatomic,assign) NSPoint dragStart;
@property(nonatomic,assign) NSRect original;
@property(nonatomic,assign) BOOL resizing;
@property(nonatomic,assign) BOOL collapsed;
@property(nonatomic,strong) NSColor *fill;
@end
@implementation ReaderSurface
- (void)drawRect:(NSRect)dirty {[(self.fill?:NSColor.clearColor) setFill];[[NSBezierPath bezierPathWithRoundedRect:self.bounds xRadius:9 yRadius:9] fill];if(!self.collapsed){[[NSColor.secondaryLabelColor colorWithAlphaComponent:.35] setStroke];NSBezierPath *p=[NSBezierPath bezierPath];[p moveToPoint:NSMakePoint(self.bounds.size.width-14,3)];[p lineToPoint:NSMakePoint(self.bounds.size.width-3,14)];[p stroke];}}
- (void)updateTrackingAreas {[super updateTrackingAreas];if(self.tracking)[self removeTrackingArea:self.tracking];self.tracking=[[NSTrackingArea alloc] initWithRect:self.bounds options:NSTrackingMouseEnteredAndExited|NSTrackingActiveAlways owner:self userInfo:nil];[self addTrackingArea:self.tracking];}
- (void)mouseEntered:(NSEvent *)event{Emit(@"hoverEnter",@{});}
- (void)mouseExited:(NSEvent *)event{if(!Thief.moving)Emit(@"hoverLeave",@{});}
- (void)mouseDown:(NSEvent *)event {self.dragStart=NSEvent.mouseLocation;self.original=self.window.frame;NSPoint local=[self convertPoint:event.locationInWindow fromView:nil];self.resizing=!self.collapsed&&local.x>self.bounds.size.width-20&&local.y<20;Thief.moving=YES;Emit(@"interactionBegin",@{});}
- (void)mouseDragged:(NSEvent *)event {NSPoint now=NSEvent.mouseLocation;CGFloat dx=now.x-self.dragStart.x,dy=now.y-self.dragStart.y;NSRect f=self.original;if(self.resizing){f.size.width=MAX(200,MIN(1600,f.size.width+dx));CGFloat h=MAX(120,MIN(1200,f.size.height-dy));f.origin.y+=f.size.height-h;f.size.height=h;}else{f.origin.x+=dx;f.origin.y+=dy;}[self.window setFrame:f display:YES];}
- (void)mouseUp:(NSEvent *)event {Thief.moving=NO;[Thief syncFrame];Emit(@"interactionEnd",@{});if(!NSPointInRect(NSEvent.mouseLocation,self.window.frame))Emit(@"hoverLeave",@{});}
- (void)rightMouseDown:(NSEvent *)event {[NSMenu popUpContextMenu:[Thief readingMenu] withEvent:event forView:self];}
@end
@implementation ThiefApp (Reader)
- (void)buildReader {self.panel=[[ReaderPanel alloc] initWithContentRect:NSMakeRect(100,100,355,236) styleMask:NSWindowStyleMaskBorderless|NSWindowStyleMaskNonactivatingPanel backing:NSBackingStoreBuffered defer:NO];self.panel.opaque=NO;self.panel.backgroundColor=NSColor.clearColor;self.panel.hasShadow=NO;self.panel.level=NSFloatingWindowLevel;self.panel.hidesOnDeactivate=NO;self.panel.collectionBehavior=NSWindowCollectionBehaviorCanJoinAllSpaces|NSWindowCollectionBehaviorFullScreenAuxiliary;ReaderSurface *surface=[[ReaderSurface alloc] initWithFrame:NSMakeRect(0,0,355,236)];self.panel.contentView=surface;
 self.readerTitle=Label(@"",11);self.readerTitle.frame=NSMakeRect(20,202,315,18);self.readerTitle.autoresizingMask=NSViewMinYMargin|NSViewWidthSizable;[surface addSubview:self.readerTitle];self.readerText=[[ReaderText alloc] initWithFrame:NSMakeRect(20,40,315,156)];self.readerText.editable=NO;self.readerText.selectable=NO;self.readerText.drawsBackground=NO;self.readerText.textContainerInset=NSZeroSize;self.readerText.textContainer.lineFragmentPadding=0;self.readerText.verticallyResizable=NO;self.readerText.horizontallyResizable=NO;self.readerText.autoresizingMask=NSViewWidthSizable|NSViewHeightSizable;[surface addSubview:self.readerText];self.readerProgress=Label(@"",10);self.readerProgress.frame=NSMakeRect(20,12,135,20);self.readerProgress.autoresizingMask=NSViewWidthSizable;[surface addSubview:self.readerProgress];
 for(NSArray *pair in @[@[@"‹",@"previous",@170],@[@"›",@"next",@220],@[@"目录",@"openChapters",@267]]){NSButton *b=ThiefButton(pair[0],self,@selector(readerAction:));b.identifier=pair[1];b.frame=NSMakeRect([pair[2] doubleValue],8,[pair[0] isEqual:@"目录"]?65:42,25);b.autoresizingMask=NSViewMinXMargin;[surface addSubview:b];}
 self.bubble=ThiefButton(@"≡",self,@selector(expandReader:));self.bubble.frame=NSMakeRect(0,0,38,38);self.bubble.bordered=NO;[surface addSubview:self.bubble];
 [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(screensChanged:) name:NSApplicationDidChangeScreenParametersNotification object:nil];
}
- (void)readerAction:(NSControl *)sender {if([sender.identifier isEqual:@"openChapters"]){Emit(@"chapters",@{@"BookID":self.snapshot[@"BookID"]?:@""});return;}Emit(sender.identifier,@{});}
- (void)expandReader:(id)sender {Emit(@"hoverEnter",@{});}
- (NSRect)clampFrame:(NSRect)frame {NSScreen *chosen=nil;for(NSScreen *screen in NSScreen.screens){if(NSIntersectsRect(frame,screen.visibleFrame)){chosen=screen;break;}}if(!chosen)chosen=NSScreen.mainScreen;NSRect bounds=chosen.visibleFrame;frame.size.width=MIN(frame.size.width,bounds.size.width);frame.size.height=MIN(frame.size.height,bounds.size.height);frame.origin.x=MAX(bounds.origin.x,MIN(frame.origin.x,NSMaxX(bounds)-frame.size.width));frame.origin.y=MAX(bounds.origin.y,MIN(frame.origin.y,NSMaxY(bounds)-frame.size.height));return frame;}
- (void)refreshReader {NSDictionary *s=self.snapshot[@"Settings"];if([self.snapshot[@"Hidden"] boolValue]||![s[@"Mode"] isEqual:@"float"]||![self.snapshot[@"BookID"] length]){[self.panel orderOut:nil];return;}NSDictionary *f=s[@"Frame"];NSRect desired=[self clampFrame:NSMakeRect([f[@"X"] doubleValue],[f[@"Y"] doubleValue],[f[@"Width"] doubleValue],[f[@"Height"] doubleValue])];BOOL collapsed=[self.snapshot[@"Collapsed"] boolValue];ReaderSurface *surface=(ReaderSurface *)self.panel.contentView;surface.collapsed=collapsed;if(collapsed){desired=NSMakeRect(NSMaxX(desired)-38,NSMaxY(desired)-38,38,38);}if(!self.moving&&!NSEqualRects(desired,self.panel.frame))[self.panel setFrame:desired display:YES];for(NSView *v in surface.subviews)v.hidden=collapsed;self.bubble.hidden=!collapsed;
 surface.fill=collapsed?[NSColor.windowBackgroundColor colorWithAlphaComponent:.9]:HexColor(s[@"BackgroundColor"],[s[@"BackgroundTransparency"] doubleValue]);[surface setNeedsDisplay:YES];if(!collapsed){NSDictionary *p=self.snapshot[@"Page"];NSColor *text=HexColor(s[@"TextColor"],[s[@"TextTransparency"] doubleValue]);self.readerTitle.stringValue=p[@"ChapterTitle"]?:@"";self.readerTitle.textColor=text;self.readerProgress.stringValue=[NSString stringWithFormat:@"位置 %@ – %@",p[@"Start"]?:@0,p[@"End"]?:@0];self.readerProgress.textColor=text;NSMutableParagraphStyle *style=[NSMutableParagraphStyle new];style.lineSpacing=[s[@"LineSpacing"] doubleValue];NSAttributedString *a=[[NSAttributedString alloc] initWithString:p[@"Text"]?:@"" attributes:@{NSFontAttributeName:[NSFont systemFontOfSize:[s[@"FontSize"] doubleValue]],NSForegroundColorAttributeName:text,NSParagraphStyleAttributeName:style}];[self.readerText.textStorage setAttributedString:a];self.readerText.textContainer.containerSize=NSMakeSize(self.readerText.frame.size.width,self.readerText.frame.size.height);}
 [self.panel orderFrontRegardless];
}
- (void)syncFrame {if(self.applying||[self.snapshot[@"Collapsed"] boolValue])return;NSRect f=[self clampFrame:self.panel.frame];[self.panel setFrame:f display:YES];Emit(@"setFrame",@{@"Frame":@{@"X":@(f.origin.x),@"Y":@(f.origin.y),@"Width":@(f.size.width),@"Height":@(f.size.height)}});}
- (void)screensChanged:(NSNotification *)note {if(self.panel.visible&&![self.snapshot[@"Collapsed"] boolValue])[self syncFrame];}
@end
