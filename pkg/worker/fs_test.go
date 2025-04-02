package worker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestOSClient(t *testing.T) {
	suite.Run(t, new(osClientSuite))
}

type osClientSuite struct {
	suite.Suite
}

func (s *osClientSuite) Test_read_write() {
	type args struct {
		filename     string
		fsize, bsize int
	}
	tests := []struct {
		name          string
		args          args
		bcount, bsize int
		fsize         int
		fsizeActual   int
	}{
		{
			"fsize smaller than bsize",
			args{
				filename: "xx",
				fsize:    1023,
				bsize:    4096,
			},
			1,
			1023,
			1023,
			1023,
		},
		{
			"fsize larger than bsize",
			args{
				filename: "yy",
				fsize:    4097,
				bsize:    4096,
			},
			2,
			4096,
			8192,
			8192,
		},
	}
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "")
			s.NoError(err)
			defer os.RemoveAll(tmpDir)

			s.T().Log("--client--")
			client := NewOSClient(tmpDir, tt.args.fsize, tt.args.bsize)
			s.T().Log(client)
			s.Equal(tt.bcount, client.bcount)
			s.Equal(tt.bsize, client.bsize)
			s.T().Log("--write--")
			err = client.Write(tt.args.filename)
			s.NoError(err)
			s.T().Log("--read--")
			err = client.Read(tt.args.filename)
			s.NoError(err)
			fileInfo, err := os.Stat(filepath.Join(tmpDir, tt.args.filename))
			s.NoError(err)
			s.EqualValues(tt.fsize, fileInfo.Size())
		})
	}
}
