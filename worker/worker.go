package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/microyahoo/fsbench/common"
)

type Worker struct {
	workQueue       *WorkQueue
	parallelClients int
	config          common.WorkerConf
}

type work struct {
	item  WorkItem
	index int // used for recording file index
}

// DoWork processes the workitems in the workChannel until
// either the time runs out or a stopper is found
func (w *Worker) DoWork(workChannel <-chan work, notifyChan <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-notifyChan:
			log.Debugf("Runtime over - Got timeout from work context")
			return
		case work := <-workChannel:
			switch work.item.(type) {
			case *Stopper:
				log.Debug("Found the end of the work Queue - stopping")
				return
			}
			err := work.item.Do(w.config.Test, work.index)
			if err != nil {
				log.WithError(err).Error("Issues when performing work - ignoring")
			}
		}
	}
}

func (w *Worker) ConnectToServer(serverAddress string) error {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	_ = encoder.Encode("ready for work")

	var response common.WorkerMessage
	for {
		err := decoder.Decode(&response)
		if err != nil {
			log.WithField("message", response).WithError(err).Error("Server responded unusually - reconnecting")
			return errors.New("Issue when receiving work from server")
		}
		log.Tracef("Response: %+v", response)
		switch response.Message {
		case "init":
			config := *response.Config
			w.config = config
			w.parallelClients = config.ParallelClients
			w.workQueue = &WorkQueue{} // init or reset work queue
			log.Infof("Got config %+v for worker %s from server with reorder tasks(%t)- starting preparations now", config.Test, config.WorkerID, config.ReorderTasks)

			w.fillWorkqueue()

			if w.config.Op == common.Write {
				// TODO: handle it concurrently
				for _, work := range w.workQueue.Queue {
					e := work.Prepare(config.Test)
					if e != nil {
						log.WithError(e).Error("Error during work preparation - ignoring")
					}
				}
			}
			log.Info("Preparations finished - waiting on server to start work")
			// TODO: Whether it is necessary to also collect and send the mkdir results to the server
			err = encoder.Encode(common.WorkerMessage{Message: "preparations done"})
			if err != nil {
				log.WithError(err).Error("Sending preparations done message to server- reconnecting")
				return errors.New("Issue when sending preparations done to server")
			}
		case "start work":
			if len(w.workQueue.Queue) == 0 {
				log.Warningf("Was instructed to start work - but the preparation step is incomplete - reconnecting")
				return nil
			}
			log.Info("Starting to work")
			duration := w.perfTest()
			benchResults := w.getCurrentPromValues()
			benchResults.Duration = duration
			benchResults.BandwidthAvg = benchResults.Bytes / duration.Seconds()
			log.Infof("PROM VALUES %+v", benchResults)
			err = encoder.Encode(common.WorkerMessage{Message: "work done", BenchResult: benchResults})
			if err != nil {
				log.WithError(err).Error("Sending work done message to server- reconnecting")
				return errors.New("Issue when sending work done to server")
			}
		case "shutdown":
			log.Info("Server told us to shut down - all work is done for today")
			return nil
		}
	}
}

// perfTest runs a performance test as configured in testConfig
func (w *Worker) perfTest() time.Duration {
	var (
		testConfig  = w.config.Test
		workChannel = make(chan work, w.parallelClients)
		notifyChan  = make(chan struct{})
		startTime   = time.Now().UTC()
		wg          = &sync.WaitGroup{}
	)
	wg.Add(w.config.ParallelClients)

	promTestStart.WithLabelValues(testConfig.Name).Set(float64(startTime.UnixNano() / int64(1000000)))
	for worker := 0; worker < w.parallelClients; worker++ {
		go w.DoWork(workChannel, notifyChan, wg)
	}
	log.Infof("Started %d parallel clients", w.parallelClients)
	w.workOneShot(workChannel)

	// Wait for all the goroutines to finish
	wg.Wait()

	log.Info("All clients finished")
	endTime := time.Now().UTC()
	promTestEnd.WithLabelValues(testConfig.Name).Set(float64(endTime.UnixNano() / int64(1000000)))

	return endTime.Sub(startTime)
}

func (w *Worker) workOneShot(workChannel chan work) {
	fsd := w.config.Test.FSD
	for _, w := range w.workQueue.Queue {
		for i := 0; i < int(fsd.Files); i++ {
			workChannel <- work{
				item:  w,
				index: i,
			}
		}
	}
	for worker := 0; worker < w.parallelClients; worker++ {
		workChannel <- work{&Stopper{}, -1}
	}
}

func (w *Worker) populateDirs() []string {
	return w.populateLevel(w.config.Test.FSD.Anchor, 0)
}

func (w *Worker) populateLevel(currentPath string, currentDepth uint64) []string {
	var dirs []string

	if currentDepth < w.config.Test.FSD.Depth {
		for i := uint64(0); i < w.config.Test.FSD.Width; i++ {
			subDir := fmt.Sprintf("fsd.%d_%d", currentDepth, i)
			if currentDepth == 0 {
				subDir = fmt.Sprintf("%s.fsd.%d_%d", w.config.WorkerID, currentDepth, i)
			}
			dirPath := filepath.Join(currentPath, subDir)
			dirs = append(dirs, w.populateLevel(dirPath, currentDepth+1)...)
		}
	} else {
		dirs = append(dirs, currentPath)
	}

	return dirs
}

func (w *Worker) fillWorkqueue() {
	testConfig := w.config.Test
	dirs := w.populateDirs()

	switch w.config.Op {
	case common.Write:
		for _, dir := range dirs {
			new := &WriteOperation{
				BaseOperation: &BaseOperation{
					TestName:  testConfig.Name,
					Directory: dir,
				},
			}
			w.workQueue.Queue = append(w.workQueue.Queue, new)
		}
	case common.Read:
		for _, dir := range dirs {
			new := &ReadOperation{
				BaseOperation: &BaseOperation{
					TestName:  testConfig.Name,
					Directory: dir,
				},
			}
			w.workQueue.Queue = append(w.workQueue.Queue, new)
		}
	case common.Stat:
		// TODO
	case common.Delete:
		// TODO
	default:
		panic(fmt.Sprintf("Invalid filesystem op: %s", w.config.Op))
	}
}
