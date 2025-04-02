package worker

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ncw/directio"
)

// https://github.com/minio/warp/blob/master/pkg/generator/generator.go
const asciiLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890()"

var (
	asciiLetterBytes [len(asciiLetters)]byte
	rng              = rand.New(rand.NewSource(time.Now().UnixNano()))
	rngMutex         = &sync.Mutex{}
)

// OSClient a filesystem based on OSClient
type OSClient struct {
	direct           bool   // direct io
	rootDir          string // root directory
	fsize, bsize     int    // file/block size in bytes
	bcount           int    // block count
	payloadGenerator string // random or empty
	testName         string // test name
}

type OSClientOption func(*OSClient)

func WithDirectIO(direct bool) OSClientOption {
	return func(c *OSClient) {
		c.direct = direct
	}
}

func WithTestName(testName string) OSClientOption {
	return func(c *OSClient) {
		c.testName = testName
	}
}

func WithPayloadGenerator(generator string) OSClientOption {
	return func(c *OSClient) {
		c.payloadGenerator = generator
	}
}

// NewOSClient returns a new OSClient
func NewOSClient(rootDir string, fsize, bsize int, options ...OSClientOption) *OSClient {
	c := &OSClient{rootDir: rootDir, fsize: fsize, bsize: bsize}
	for _, option := range options {
		option(c)
	}
	if fsize <= bsize {
		c.bcount = 1
		c.bsize = fsize
	} else {
		c.bcount = (fsize-1)/bsize + 1
		c.fsize = c.bcount * bsize
	}
	return c
}

// Create creates a new OS File
func (c *OSClient) Create(filename string) (*os.File, error) {
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	return c.open(filename, flags)
}

// Open opens a new OS File
func (c *OSClient) Open(filename string) (*os.File, error) {
	flags := os.O_RDONLY
	return c.open(filename, flags)
}

func (c *OSClient) String() string {
	return fmt.Sprintf("os client root dir: %s, bcount: %d, bsize: %d, fsize: %d", c.rootDir, c.bcount, c.bsize, c.fsize)
}

func (c *OSClient) open(filename string, flags int) (*os.File, error) {
	fullpath := filepath.Join(c.rootDir, filename)
	err := os.MkdirAll(filepath.Dir(fullpath), 0755)
	if err != nil {
		return nil, err
	}

	var f *os.File
	if c.direct {
		f, err = directio.OpenFile(fullpath, flags, 0666)
	} else {
		f, err = os.OpenFile(fullpath, flags, 0644)
	}

	if err != nil {
		return nil, err
	}

	return f, nil
}

func (c *OSClient) Delete(filename string) error {
	fullpath := filepath.Join(c.rootDir, filename)

	return os.Remove(fullpath)
}

func (c *OSClient) Stat(filename string) (os.FileInfo, error) {
	fullpath := filepath.Join(c.rootDir, filename)

	return os.Stat(fullpath)
}

func (c *OSClient) Write(filename string) error {
	file, err := c.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	block := directio.AlignedBlock(int(c.bsize))
	for i := 0; i < c.bcount; i++ {
		generateBytes(c.payloadGenerator, c.testName, block)
		if _, err = file.Write(block); err != nil {
			return err
		}
	}

	return nil
}

func (c *OSClient) Read(filename string) error {
	file, err := c.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	block := directio.AlignedBlock(c.bsize)
	for i := 0; i < c.bcount; i++ {
		if n, err := file.Read(block); err != nil {
			return err
		} else if n != c.bsize {
			return fmt.Errorf("Failed to read file %s, expected: %d, actual: %d", filename, c.bsize, n)
		}
	}

	return nil
}

func generateBytes(payloadGenerator, testName string, data []byte) {
	start := time.Now()
	switch payloadGenerator {
	case "empty": // https://github.com/dvassallo/s3-benchmark/blob/aebfe8e05c1553f35c16362b4ac388d891eee440/main.go#L290-L297
	// case "random":
	default:
		randASCIIBytes(data, rng)
	}
	duration := time.Since(start)
	promGenBytesLatency.WithLabelValues(testName).Observe(float64(duration.Milliseconds()))
	promGenBytesSize.WithLabelValues(testName).Set(float64(len(data)))
}

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
