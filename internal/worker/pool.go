package worker

type Task func()

type Pool struct {
	workerCount int
	taskQueue   chan Task
}

func NewPool(workerCount, queueSize int) *Pool {
	p := &Pool{
		workerCount: workerCount,
		taskQueue:   make(chan Task, queueSize),
	}

	for i := 0; i < workerCount; i++ {
		go p.startWorker()
	}
	return p
}

func (p *Pool) startWorker() {
	for task := range p.taskQueue {
		task() // Execute the task
	}
}

func (p *Pool) Submit(t Task) {
	p.taskQueue <- t
}
