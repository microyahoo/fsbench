package worker

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/microyahoo/fsbench/pkg/common"
)

func TestWorker(t *testing.T) {
	suite.Run(t, new(workerSuite))
}

type workerSuite struct {
	suite.Suite
}

func (s *workerSuite) Test_populateDirs() {
	type args struct {
		// testcase *TestConf
		worker *Worker
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"Depth and width not specified", args{&Worker{
			config: common.WorkerConf{
				Test: &common.TestCaseConfiguration{
					FSD: &common.FSD{
						Anchor: "/volume",
					},
				},
			},
		}}, []string{"/volume"}},
		{"Depth and width specified", args{&Worker{
			config: common.WorkerConf{
				WorkerID: "w1",
				Test: &common.TestCaseConfiguration{
					FSD: &common.FSD{
						Depth:  4,
						Width:  2,
						Anchor: "/volume",
					},
				},
			},
		}}, []string{
			"/volume/w1.fsd.0_0/fsd.1_0/fsd.2_0/fsd.3_0",
			"/volume/w1.fsd.0_0/fsd.1_0/fsd.2_0/fsd.3_1",
			"/volume/w1.fsd.0_0/fsd.1_0/fsd.2_1/fsd.3_0",
			"/volume/w1.fsd.0_0/fsd.1_0/fsd.2_1/fsd.3_1",
			"/volume/w1.fsd.0_0/fsd.1_1/fsd.2_0/fsd.3_0",
			"/volume/w1.fsd.0_0/fsd.1_1/fsd.2_0/fsd.3_1",
			"/volume/w1.fsd.0_0/fsd.1_1/fsd.2_1/fsd.3_0",
			"/volume/w1.fsd.0_0/fsd.1_1/fsd.2_1/fsd.3_1",
			"/volume/w1.fsd.0_1/fsd.1_0/fsd.2_0/fsd.3_0",
			"/volume/w1.fsd.0_1/fsd.1_0/fsd.2_0/fsd.3_1",
			"/volume/w1.fsd.0_1/fsd.1_0/fsd.2_1/fsd.3_0",
			"/volume/w1.fsd.0_1/fsd.1_0/fsd.2_1/fsd.3_1",
			"/volume/w1.fsd.0_1/fsd.1_1/fsd.2_0/fsd.3_0",
			"/volume/w1.fsd.0_1/fsd.1_1/fsd.2_0/fsd.3_1",
			"/volume/w1.fsd.0_1/fsd.1_1/fsd.2_1/fsd.3_0",
			"/volume/w1.fsd.0_1/fsd.1_1/fsd.2_1/fsd.3_1",
		}},
	}
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			if dirs := tt.args.worker.populateDirs(); !s.EqualValues(tt.want, dirs) {
				t.Errorf("checkTestCase() want = %v, got = %v", tt.want, dirs)
			}
		})
	}
}
