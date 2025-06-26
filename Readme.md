# Fsbench

Fsbench is the Golang reimplementation of [Vdbench](https://www.oracle.com/downloads/server-storage/vdbench-downloads.html).
It is a distributed FS performance benchmark tool with [Prometheus exporter](https://opencensus.io/exporters/supported-exporters/go/prometheus/).

For more details, please refer to [fsbench wiki](https://deepwiki.com/microyahoo/fsbench)

## Usage

Fsbench consists of two parts:

* Server: Coordinates Workers and general test queue
* Workers: Actually perform reading, writing, deleting and listing of objects

INFO: `--debug` activates debug logging, `--trace` activates trace logging

### Running a test

#### Running locally
1. Build the server: `go build -o ./bin/fsbench-server ./cmd/server`
2. Run the server, specifying a config file: `bin/fsbench-server --config.file path/to/config.yaml` - you can find an example config [in the deploy folder](deploy/example_config.yaml). Additionally, you can configure the S3-related settings in the configuration file, so the test results will be automatically saved to S3 after the test run. If running locally, the test results will be saved in the /tmp directory with filenames formatted as `/tmp/fsbench_result_<timestamp>.<format>`.
3. The server will open port 2000 for workers to connect to - make sure this port is not blocked by your firewall!
4. Build the worker: `go build -o ./bin/fsbench-worker ./cmd/worker`
5. Run the worker, specifying the server connection details: `bin/fsbench-worker --server.address <server-ip>:2000`
6. The worker will immediately connect to the server and will start to get to work.
The worker opens port 8888 for the Prometheus exporter. Please make sure this port is allowed in your firewall and that you added the worker to the Prometheus config.

#### Running in K8s
1. Run `make build` to generate container images for both the fsbench server and worker. Note that you may need to replace the base image in [Dockerfile](Dockerfile).
2. Modify `deploy/fsbench.yaml`, then run `kubectl apply -f deploy/fsbench.yaml`.
