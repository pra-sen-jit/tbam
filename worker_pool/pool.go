package workerpool

type Task func()

type Pool struct {
	taskQueue chan Task
	workerCount int
}

func New(workerCount int, queueSize int) *Pool {
	p := &Pool{
		taskQueue: make(chan Task, queueSize),
		workerCount: workerCount,
	}


	for i := 1; i <= workerCount; i++ {
		go p.startWorker(i)
	}

	return p
}

func (p *Pool) startWorker(id int) {
	for task := range p.taskQueue {
		task() 
	}
}

func (p *Pool) Submit(t Task) {
	p.taskQueue <- t
}
