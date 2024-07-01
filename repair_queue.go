package quic

import (
	"sync"

	"github.com/quic-go/quic-go/internal/utils/ringbuffer"
	"github.com/quic-go/quic-go/internal/wire"
)

// TODO (ddritzenhoff) do a find-f for datagram once you've added everything.

const (
	// TODO (ddritzenhoff) maybe make this based on the schema, somehow?
	maxRepairSendQueueLen = 32
)

type repairQueue struct {
	sendMx    sync.Mutex
	sendQueue ringbuffer.RingBuffer[*wire.RepairFrame]
	sent      chan struct{} // used to notify Add that a repair frame was dequeued

	// TODO (ddritzenhoff) I'm pretty sure I don't need a receive queue, as I'd immediately just add the repair from to the fecManager block.

	closeErr error
	closed   chan struct{}

	// hasData lets the main sending loop know there's more data in the send queue.
	hasData func()
}

func newRepairQueue(hasData func()) *repairQueue {
	return &repairQueue{
		hasData: hasData,
		sent:    make(chan struct{}, 1),
		closed:  make(chan struct{}),
	}
}

// Add queues a new REPAIR frame for sending.
// Up to 32 REPAIR frames will be queued.
// Once that limit is reached, Add blocks until the queue size has reduced.
func (h *repairQueue) Add(f *wire.RepairFrame) error {
	h.sendMx.Lock()

	for {
		if h.sendQueue.Len() < maxRepairSendQueueLen {
			h.sendQueue.PushBack(f)
			h.sendMx.Unlock()
			h.hasData()
			return nil
		}
		// TODO (ddritzenhoff) Must come up with better implementation.
		panic("repair queue full")

		/*
			The problem with the following code below is that it blocks until
			a spot opens up in the queue. Repair frames are unique in that they
			are created in the 'hot path', which means that blocking would completely
			brick the functionality of the system.
		*/
		// select {
		// case <-h.sent: // drain the queue so we don't loop immediately
		// default:
		// }
		// h.sendMx.Unlock()
		// select {
		// case <-h.closed:
		// 	return h.closeErr
		// case <-h.sent:
		// }
		// h.sendMx.Lock()
	}
}

// Peek gets the next REPAIR frame for sending.
// If actually sent out, Pop needs to be called before the next call to Peek.
func (h *repairQueue) Peek() *wire.RepairFrame {
	h.sendMx.Lock()
	defer h.sendMx.Unlock()
	if h.sendQueue.Empty() {
		return nil
	}
	return h.sendQueue.PeekFront()
}

func (h *repairQueue) Pop() {
	h.sendMx.Lock()
	defer h.sendMx.Unlock()
	_ = h.sendQueue.PopFront()
	select {
	case h.sent <- struct{}{}:
	default:
	}
}

func (h *repairQueue) CloseWithError(e error) {
	h.closeErr = e
	close(h.closed)
}
