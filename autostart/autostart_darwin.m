#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>
#import "autostart_darwin.h"

int enableLoginItem() {
    if (@available(macOS 13.0, *)) {
        SMAppService *service = [SMAppService mainAppService];
        NSError *error = nil;
        BOOL success = [service registerAndReturnError:&error];
        if (!success) {
            NSLog(@"Failed to register login item: %@", error);
            return 0;
        }
        return 1;
    }
    return 0;
}

int disableLoginItem() {
    if (@available(macOS 13.0, *)) {
        SMAppService *service = [SMAppService mainAppService];
        NSError *error = nil;
        BOOL success = [service unregisterAndReturnError:&error];
        if (!success) {
            NSLog(@"Failed to unregister login item: %@", error);
            return 0;
        }
        return 1;
    }
    return 0;
}

int isLoginItemEnabled() {
    if (@available(macOS 13.0, *)) {
        SMAppService *service = [SMAppService mainAppService];
        SMAppServiceStatus status = service.status;
        return (status == SMAppServiceStatusEnabled || status == SMAppServiceStatusRequiresApproval) ? 1 : 0;
    }
    return 0;
}
