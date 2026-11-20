#import <AppKit/AppKit.h>
#import <Carbon/Carbon.h>
#import "bridge.h"
@class ThiefApp;
extern ThiefApp *Thief;
void Emit(NSString *kind,NSDictionary *payload);
NSColor *HexColor(NSString *hex,double transparency);
NSTextField *Label(NSString *text,CGFloat size);
NSButton *ThiefButton(NSString *title,id target,SEL action);
@interface ThiefApp : NSObject <NSApplicationDelegate,NSMenuDelegate>
@property(nonatomic,strong) NSStatusItem *status;
@property(nonatomic,strong) NSDictionary *snapshot;
@property(nonatomic,strong) NSWindow *manager;
@property(nonatomic,strong) NSTabView *tabs;
@property(nonatomic,strong) NSTableView *shelfTable;
@property(nonatomic,strong) NSTextField *bookSearch;
@property(nonatomic,strong) NSTextField *message;
@property(nonatomic,strong) NSArray *filteredBooks;
@property(nonatomic,strong) NSWindow *chapterWindow;
@property(nonatomic,strong) NSTableView *chapterTable;
@property(nonatomic,strong) NSTextField *chapterSearch;
@property(nonatomic,strong) NSTextField *chapterTitle;
@property(nonatomic,strong) NSArray *chapters;
@property(nonatomic,strong) NSArray *filteredChapters;
@property(nonatomic,copy) NSString *chapterBookID;
@property(nonatomic,strong) NSMutableDictionary *fields;
@property(nonatomic,strong) NSPanel *panel;
@property(nonatomic,strong) NSTextField *readerTitle;
@property(nonatomic,strong) NSTextField *readerProgress;
@property(nonatomic,strong) NSTextView *readerText;
@property(nonatomic,strong) NSButton *bubble;
@property(nonatomic,strong) NSRunningApplication *previousApp;
@property(nonatomic,strong) NSDictionary *registeredHotkeys;
@property(nonatomic,strong) NSMutableDictionary *hotkeyRefs;
@property(nonatomic,assign) BOOL applying;
@property(nonatomic,assign) BOOL moving;
@property(nonatomic,assign) BOOL interacting;
@property(nonatomic,assign) BOOL chapterInteraction;
@property(nonatomic,assign) uint64_t displayedRevision;
@property(nonatomic,assign) uint64_t handledReadAction;
@property(nonatomic,assign) uint64_t handledRequest;
@property(nonatomic,copy) NSDictionary *lastHotkeyAttempt;
@property(nonatomic,copy) NSString *hotkeyError;
- (void)applySnapshot:(NSDictionary *)snapshot;
- (void)action:(id)sender;
- (NSMenu *)readingMenu;
- (void)openManager:(id)sender;
@end
@interface ThiefApp (Manager) <NSTableViewDataSource,NSTableViewDelegate,NSTextFieldDelegate,NSWindowDelegate>
- (void)buildManager;
- (void)refreshManager;
- (void)showChapters:(NSDictionary *)payload;
- (void)endChapterInteraction;
@end
@interface ThiefApp (Reader)
- (void)buildReader;
- (void)refreshReader;
- (void)syncFrame;
@end
@interface ThiefApp (Hotkeys)
- (BOOL)registerHotkeys:(NSDictionary *)bindings atomic:(BOOL)atomic;
- (void)configureHotkeys;
@end
