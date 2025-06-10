package utils

type Queue struct {
	data []any
	head int
	tail int
	cap  int
}

func NewQueue(cap int) *Queue {
	return &Queue{
		data: make([]any, cap),
		cap:  cap,
		head: 0,
		tail: 0,
	}
}

func (q *Queue) Enqueue(value any) bool {
	if (q.tail+1)%q.cap == q.head {
		return false // Queue is full
	}
	q.data[q.tail] = value
	q.tail = (q.tail + 1) % q.cap
	return true
}

func (q *Queue) Dequeue() (any, bool) {
	if q.head == q.tail {
		return 0, false // Queue is empty
	}
	value := q.data[q.head]
	q.head = (q.head + 1) % q.cap
	return value, true
}

func (q *Queue) IsEmpty() bool {
	return q.head == q.tail
}

func (q *Queue) IsFull() bool {
	return (q.tail+1)%q.cap == q.head
}
