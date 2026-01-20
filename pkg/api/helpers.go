//nolint:revive // package name "api" is conventional for API type definitions
package api

import "time"

// timeNow is a variable that can be overridden in tests
var timeNow = time.Now
