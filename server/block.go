package server

import "net"

type BlockingManager struct {
	cmdQueue map[CmdName][]*BlockedClient
}

/* NOTE: We don't need an ID for this struct because we're using it as
* a pointer, ergo we can do checks against the memory address to determine
* when to pop a particular BlockedClient from the above KeyQueue.
 */
type BlockedClient struct {
	conn net.Conn
	subs []CmdName
}

func (bm *BlockingManager) UnregisterClient(bc *BlockedClient) {
	for _, cmds := range bc.subs {
		bmCmdQueue := bm.cmdQueue[cmds]

		i := 0
		for _, c := range bmCmdQueue {
			if bmCmdQueue[i] != bc {
				bmCmdQueue[i] = c
				i++
			}
		}

		bmCmdQueue = bmCmdQueue[:i]
	}
}
