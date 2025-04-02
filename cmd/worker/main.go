package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/microyahoo/fsbench/pkg/worker"
)

const (
	fsbenchInContainer = "FSBENCH_IN_CONTAINER"
)

var (
	prometheusPort int
	debug, trace   bool
	serverAddress  string
)

func main() {
	rootCmd := newCommand()
	cobra.CheckErr(rootCmd.Execute())
}

func newCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use: "fsbench-worker",
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
	cmds.Flags().SortFlags = false

	viper.SetDefault("DEBUG", false)
	viper.SetDefault("TRACE", false)
	viper.SetDefault("PROMETHEUSPORT", 8888)

	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)

	cmds.Flags().BoolVar(&trace, "trace", viper.GetBool("TRACE"), "enable trace log output")
	cmds.Flags().BoolVar(&debug, "debug", viper.GetBool("DEBUG"), "enable debug log output")
	cmds.Flags().StringVar(&serverAddress, "server.address", viper.GetString("SERVERADDRESS"), "Fsbench Server IP and Port in the form '191.168.1.1:2000'")
	cmds.Flags().IntVar(&prometheusPort, "prometheus.port", viper.GetInt("PROMETHEUSPORT"), "Port on which the Prometheus Exporter will be available. Default: 8888")

	// cmds.MarkFlagRequired("server.address")

	return cmds
}

func run() {
	if serverAddress == "" {
		log.Fatal("--server.address is a mandatory parameter - please specify the server IP and Port")
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	} else if trace {
		log.SetLevel(log.TraceLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.Debugf("viper settings=%+v", viper.AllSettings())
	log.Debugf("fsbench worker serverAddress=%s, prometheusPort=%d", serverAddress, prometheusPort)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", worker.NewHandler())
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html>
             <head><title>Fsbench Exporter</title></head>
             <body>
             <h1>Fsbench Exporter</h1>
             <p><a href="` + `/metrics` + `">Metrics</a></p>
             </body>
             </html>`))
		})
		// http://localhost:8888/metrics
		log.Infof("Starting Prometheus Exporter on port %d", prometheusPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", prometheusPort), mux); err != nil {
			log.WithError(err).Fatalf("Failed to run Prometheus /metrics endpoint:")
		}
	}()

	worker := &worker.Worker{}
	for {
		err := worker.ConnectToServer(serverAddress)
		if err != nil {
			log.WithError(err).Error("Issues with server connection")
			time.Sleep(time.Second)
		} else if os.Getenv(fsbenchInContainer) == "true" {
			select {} // sleep forever in container!
		} else {
			os.Exit(0)
		}
	}
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
