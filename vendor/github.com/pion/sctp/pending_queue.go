package sctp

import (
	"fmt"
)

////////////////////////////////////////////////////////////////////////////////
// pendingBaseQueue

type pendingBaseQueue struct {
	queue []*chunkPayloadData
}

func newPendingBaseQueue() *pendingBaseQueue {
	return &pendingBaseQueue{queue: []*chunkPayloadData{}}
}

func (q *pendingBaseQueue) push(c *chunkPayloadData) {
	q.queue = append(q.queue, c)
}

func (q *pendingBaseQueue) pop() *chunkPayloadData {
	if len(q.queue) == 0 {
		return nil
	}
	c := q.queue[0]
	q.queue = q.queue[1:]
	return c
}

func (q *pendingBaseQueue) get(i int) *chunkPayloadData {
	if len(q.queue) == 0 || i < 0 || i >= len(q.queue) {
		return nil
	}
	return q.queue[i]
}

func (q *pendingBaseQueue) size() int {
	return len(q.queue)
}

////////////////////////////////////////////////////////////////////////////////
// pendingQueue

type pendingQueue struct {
	unorderedQueue *pendingBaseQueue
	orderedQueue   *pendingBaseQueue
	nBytes         int
}

func newPendingQueue() *pendingQueue {
	return &pendingQueue{
		unorderedQueue: newPendingBaseQueue(),
		orderedQueue:   newPendingBaseQueue(),
	}
}

func (q *pendingQueue) push(c *chunkPayloadData) {
	if c.unordered {
		q.unorderedQueue.push(c)
	} else {
		q.orderedQueue.push(c)
	}
	q.nBytes += len(c.userData)
}

func (q *pendingQueue) peek() *chunkPayloadData {
	if c := q.unorderedQueue.get(0); c != nil {
		return c
	}
	return q.orderedQueue.get(0)
}

func (q *pendingQueue) pop(c *chunkPayloadData) error {
	if c.unordered {
		popped := q.unorderedQueue.pop()
		if popped != c {
			return fmt.Errorf("unexped chunk popped (unordered)")
		}
	} else {
		popped := q.orderedQueue.pop()
		if popped != c {
			return fmt.Errorf("unexped chunk popped (ordered)")
		}
	}
	q.nBytes -= len(c.userData)
	return nil
}

func (q *pendingQueue) getNumBytes() int {
	return q.nBytes
}

func (q *pendingQueue) size() int {
	return q.unorderedQueue.size() + q.orderedQueue.size()
}
