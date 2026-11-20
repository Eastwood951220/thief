#import "ui.h"
static NSDictionary *hotkeyKinds;
static OSStatus HotkeyHandler(EventHandlerCallRef next,EventRef event,void *user){EventHotKeyID id={0};OSStatus result=GetEventParameter(event,kEventParamDirectObject,typeEventHotKeyID,NULL,sizeof(id),NULL,&id);if(result!=noErr||id.signature!='thif')return eventNotHandledErr;NSString *kind=hotkeyKinds[@(id.id)];if(!kind)return eventNotHandledErr;Emit(kind,@{});return noErr;}
@implementation ThiefApp (Hotkeys)
- (BOOL)registerHotkeys:(NSDictionary *)bindings atomic:(BOOL)atomic {
 if(!bindings)return YES;static BOOL installed=NO;
 if(!installed){EventTypeSpec type={kEventClassKeyboard,kEventHotKeyPressed};OSStatus result=InstallEventHandler(GetEventDispatcherTarget(),&HotkeyHandler,1,&type,NULL,NULL);if(result!=noErr){self.hotkeyError=[NSString stringWithFormat:@"无法安装全局快捷键处理器（%d）",(int)result];return NO;}installed=YES;}
 if(!self.hotkeyRefs)self.hotkeyRefs=[NSMutableDictionary new];NSMutableDictionary *next=[NSMutableDictionary new],*actual=[NSMutableDictionary new],*kinds=[NSMutableDictionary new];static uint32_t index=100;NSMutableArray *errors=[NSMutableArray new];
 NSDictionary *names=@{@"next":@"下一页",@"previous":@"上一页",@"nextChapter":@"下一章",@"previousChapter":@"上一章",@"toggleHidden":@"隐藏 / 恢复",@"toggleMode":@"切换模式"};
 for(NSString *kind in [[bindings allKeys] sortedArrayUsingSelector:@selector(compare:)]){NSDictionary *b=bindings[kind];if([b[@"Modifiers"] unsignedIntValue]==0){actual[kind]=b;continue;}if([b isEqual:self.registeredHotkeys[kind]]&&self.hotkeyRefs[kind]){next[kind]=self.hotkeyRefs[kind];actual[kind]=b;continue;}EventHotKeyRef ref=NULL;EventHotKeyID id={'thif',index++};OSStatus result=RegisterEventHotKey([b[@"KeyCode"] unsignedIntValue],[b[@"Modifiers"] unsignedIntValue],id,GetEventDispatcherTarget(),kEventHotKeyExclusive,&ref);if(result!=noErr){[errors addObject:[NSString stringWithFormat:@"%@ 未启用（%d），请更换组合或关闭此快捷键",names[kind]?:kind,(int)result]];if(atomic)break;continue;}next[kind]=[NSValue valueWithPointer:ref];actual[kind]=b;kinds[@(id.id)]=kind;}
 self.hotkeyError=[errors componentsJoinedByString:@"；"];
 if(errors.count&&atomic){for(NSString *kind in next){if(next[kind]!=self.hotkeyRefs[kind])UnregisterEventHotKey([next[kind] pointerValue]);}return NO;}
 NSMutableDictionary *all=[hotkeyKinds mutableCopy]?:[NSMutableDictionary new];for(NSNumber *key in [all.allKeys copy]){NSString *kind=all[key];if(next[kind]!=self.hotkeyRefs[kind])[all removeObjectForKey:key];}[all addEntriesFromDictionary:kinds];for(NSString *kind in self.hotkeyRefs){if(next[kind]!=self.hotkeyRefs[kind])UnregisterEventHotKey([self.hotkeyRefs[kind] pointerValue]);}hotkeyKinds=all;self.hotkeyRefs=next;self.registeredHotkeys=actual;return errors.count==0;
}
- (void)configureHotkeys {
 id pending=self.snapshot[@"PendingSettings"];uint64_t request=[self.snapshot[@"RequestID"] unsignedLongLongValue];
 if([pending isKindOfClass:NSDictionary.class]){if(request!=self.handledRequest){self.handledRequest=request;NSDictionary *bindings=pending[@"Hotkeys"];BOOL success=[self registerHotkeys:bindings atomic:YES];self.lastHotkeyAttempt=bindings;Emit(success?@"settingsAccepted":@"settingsRejected",@{@"RequestID":@(request),@"Query":self.hotkeyError?:@""});}}
 else {NSDictionary *bindings=self.snapshot[@"Settings"][@"Hotkeys"];if(![bindings isEqual:self.lastHotkeyAttempt]){[self registerHotkeys:bindings atomic:NO];self.lastHotkeyAttempt=bindings;}}
 if(self.hotkeyError.length&&![self.snapshot[@"Error"] length]){self.message.textColor=NSColor.systemRedColor;self.message.stringValue=self.hotkeyError;}
}
@end
