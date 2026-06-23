#import <Foundation/Foundation.h>
#import <LocalAuthentication/LocalAuthentication.h>
#import <dispatch/dispatch.h>
#import "touchid.h"

#include <string.h>

static void copy_error(char *buf, size_t len, NSString *msg) {
	if (buf == NULL || len == 0) {
		return;
	}
	buf[0] = '\0';
	if (msg == nil) {
		return;
	}
	const char *s = [msg UTF8String];
	if (s == NULL) {
		return;
	}
	strncpy(buf, s, len - 1);
	buf[len - 1] = '\0';
}

int ConfirmDeviceOwner(const char *reason, char *errbuf, size_t errbuf_len) {
	@autoreleasepool {
		LAContext *ctx = [[LAContext alloc] init];

		NSString *reasonString;
		if (reason != NULL) {
			reasonString = [NSString stringWithUTF8String:reason];
		} else {
			reasonString = @"Confirm this action";
		}

		NSError *canEvalError = nil;
		LAPolicy policy = LAPolicyDeviceOwnerAuthentication;

		if (![ctx canEvaluatePolicy:policy error:&canEvalError]) {
			copy_error(errbuf, errbuf_len, [canEvalError localizedDescription]);
			return 2;
		}

		dispatch_semaphore_t sem = dispatch_semaphore_create(0);
		__block BOOL ok = NO;

		[ctx evaluatePolicy:policy
		    localizedReason:reasonString
		              reply:^(BOOL success, NSError *evalError) {
			ok = success;
			if (!success && evalError != nil) {
				copy_error(errbuf, errbuf_len, [evalError localizedDescription]);
			}
			dispatch_semaphore_signal(sem);
		}];

		dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
		return ok ? 0 : 1;
	}
}
