package worker

import (
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/microyahoo/fsbench/common"
)

type Worker struct {
	workQueue       *WorkQueue
	parallelClients int
	op              common.OpType
	config          common.WorkerConf
}

// DoWork processes the workitems in the workChannel until
// either the time runs out or a stopper is found
func (w *Worker) DoWork(workChannel <-chan WorkItem, notifyChan <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-notifyChan:
			log.Debugf("Runtime over - Got timeout from work context")
			return
		case work := <-workChannel:
			switch work.(type) {
			case *Stopper:
				log.Debug("Found the end of the work Queue - stopping")
				return
			}
			err := work.Do(w.config.Test)
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
			w.op = config.Op
			w.workQueue = &WorkQueue{} // init or reset work queue
			log.Infof("Got config %+v for worker %s from server with reorder tasks(%t)- starting preparations now", config.Test, config.WorkerID, config.ReorderTasks)

			// w.initS3()
			w.fillWorkqueue()

			// if !config.Test.SkipPrepare {
			// 	for _, work := range w.workQueue.Queue {
			// 		e := work.Prepare(config.Test)
			// 		if e != nil {
			// 			log.WithError(e).Error("Error during work preparation - ignoring")
			// 		}
			// 	}
			// }
			log.Info("Preparations finished - waiting on server to start work")
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
			// benchResults.S3Endpoint = w.s3Endpoint
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
		// workerID    = w.config.WorkerID
		testConfig  = w.config.Test
		workChannel = make(chan WorkItem, len(w.workQueue.Queue))
		notifyChan  = make(chan struct{})
		startTime   = time.Now().UTC()
		wg          = &sync.WaitGroup{}
	)
	wg.Add(w.config.ParallelClients)

	promTestStart.WithLabelValues(testConfig.Name).Set(float64(startTime.UnixNano() / int64(1000000)))
	// promTestGauge.WithLabelValues(testConfig.Name).Inc()
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

func (w *Worker) workOneShot(workChannel chan WorkItem) {
	for _, work := range w.workQueue.Queue {
		workChannel <- work
	}
	for worker := 0; worker < w.parallelClients; worker++ {
		workChannel <- &Stopper{}
	}
}

func (w *Worker) fillWorkqueue() {
	// workerID := w.config.WorkerID
	// shareBucketName := w.config.Test.WorkerShareBuckets
	// testConfig := w.config.Test

	// if testConfig.ReadWeight > 0 {
	// 	w.workQueue.OperationValues = append(w.workQueue.OperationValues, KV{Key: "read"})
	// }
	// if testConfig.WriteWeight > 0 {
	// 	w.workQueue.OperationValues = append(w.workQueue.OperationValues, KV{Key: "write"})
	// }
	// if testConfig.ListWeight > 0 {
	// 	w.workQueue.OperationValues = append(w.workQueue.OperationValues, KV{Key: "list"})
	// }
	// if testConfig.DeleteWeight > 0 {
	// 	w.workQueue.OperationValues = append(w.workQueue.OperationValues, KV{Key: "delete"})
	// }

	// for bucketn := testConfig.Buckets.NumberMin; bucketn <= testConfig.Buckets.NumberMax; bucketn++ {
	// 	bucket := common.EvaluateDistribution(testConfig.Buckets.NumberMin, testConfig.Buckets.NumberMax, &testConfig.Buckets.NumberLast, 1, testConfig.Buckets.NumberDistribution)

	// 	bucketName := fmt.Sprintf("%s%s%d", workerID, testConfig.BucketPrefix, bucket)
	// 	if shareBucketName {
	// 		bucketName = fmt.Sprintf("%s%d", testConfig.BucketPrefix, bucket)
	// 	}
	// 	err := createBucket(housekeepingSvc, bucketName)
	// 	if err != nil {
	// 		log.WithError(err).WithField("bucket", bucketName).Error("Error when creating bucket")
	// 	}

	// 	for objectn := testConfig.Objects.NumberMin; objectn <= testConfig.Objects.NumberMax; objectn++ {
	// 		object := common.EvaluateDistribution(testConfig.Objects.NumberMin, testConfig.Objects.NumberMax, &testConfig.Objects.NumberLast, 1, testConfig.Objects.NumberDistribution)
	// 		objectSize := common.EvaluateDistribution(testConfig.Objects.SizeMin, testConfig.Objects.SizeMax, &testConfig.Objects.SizeLast, 1, testConfig.Objects.SizeDistribution)

	// 		nextOp := GetNextOperation(w.workQueue)
	// 		switch nextOp {
	// 		case "read":
	// 			err := IncreaseOperationValue(nextOp, 1/float64(testConfig.ReadWeight), w.workQueue)
	// 			if err != nil {
	// 				log.WithError(err).Error("Could not increase operational Value - ignoring")
	// 			}
	// 			new := &ReadOperation{
	// 				BaseOperation: &BaseOperation{
	// 					TestName:   testConfig.Name,
	// 					Bucket:     bucketName,
	// 					ObjectName: fmt.Sprintf("%s%s%d", workerID, testConfig.ObjectPrefix, object),
	// 					ObjectSize: objectSize,
	// 				},
	// 			}
	// 			w.workQueue.Queue = append(w.workQueue.Queue, new)
	// 		case "write":
	// 			err := IncreaseOperationValue(nextOp, 1/float64(testConfig.WriteWeight), w.workQueue)
	// 			if err != nil {
	// 				log.WithError(err).Error("Could not increase operational Value - ignoring")
	// 			}
	// 			new := &WriteOperation{
	// 				BaseOperation: &BaseOperation{
	// 					TestName:   testConfig.Name,
	// 					Bucket:     bucketName,
	// 					ObjectName: fmt.Sprintf("%s%s%d", workerID, testConfig.ObjectPrefix, object),
	// 					ObjectSize: objectSize,
	// 				},
	// 			}
	// 			w.workQueue.Queue = append(w.workQueue.Queue, new)
	// 		case "list":
	// 			err := IncreaseOperationValue(nextOp, 1/float64(testConfig.ListWeight), w.workQueue)
	// 			if err != nil {
	// 				log.WithError(err).Error("Could not increase operational Value - ignoring")
	// 			}
	// 			new := &ListOperation{
	// 				BaseOperation: &BaseOperation{
	// 					TestName:   testConfig.Name,
	// 					Bucket:     bucketName,
	// 					ObjectName: fmt.Sprintf("%s%s%d", workerID, testConfig.ObjectPrefix, object),
	// 					ObjectSize: objectSize,
	// 				},
	// 			}
	// 			w.workQueue.Queue = append(w.workQueue.Queue, new)
	// 		case "delete":
	// 			err := IncreaseOperationValue(nextOp, 1/float64(testConfig.DeleteWeight), w.workQueue)
	// 			if err != nil {
	// 				log.WithError(err).Error("Could not increase operational Value - ignoring")
	// 			}
	// 			new := &DeleteOperation{
	// 				BaseOperation: &BaseOperation{
	// 					TestName:   testConfig.Name,
	// 					Bucket:     bucketName,
	// 					ObjectName: fmt.Sprintf("%s%s%d", workerID, testConfig.ObjectPrefix, object),
	// 					ObjectSize: objectSize,
	// 				},
	// 			}
	// 			w.workQueue.Queue = append(w.workQueue.Queue, new)
	// 		}
	// 	}
	// }
}
