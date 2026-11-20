#import "ui.h"
ThiefApp *Thief;
static NSUInteger received;
void Emit(NSString *kind,NSDictionary *payload){if([kind isEqual:@"next"])received++;}
@implementation ThiefApp
@end
int main(void){@autoreleasepool{
 [NSApplication sharedApplication];Thief=[ThiefApp new];
 // An uncommon key avoids taking any of the user's normal shortcuts.
 NSDictionary *bindings=@{@"next":@{@"KeyCode":@(kVK_F19),@"Modifiers":@(cmdKey|controlKey|optionKey|shiftKey)}};
 if(![Thief registerHotkeys:bindings atomic:YES]){fprintf(stderr,"registration failed: %s\n",Thief.hotkeyError.UTF8String);return 2;}
 // Dispatch where Cocoa receives system hotkey events, without synthesizing keys.
 for(UInt32 i=0;i<1000;i++){EventRef event=NULL;CreateEvent(NULL,kEventClassKeyboard,kEventHotKeyPressed,GetCurrentEventTime(),0,&event);EventHotKeyID id={'thif',i};SetEventParameter(event,kEventParamDirectObject,typeEventHotKeyID,sizeof(id),&id);SendEventToEventTarget(event,GetEventDispatcherTarget());ReleaseEvent(event);}
 for(NSValue *v in Thief.hotkeyRefs.allValues)UnregisterEventHotKey(v.pointerValue);
 if(received!=1){fprintf(stderr,"dispatcher delivered %lu hotkey actions, want 1\n",(unsigned long)received);return 1;}return 0;
}}
