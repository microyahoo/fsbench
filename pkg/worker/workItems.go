package worker

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/microyahoo/fsbench/pkg/common"
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
}

func (b *BaseOperation) fileName(index int) string {
	return fmt.Sprintf("bat.%d", index)
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

// WorkQueue contains the Queue and the valid operation's
// values to determine which operation should be done next
// in order to satisfy the set ratios.
type WorkQueue struct {
	Queue []WorkItem
}

// Prepare prepares the execution of the ReadOperation
func (op *ReadOperation) Prepare(conf *common.TestCaseConfiguration) error {
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
	return nil
}

// Prepare prepares the execution of the DeleteOperation
func (op *DeleteOperation) Prepare(conf *common.TestCaseConfiguration) error {
	return nil
}

// Prepare does nothing here
func (op *Stopper) Prepare(conf *common.TestCaseConfiguration) error {
	return nil
}

// Do executes the actual work of the ReadOperation
func (op *ReadOperation) Do(conf *common.TestCaseConfiguration, index int) error {
	fileName := op.fileName(index)
	log.WithField("dir", op.Directory).WithField("filename", fileName).Debug("Doing ReadOperation")
	var (
		err    error
		direct = *conf.FSD.DirectIO
		fsize  = conf.FSD.Size
		bsize  = conf.FWD.BlockSize
		start  = time.Now()
	)
	c := NewOSClient(op.Directory, int(fsize), int(bsize),
		WithDirectIO(direct),
		WithPayloadGenerator(conf.PayloadGenerator),
		WithTestName(conf.Name))
	err = c.Read(fileName)

	duration := time.Since(start)
	promLatency.WithLabelValues(op.TestName, "READ").Observe(float64(duration.Milliseconds()))
	if err != nil {
		promFailedOps.WithLabelValues(op.TestName, "READ").Inc()
	} else {
		promFinishedOps.WithLabelValues(op.TestName, "READ").Inc()
	}
	promDownloadedBytes.WithLabelValues(op.TestName, "READ").Add(float64(fsize))
	return err
}

// Do executes the actual work of the WriteOperation
func (op *WriteOperation) Do(conf *common.TestCaseConfiguration, index int) error {
	fileName := op.fileName(index)
	log.WithField("dir", op.Directory).WithField("filename", fileName).Debug("Doing WriteOperation")
	var (
		err    error
		direct = *conf.FSD.DirectIO
		fsize  = conf.FSD.Size
		bsize  = conf.FWD.BlockSize
		start  = time.Now()
	)
	c := NewOSClient(op.Directory, int(fsize), int(bsize),
		WithDirectIO(direct),
		WithPayloadGenerator(conf.PayloadGenerator),
		WithTestName(conf.Name))
	err = c.Write(fileName)

	duration := time.Since(start)
	promLatency.WithLabelValues(op.TestName, "WRITE").Observe(float64(duration.Milliseconds()))
	if err != nil {
		promFailedOps.WithLabelValues(op.TestName, "WRITE").Inc()
	} else {
		promFinishedOps.WithLabelValues(op.TestName, "WRITE").Inc()
	}
	promUploadedBytes.WithLabelValues(op.TestName, "WRITE").Add(float64(fsize))

	return err
}

// Do executes the actual work of the StatOperation
func (op *StatOperation) Do(conf *common.TestCaseConfiguration, index int) error {
	fileName := op.fileName(index)
	log.WithField("dir", op.Directory).WithField("filename", fileName).Debug("Doing StatOperation")
	var (
		err   error
		fsize = conf.FSD.Size
		bsize = conf.FWD.BlockSize
		start = time.Now()
	)
	c := NewOSClient(op.Directory, int(fsize), int(bsize))
	_, err = c.Stat(fileName)

	duration := time.Since(start)
	promLatency.WithLabelValues(op.TestName, "STAT").Observe(float64(duration.Milliseconds()))
	if err != nil {
		promFailedOps.WithLabelValues(op.TestName, "STAT").Inc()
	} else {
		promFinishedOps.WithLabelValues(op.TestName, "STAT").Inc()
	}
	return err
}

// Do executes the actual work of the DeleteOperation
func (op *DeleteOperation) Do(conf *common.TestCaseConfiguration, index int) error {
	fileName := op.fileName(index)
	log.WithField("dir", op.Directory).WithField("filename", fileName).Debug("Doing DeleteOperation")
	var (
		err   error
		fsize = conf.FSD.Size
		bsize = conf.FWD.BlockSize
		start = time.Now()
	)
	c := NewOSClient(op.Directory, int(fsize), int(bsize))
	err = c.Delete(fileName)

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
	return nil
}

// Clean removes the objects and buckets left from the previous WriteOperation
func (op *WriteOperation) Clean() error {
	return nil
}

// Clean removes the objects and buckets left from the previous StatOperation
func (op *StatOperation) Clean() error {
	return nil
}

// Clean removes the objects and buckets left from the previous DeleteOperation
func (op *DeleteOperation) Clean() error {
	log.WithField("dir", op.Directory).Debug("Doing DeleteOperation clean")
	return os.Remove(op.Directory)
}

// Clean does nothing here
func (op *Stopper) Clean() error {
	return nil
}
