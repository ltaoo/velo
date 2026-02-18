#import <Cocoa/Cocoa.h>
#include "_cgo_export.h"

@interface TrayDelegate : NSObject <NSApplicationDelegate, NSMenuDelegate>
@property (strong) NSStatusItem *statusItem;
@property (strong) NSMutableDictionary *menuItems;
@end

static TrayDelegate *delegate;

@implementation TrayDelegate

- (void)applicationDidFinishLaunching:(NSNotification *)aNotification {
    // We create the status item here to ensure it's on the main thread and after app launch
    if (!self.statusItem) {
        self.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        self.menuItems = [[NSMutableDictionary alloc] init];
        
        NSMenu *menu = [[NSMenu alloc] init];
        [menu setAutoenablesItems:NO];
        [menu setDelegate:self];
        self.statusItem.menu = menu;
    }
}

- (void)onItemClick:(id)sender {
    NSMenuItem *item = (NSMenuItem *)sender;
    trayCallback((int)item.tag);
}

@end

void init_tray() {
    [NSApplication sharedApplication];
    delegate = [[TrayDelegate alloc] init];
    // We set the delegate but we also need to handle the case where the app is already running
    [NSApp setDelegate:delegate];
}

void ensure_tray_created() {
    if (!delegate.statusItem) {
         delegate.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
         delegate.menuItems = [[NSMutableDictionary alloc] init];
         NSMenu *menu = [[NSMenu alloc] init];
         [menu setAutoenablesItems:NO];
         delegate.statusItem.menu = menu;
    }
}

void set_icon(const char* data, int length, int isTemplate) {
    if (!delegate) return;
    
    NSData *nsData = [NSData dataWithBytes:data length:length];
    
    dispatch_async(dispatch_get_main_queue(), ^{
        ensure_tray_created();

        NSImage *image = [[NSImage alloc] initWithData:nsData];
        if (isTemplate) {
            [image setTemplate:YES];
        }
        delegate.statusItem.button.image = image;
    });
}

void set_title(const char* title) {
    if (!delegate) return;
    NSString *nsTitle = [NSString stringWithUTF8String:title];
    dispatch_async(dispatch_get_main_queue(), ^{
        ensure_tray_created();
        delegate.statusItem.button.title = nsTitle;
    });
}

void set_tooltip(const char* tooltip) {
    if (!delegate) return;
    NSString *nsTooltip = [NSString stringWithUTF8String:tooltip];
    dispatch_async(dispatch_get_main_queue(), ^{
        ensure_tray_created();
        delegate.statusItem.button.toolTip = nsTooltip;
    });
}

void add_menu_item(int id, const char* title, const char* shortcut, int disabled, int checked, int parentId, const char* imgData, int imgLen) {
    if (!delegate) return;
    NSString *nsTitle = [NSString stringWithUTF8String:title];
    NSString *nsShortcut = [NSString stringWithUTF8String:shortcut];
    NSData *nsImgData = (imgData && imgLen > 0) ? [NSData dataWithBytes:imgData length:imgLen] : nil;

    dispatch_async(dispatch_get_main_queue(), ^{
        ensure_tray_created();

        NSString *key = @"";
        NSEventModifierFlags mask = 0;

        if (nsShortcut.length > 0) {
            NSArray *parts = [nsShortcut componentsSeparatedByString:@"+"];
            if (parts.count > 0) {
                NSString *lastPart = [parts lastObject];
                key = lastPart.lowercaseString;
                for (NSString *part in parts) {
                    if ([part caseInsensitiveCompare:@"Cmd"] == NSOrderedSame || [part caseInsensitiveCompare:@"Command"] == NSOrderedSame) {
                        mask |= NSEventModifierFlagCommand;
                    } else if ([part caseInsensitiveCompare:@"Ctrl"] == NSOrderedSame || [part caseInsensitiveCompare:@"Control"] == NSOrderedSame) {
                        mask |= NSEventModifierFlagControl;
                    } else if ([part caseInsensitiveCompare:@"Alt"] == NSOrderedSame || [part caseInsensitiveCompare:@"Option"] == NSOrderedSame) {
                        mask |= NSEventModifierFlagOption;
                    } else if ([part caseInsensitiveCompare:@"Shift"] == NSOrderedSame) {
                        mask |= NSEventModifierFlagShift;
                    }
                }
            }
        }

        NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:nsTitle action:@selector(onItemClick:) keyEquivalent:key];
        if (mask > 0) {
            [item setKeyEquivalentModifierMask:mask];
        }

        item.tag = id;
        item.target = delegate;
        if (disabled) [item setEnabled:NO];
        if (checked) [item setState:NSControlStateValueOn];

        if (nsImgData) {
            NSImage *image = [[NSImage alloc] initWithData:nsImgData];
            if (image) {
                [image setSize:NSMakeSize(16, 16)];
                item.image = image;
            }
        }

        [delegate.menuItems setObject:item forKey:@(id)];

        NSMenu *parentMenu;
        if (parentId == 0) {
            parentMenu = delegate.statusItem.menu;
        } else {
            NSMenuItem *parentItem = [delegate.menuItems objectForKey:@(parentId)];
            if (!parentItem.submenu) {
                parentItem.submenu = [[NSMenu alloc] init];
                [parentItem.submenu setAutoenablesItems:NO];
            }
            parentMenu = parentItem.submenu;
        }
        [parentMenu addItem:item];
    });
}

void add_separator(int parentId) {
    if (!delegate) return;
    dispatch_async(dispatch_get_main_queue(), ^{
        ensure_tray_created();

        NSMenu *parentMenu;
        if (parentId == 0) {
            parentMenu = delegate.statusItem.menu;
        } else {
            NSMenuItem *parentItem = [delegate.menuItems objectForKey:@(parentId)];
            if (!parentItem.submenu) {
                parentItem.submenu = [[NSMenu alloc] init];
                 [parentItem.submenu setAutoenablesItems:NO];
            }
            parentMenu = parentItem.submenu;
        }
        [parentMenu addItem:[NSMenuItem separatorItem]];
    });
}

void run_loop() {
    [NSApp run];
}

void quit_app() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp terminate:nil];
    });
}

void set_item_label(int id, const char* label) {
    if (!delegate) return;
    NSString *nsLabel = [NSString stringWithUTF8String:label];
    dispatch_async(dispatch_get_main_queue(), ^{
        NSMenuItem *item = [delegate.menuItems objectForKey:@(id)];
        if (item) {
            item.title = nsLabel;
        }
    });
}

void set_item_tooltip(int id, const char* tooltip) {
    if (!delegate) return;
    NSString *nsTooltip = [NSString stringWithUTF8String:tooltip];
    dispatch_async(dispatch_get_main_queue(), ^{
        NSMenuItem *item = [delegate.menuItems objectForKey:@(id)];
        if (item) {
            item.toolTip = nsTooltip;
        }
    });
}

void set_item_checked(int id, int checked) {
    if (!delegate) return;
    dispatch_async(dispatch_get_main_queue(), ^{
        NSMenuItem *item = [delegate.menuItems objectForKey:@(id)];
        if (item) {
            [item setState:checked ? NSControlStateValueOn : NSControlStateValueOff];
        }
    });
}

void set_item_disabled(int id, int disabled) {
    if (!delegate) return;
    dispatch_async(dispatch_get_main_queue(), ^{
        NSMenuItem *item = [delegate.menuItems objectForKey:@(id)];
        if (item) {
            [item setEnabled:!disabled];
        }
    });
}
