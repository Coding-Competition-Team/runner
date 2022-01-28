package workers

import (
	"time"

	"runner/internal/api_portainer"
	"runner/internal/api_sql"
	"runner/internal/ds"
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
	log.Info("Current Instances:")
	it := ds.InstanceQueue.Iterator()
	current_timestamp := time.Now().UnixNano()

	for it.Next() { //Sorted by time
		timestamp, InstanceId := it.Key().(int64), it.Value().(int)
		if timestamp <= current_timestamp {
			api_sql.DeleteInstance(InstanceId)

			PortainerId := ds.InstanceMap[InstanceId].PortainerId
			if ds.ChallengeMap[ds.InstanceMap[InstanceId].ChallengeId].DockerCompose {
				api_portainer.DeleteStack(PortainerId)
			} else {
				api_portainer.DeleteContainer(PortainerId)
			}

			UserId := ds.InstanceMap[InstanceId].UserId
			ds.InstanceQueue.Remove(timestamp)
			for _, v := range ds.InstanceMap[InstanceId].Ports {
				delete(ds.UsedPorts, v)
			}
			delete(ds.InstanceMap, InstanceId)
			delete(ds.ActiveUserInstance, UserId)

			break //Only handle 1 instance a time to prevent tree iterator nonsense
		}
		log.Info(timestamp, ":", InstanceId)
	}
}