#pragma once
#include <stddef.h>

// Returns 0 on success, non-zero on failure.
int ConfirmDeviceOwner(const char *reason, char *errbuf, size_t errbuf_len);
