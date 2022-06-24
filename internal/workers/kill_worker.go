package workers

import (
	"time"

	"runner/internal/api_portainer"
	"runner/internal/api_sql"
	"runner/internal/ds"
	"runner/internal/creds"
	"runner/internal/log"
)

//
// Worker Threads
// Source: https://bbengfort.github.io/2016/06/background-work-goroutines-timer/
//

// Worker will do its Action once every interval, making up for lost time that
// happened during the Action by only waiting the time left in the interval.
type Worker struct {
	Stopped         bool          // A flag determining the state of the worker
	ShutdownChannel chan string   // A channel to communicate to the routine
	Interval        time.Duration // The interval with which to run the Action
	period          time.Duration // The actual period of the wait
}

// NewWorker creates a new worker and instantiates all the data structures required.
func NewWorker(interval time.Duration) *Worker {
	return &Worker{
		Stopped:         false,
		ShutdownChannel: make(chan string),
		Interval:        interval,
		period:          interval,
	}
}

// Run starts the worker and listens for a shutdown call.
func (w *Worker) Run() {
	log.Info("Worker Started")
	// Loop that runs forever
	for {
		select {
		case <-w.ShutdownChannel:
			w.ShutdownChannel <- "Down"
			return
		case <-time.After(w.period):
			// This breaks out of the select, not the for loop.
			break
		}

		started := time.Now()
		w.Action()
		finished := time.Now()

		duration := finished.Sub(started)
		w.period = w.Interval - duration
	}
}

// Shutdown is a graceful shutdown mechanism
func (w *Worker) Shutdown() {
	w.Stopped = true
	w.ShutdownChannel <- "Down"
	<-w.ShutdownChannel
	close(w.ShutdownChannel)
}

// Action defines what the worker does; override this.
// For now we'll just wait two seconds and print to simulate work.
func (w *Worker) Action() {
	ClearInstanceQueue() //TODO: Make this async?
}

// Clears the current InstanceQueue for all instances with timestamp <= current_timestamp
func ClearInstanceQueue(){
	current_timestamp := time.Now().UnixNano()
	log.Info("Kill Worker", current_timestamp)

	for !ds.InstanceQueue.Empty() {
		it := ds.InstanceQueue.Iterator()
		it.Next()
		timestamp, InstanceId := it.Key().(int64), it.Value().(int)

		if timestamp > current_timestamp {
			break
		}
		
		log.Info("Clearing Instance", InstanceId)

		instance := api_sql.GetInstance(InstanceId) //Save a copy of the instance before it gets deleted
		api_sql.DeleteInstance(InstanceId)

		if api_sql.GetChallenge(instance.Challenge_Id).Docker_Compose {
			api_portainer.DeleteStack(instance.Portainer_Url, instance.Portainer_Id)
		} else {
			api_portainer.DeleteContainer(instance.Portainer_Url, instance.Portainer_Id)
		}

		creds.DecrementPortainerQueue(instance.Portainer_Url)

		ds.InstanceQueue.Remove(timestamp)
		for _, v := range api_sql.DeserializeI(instance.Ports_Used) {
			delete(ds.UsedPorts, v)
		}
	}
}