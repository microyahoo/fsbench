package worker

import (
	"math/rand"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/microyahoo/fsbench/common"
)

// WorkItem is an interface for general work operations
// They can be read,write,list,delete or a stopper
type WorkItem interface {
	Prepare(conf *common.TestCaseConfiguration) error
	Do(conf *common.TestCaseConfiguration, index int) error
	Clean() error
}

type BaseOperation struct {
	TestName  string
	Directory string
	BlockSize uint64
	Size      uint64
}

// ReadOperation stands for a read operation
type ReadOperation struct {
	*BaseOperation
}

// WriteOperation stands for a write operation
type WriteOperation struct {
	*BaseOperation
}

// StatOperation stands for a stat operation
type StatOperation struct {
	*BaseOperation
}

// DeleteOperation stands for a delete operation
type DeleteOperation struct {
	*BaseOperation
}

// Stopper marks the end of a workqueue when using
// maxOps as testCase end criterium
type Stopper struct{}

// // KV is a simple key-value struct
// type KV struct {
// 	Key   common.OpType
// 	Value float64
// }

// WorkQueue contains the Queue and the valid operation's
// values to determine which operation should be done next
// in order to satisfy the set ratios.
type WorkQueue struct {
	// OperationValues []KV
	Queue []WorkItem
}

// Prepare prepares the execution of the ReadOperation
func (op *ReadOperation) Prepare(conf *common.TestCaseConfiguration) error {
	// log.WithField("bucket", op.Bucket).WithField("object", op.ObjectName).Debug("Preparing ReadOperation")
	return nil
}

// Prepare prepares the execution of the WriteOperation
func (op *WriteOperation) Prepare(conf *common.TestCaseConfiguration) error {
	log.WithField("dir", op.Directory).Debug("Preparing WriteOperation")

	start := time.Now()
	err := os.MkdirAll(op.Directory, 0755)
	duration := time.Since(start)

	promLatency.WithLabelValues(op.TestName, "MKDIR").Observe(float64(duration.Milliseconds()))
	if err != nil {
		promFailedOps.WithLabelValues(op.TestName, "MKDIR").Inc()
	} else {
		promFinishedOps.WithLabelValues(op.TestName, "MKDIR").Inc()
	}
	return err
}

// Prepare prepares the execution of the StatOperation
func (op *StatOperation) Prepare(conf *common.TestCaseConfiguration) error {
	// log.WithField("bucket", op.Bucket).WithField("object", op.ObjectName).Debug("Preparing StatOperation")
	// return putObject(housekeepingSvc, conf, op.BaseOperation)
	return nil
}

// Prepare prepares the execution of the DeleteOperation
func (op *DeleteOperation) Prepare(conf *common.TestCaseConfiguration) error {
	// log.WithField("bucket", op.Bucket).WithField("object", op.ObjectName).Debug("Preparing DeleteOperation")
	// check whether object is exist or not
	// _, err := headObject(housekeepingSvc, op.ObjectName, op.Bucket)
	// // object already exist
	// if err == nil {
	// 	return nil
	// }
	// return putObject(housekeepingSvc, conf, op.BaseOperation)
	return nil
}

// Prepare does nothing here
func (op *Stopper) Prepare(conf *common.TestCaseConfiguration) error {
	return nil
}

// Do executes the actual work of the ReadOperation
func (op *ReadOperation) Do(conf *common.TestCaseConfiguration, index int) error {
	// log.WithField("bucket", op.Bucket).WithField("object", op.ObjectName).Debug("Doing ReadOperation")
	start := time.Now()
	var err error
	// err := getObject(svc, conf, op.BaseOperation)
	duration := time.Since(start)
	promLatency.WithLabelValues(op.TestName, "GET").Observe(float64(duration.Milliseconds()))
	if err != nil {
		promFailedOps.WithLabelValues(op.TestName, "GET").Inc()
	} else {
		promFinishedOps.WithLabelValues(op.TestName, "GET").Inc()
	}
	// promDownloadedBytes.WithLabelValues(op.TestName, "GET").Add(float64(op.ObjectSize))
	return err
}

// Do executes the actual work of the WriteOperation
func (op *WriteOperation) Do(conf *common.TestCaseConfiguration, index int) error {
	// log.WithField("bucket", op.Bucket).WithField("object", op.ObjectName).Debug("Doing WriteOperation")
	var err error
	// err := putObject(svc, conf, op.BaseOperation)
	if err != nil {
		promFailedOps.WithLabelValues(op.TestName, "PUT").Inc()
	} else {
		promFinishedOps.WithLabelValues(op.TestName, "PUT").Inc()
	}
	// promUploadedBytes.WithLabelValues(op.TestName, "PUT").Add(float64(op.ObjectSize))
	return err
}

