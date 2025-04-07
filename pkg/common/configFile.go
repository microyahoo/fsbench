package common

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"
)

type OpType int

const (
	Read OpType = 1 << iota
	// Mkdir
	Write
	Stat
	Delete
)

func (t OpType) String() string {
	var types []string
	if t&Read != 0 {
		types = append(types, "read")
	}
	// if t&Mkdir != 0 {
	// 	types = append(types, "mkdir")
	// }
	if t&Write != 0 {
		types = append(types, "write")
	}
	if t&Stat != 0 {
		types = append(types, "stat")
	}
	if t&Delete != 0 {
		types = append(types, "delete")
	}
	return strings.Join(types, ",")
}

func ToOpType(op string) OpType {
	switch op {
	case "write":
		return Write
	case "read":
		return Read
	// case "mkdir":
	// 	return Mkdir
	case "stat":
		return Stat
	case "delete":
		return Delete
	default:
		panic(fmt.Sprintf("invalid op: %s", op))
	}
}

// S3Configuration contains all information to connect to a certain S3 endpoint
type S3Configuration struct {
	AccessKey     string        `yaml:"access_key" json:"access_key"`
	SecretKey     string        `yaml:"secret_key" json:"secret_key"`
	Region        string        `yaml:"region" json:"region"`
	Endpoint      string        `yaml:"endpoint" json:"endpoint"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
	SkipSSLVerify bool          `yaml:"skipSSLverify" json:"skipSSLverify"`
	Name          string        `yaml:"name" json:"name"`
}
type fsdAlias struct {
	Anchor   string `yaml:"anchor"`
	DirectIO *bool  `yaml:"direct_io"`
	Depth    uint64 `yaml:"depth"`
	Width    uint64 `yaml:"width"`
	Files    uint64 `yaml:"files"`
	Size     string `yaml:"size"`
}

// filesystem define
type FSD struct {
	Anchor   string `yaml:"anchor" json:"anchor"`       // will override global config
	DirectIO *bool  `yaml:"direct_io" json:"direct_io"` // will override global config
	Depth    uint64 `yaml:"depth" json:"depth"`
	Width    uint64 `yaml:"width" json:"width"`
	Files    uint64 `yaml:"files" json:"files"` // total files = (width^depth)*files
	Size     uint64 `yaml:"size" json:"size"`   // 4k, 1m, 1g; total size = (total files) * size
}

func (s *FSD) UnmarshalJSON(data []byte) error {
	type Alias FSD
	aux := &struct {
		*Alias
		Size string `yaml:"size" json:"size"`
	}{Alias: (*Alias)(s)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.Depth < 1 {
		return fmt.Errorf("Minimum fs depth is 1")
	}
	size, err := ToBytes(aux.Size)
	if err != nil {
		return err
	}
	s.Size = size

	return nil
}

func (s *FSD) MarshalJSON() ([]byte, error) {
	type Alias FSD
	aux := &struct {
		*Alias
		Size string `yaml:"size" json:"size"`
	}{Alias: (*Alias)(s)}
	size := ByteSize(s.Size)
	aux.Size = size

	return json.Marshal(aux)
}

func (s *FSD) MarshalYAML() ([]byte, error) {
	size := ByteSize(s.Size)
	aux := &fsdAlias{
		Anchor:   s.Anchor,
		DirectIO: s.DirectIO,
		Depth:    s.Depth,
		Width:    s.Width,
		Files:    s.Files,
		Size:     size,
	}

	return yaml.Marshal(aux)
}

func (s *FSD) UnmarshalYAML(data []byte) error {
	var (
		aux fsdAlias
		err error
	)
	if err := yaml.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Depth < 1 {
		return fmt.Errorf("Minimum fs depth is 1")
	}
	size, err := ToBytes(aux.Size)
	if err != nil {
		return err
	}
	s.Size = size
	s.Anchor = aux.Anchor
	s.DirectIO = aux.DirectIO
	s.Depth = aux.Depth
	s.Width = aux.Width
	s.Files = aux.Files

	return nil
}

type fwdAlias struct {
	Operations []string `yaml:"operations" json:"operations"` // will override global config
	Threads    []uint64 `yaml:"threads" json:"threads"`       // will override global config
	BlockSize  string   `yaml:"block_size" json:"block_size"` // 4k, 1m, 1g
}

// filesystem workload define
type FWD struct {
	Operations []string `yaml:"operations" json:"operations"` // will override global config
	Threads    []uint64 `yaml:"threads" json:"threads"`       // will override global config
	BlockSize  uint64   `yaml:"block_size" json:"block_size"` // 4k, 1m, 4m
}

func (w *FWD) UnmarshalJSON(data []byte) error {
	type Alias FWD
	aux := &struct {
		*Alias
		BlockSize string `yaml:"block_size" json:"block_size"` // 4k, 1m, 1g
	}{Alias: (*Alias)(w)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	bs, err := ToBytes(aux.BlockSize)
	if err != nil {
		return err
	}
	w.BlockSize = bs

	return nil
}

func (w *FWD) MarshalJSON() ([]byte, error) {
	type Alias FWD
	aux := &struct {
		*Alias
		BlockSize string `yaml:"block_size" json:"block_size"` // 4k, 1m, 1g
	}{Alias: (*Alias)(w)}
	bs := ByteSize(w.BlockSize)
	aux.BlockSize = bs

	return json.Marshal(aux)
}

func (w *FWD) MarshalYAML() ([]byte, error) {
	bs := ByteSize(w.BlockSize)
	aux := &fwdAlias{
		BlockSize:  bs,
		Operations: w.Operations,
		Threads:    w.Threads,
	}

	return yaml.Marshal(aux)
}

func (w *FWD) UnmarshalYAML(data []byte) error {
	type Alias FWD
	var (
		aux fwdAlias
		err error
		bs  uint64
	)
	if err = yaml.Unmarshal(data, &aux); err != nil {
		return err
	}
	if bs, err = ToBytes(aux.BlockSize); err != nil {
		return err
	}
	w.BlockSize = bs
	w.Operations = aux.Operations
	w.Threads = aux.Threads

	return nil
}

// TestCaseConfiguration is the configuration of a performance test
type TestCaseConfiguration struct {
	// SkipPrepare      bool   `yaml:"skip_prepare" json:"skip_prepare"`
	Name             string `yaml:"name" json:"name"`
	FSD              *FSD   `yaml:"fsd" json:"fsd"`
	FWD              *FWD   `yaml:"fwd" json:"fwd"`
	PayloadGenerator string `yaml:"payload_generator" json:"payload_generator"` // empty or random
	ClearDirs        bool   `yaml:"clear_dirs" json:"clear_dirs"`
}

type ClientConfiguration struct {
	Name string `yaml:"name" json:"name"`
	IP   string `yaml:"ip" json:"ip"`
}

// TestConf contains all the information necessary to set up a distributed test
type TestConf struct {
	ReportConfig *ReportConfiguration     `yaml:"report_config" json:"report_config"`
	GlobalConfig *GlobalConfiguration     `yaml:"global_config" json:"global_config"`
	Tests        []*TestCaseConfiguration `yaml:"tests" json:"tests"`
}

type GlobalConfiguration struct {
	ReorderTasks bool     `yaml:"reorder_tasks" json:"reorder_tasks"` // having a different node read/stat back data than wrote it
	Anchor       string   `yaml:"anchor" json:"anchor"`
	DirectIO     *bool    `yaml:"direct_io" json:"direct_io"`
	Threads      []uint64 `yaml:"threads" json:"threads"`
	Operations   []string `yaml:"operations" json:"operations"`
	Workers      int      `yaml:"workers" json:"workers"`
}

type ReportConfiguration struct {
	Format   string           `yaml:"format" json:"format"` // md, csv or html
	Bucket   string           `yaml:"bucket" json:"bucket"` // report will upload to s3 bucket to persist
	S3Config *S3Configuration `yaml:"s3_config" json:"s3_config"`
}

// WorkerConf is the configuration that is sent to each worker
// It includes a subset of information from the Testconf
type WorkerConf struct {
	Test            *TestCaseConfiguration
	WorkerID        string
	ID              int
	Op              OpType // write, read, stat, delete
	ParallelClients int
}

// BenchResult is the struct that will contain the benchmark results from a
// worker after it has finished its benchmark
type BenchmarkResult struct {
	TestName             string
	SuccessfulOperations float64
	FailedOperations     float64
	Workers              float64
	ParallelClients      float64
	Bytes                float64
	// BandwidthAvg is the amount of Bytes per second of runtime
	BandwidthAvg       float64
	LatencyAvg         float64
	GenBytesLatencyAvg float64
	Duration           time.Duration
	Type               OpType
	FileSize           uint64
	Depth              uint64
	Width              uint64
	BlockSize          uint64
}

// WorkerMessage is the struct that is exchanged in the communication between
// server and worker. It usually only contains a message, but during the init
// phase, also contains the config for the worker
type WorkerMessage struct {
	Message     string
	Config      *WorkerConf
	BenchResult BenchmarkResult
}

// CheckSetConfig checks the global config
func CheckSetConfig(config *TestConf) {
	if config.GlobalConfig == nil { // TODO: check more configs
		log.WithError(fmt.Errorf("fs global configs need to be set")).Fatalf("Issue detected when scanning through the fs global configs")
	}
	if err := checkTestCase(config); err != nil {
		log.WithError(err).Fatalf("Issue detected when scanning through the config file")
	}
}

func checkTestCase(config *TestConf) error {
	if len(config.Tests) == 0 {
		return fmt.Errorf("Filesystem test cases needs to be set")
	}
	for _, testcase := range config.Tests {
		if testcase.PayloadGenerator != "" && testcase.PayloadGenerator != "random" && testcase.PayloadGenerator != "empty" {
			return fmt.Errorf("Either random or empty needs to be set for payload generator")
		}
		if testcase.FSD == nil || testcase.FWD == nil {
			return fmt.Errorf("Filesystem define and workload define needs to be set")
		}
		if config.GlobalConfig.Anchor == "" && testcase.FSD.Anchor == "" {
			return fmt.Errorf("Filesystem anchor needs to be set")
		}
		if config.GlobalConfig.DirectIO == nil && testcase.FSD.DirectIO == nil {
			return fmt.Errorf("Filesystem direct_io needs to be set")
		}
		if len(config.GlobalConfig.Operations) == 0 && len(testcase.FWD.Operations) == 0 {
			return fmt.Errorf("Filesystem workload operations needs to be set")
		}
		if len(config.GlobalConfig.Threads) == 0 && len(testcase.FWD.Threads) == 0 {
			return fmt.Errorf("Filesystem workload threads needs to be set")
		}
		if testcase.FSD.Anchor == "" {
			testcase.FSD.Anchor = config.GlobalConfig.Anchor
		}
		if testcase.FSD.DirectIO == nil {
			testcase.FSD.DirectIO = config.GlobalConfig.DirectIO
		}
		if len(testcase.FWD.Operations) == 0 {
			testcase.FWD.Operations = config.GlobalConfig.Operations
		}
		if len(testcase.FWD.Threads) == 0 {
			testcase.FWD.Threads = config.GlobalConfig.Threads
		}
	}
	return nil
}

var ReadFile = os.ReadFile

func LoadConfigFromFile(configFile string) *TestConf {
	configFileContent, err := ReadFile(configFile)
	if err != nil {
		log.WithError(err).Fatalf("Error reading config file: %s", configFile)
	}
	var config TestConf

	if strings.HasSuffix(configFile, ".yaml") || strings.HasSuffix(configFile, ".yml") {
		err = yaml.Unmarshal(configFileContent, &config)
		if err != nil {
			log.WithError(err).Fatalf("Error unmarshaling yaml config file: %s", configFile)
		}
	} else if strings.HasSuffix(configFile, ".json") {
		err = json.Unmarshal(configFileContent, &config)
		if err != nil {
			log.WithError(err).Fatalf("Error unmarshaling json config file: %s", configFile)
		}
	} else {
		log.WithError(err).Fatalf("Configuration file must be a yaml or json formatted file")
	}

	return &config
}
