package constant

import "context"

var DoneServerCtx, DoneServer = context.WithCancel(context.Background())
