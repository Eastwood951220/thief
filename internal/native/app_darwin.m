#import <AppKit/AppKit.h>
#import "bridge.h"

@interface ThiefApp : NSObject <NSApplicationDelegate>
@property(nonatomic,strong) NSStatusItem *status;
@end
@implementation ThiefApp
- (void)applicationDidFinishLaunching:(NSNotification *)note {
 self.status=[[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
 self.status.button.title=@"≡";
 NSMenu *menu=[NSMenu new];
 [menu addItemWithTitle:@"退出 thief" action:@selector(terminate:) keyEquivalent:@"q"];
 self.status.menu=menu;
}
@end
static ThiefApp *delegate;
void thief_run(void) { @autoreleasepool { [NSApplication sharedApplication]; delegate=[ThiefApp new]; NSApp.delegate=delegate; [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory]; [NSApp run]; } }
void thief_apply(const char *json) { NSString *copy=[[NSString alloc] initWithUTF8String:json]; dispatch_async(dispatch_get_main_queue(), ^{ (void)copy; }); }