// Do executes the actual work of the StatOperation
func (op *StatOperation) Do(conf *common.TestCaseConfiguration, index int) error {
	// log.WithField("bucket", op.Bucket).WithField("object", op.ObjectName).Debug("Doing StatOperation")
	start := time.Now()
	var err error
	// _, err := listObjects(svc, op.ObjectName, op.Bucket)
	duration := time.Since(start)
	promLatency.WithLabelValues(op.TestName, "LIST").Observe(float64(duration.Milliseconds()))
	if err != nil {
		promFailedOps.WithLabelValues(op.TestName, "LIST").Inc()
	} else {
		promFinishedOps.WithLabelValues(op.TestName, "LIST").Inc()
	}
	return err
}

// Do executes the actual work of the DeleteOperation
func (op *DeleteOperation) Do(conf *common.TestCaseConfiguration, index int) error {
	// log.WithField("bucket", op.Bucket).WithField("object", op.ObjectName).Debug("Doing DeleteOperation")
	start := time.Now()
	var err error
	// err := deleteObject(svc, op.ObjectName, op.Bucket)
	duration := time.Since(start)
	promLatency.WithLabelValues(op.TestName, "DELETE").Observe(float64(duration.Milliseconds()))
	if err != nil {
		promFailedOps.WithLabelValues(op.TestName, "DELETE").Inc()
	} else {
		promFinishedOps.WithLabelValues(op.TestName, "DELETE").Inc()
	}
	return err
}

// Do does nothing here
func (op *Stopper) Do(conf *common.TestCaseConfiguration, index int) error {
	return nil
}

// Clean removes the objects and buckets left from the previous ReadOperation
func (op *ReadOperation) Clean() error {
	// log.WithField("bucket", op.Bucket).WithField("object", op.ObjectName).Debug("Cleaning up ReadOperation")
	// return deleteObject(housekeepingSvc, op.ObjectName, op.Bucket)
	return nil
}

// Clean removes the objects and buckets left from the previous WriteOperation
func (op *WriteOperation) Clean() error {
	// return deleteObject(housekeepingSvc, op.ObjectName, op.Bucket)
	return nil
}

// Clean removes the objects and buckets left from the previous StatOperation
func (op *StatOperation) Clean() error {
	// return deleteObject(housekeepingSvc, op.ObjectName, op.Bucket)
	return nil
}

// Clean removes the objects and buckets left from the previous DeleteOperation
func (op *DeleteOperation) Clean() error {
	return nil
}

// Clean does nothing here
func (op *Stopper) Clean() error {
	return nil
}

func generateBytes(payloadGenerator, testName string, size uint64) []byte {
	start := time.Now()
	data := make([]byte, size)
	switch payloadGenerator {
	case "empty": // https://github.com/dvassallo/s3-benchmark/blob/aebfe8e05c1553f35c16362b4ac388d891eee440/main.go#L290-L297
	// case "random":
	default:
		randASCIIBytes(data, rng)
	}
	duration := time.Since(start)
	promGenBytesLatency.WithLabelValues(testName).Observe(float64(duration.Milliseconds()))
	promGenBytesSize.WithLabelValues(testName).Set(float64(size))

	return data
}

// https://github.com/minio/warp/blob/master/pkg/generator/generator.go
const asciiLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890()"

var (
	asciiLetterBytes [len(asciiLetters)]byte
	rng              = rand.New(rand.NewSource(time.Now().UnixNano()))
	rngMutex         = &sync.Mutex{}
)

// randASCIIBytes fill destination with pseudorandom ASCII characters [a-ZA-Z0-9].
// Should never be considered for true random data generation.
func randASCIIBytes(dst []byte, rng *rand.Rand) {
	// unprotected access to custom rand.Rand objects can cause panics
	// https://github.com/golang/go/issues/3611
	rngMutex.Lock()
	// Use a single seed.
	v := rng.Uint64()
	rngMutex.Unlock()
	rnd := uint32(v)
	rnd2 := uint32(v >> 32)
	for i := range dst {
		dst[i] = asciiLetterBytes[int(rnd>>16)%len(asciiLetterBytes)]
		rnd ^= rnd2
		rnd *= 2654435761
	}
}

func init() {
	for i, v := range asciiLetters {
		asciiLetterBytes[i] = byte(v)
	}
}
