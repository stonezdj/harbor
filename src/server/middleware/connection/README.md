## Test code to track active connections in Harbor registry

This middleware tracks the number of active connections to the Harbor registry. It increments a counter when a new connection is established and decrements it when the connection is closed. The current count of active connections can be accessed via the `ConnectionCount` field in the system information API.

### Usage

1. Apply the middleware
2. Monitor the active connection count through tracker/watch_connection
