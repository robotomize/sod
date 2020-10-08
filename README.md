# SOD

**Note**: The `master` branch may be in an *unstable or even broken state* during development. Please use [releases][github-release] instead of the `master` branch in order to get stable binaries.

**Note**: The current version is not intended for use in production

![SOD Logo](docs/images/sod-horizontal-small.svg)

SOD  - simple outlier detection. A simple solution for detecting anomalies in a vector data stream with a focus on being:

* *Simple*: well-defined, user-facing HTTP API
* *Fast*: benchmarked 1,000 prediction/sec on core, million samples
* *Storage*: automatically maintains the actual required range of the sample data
* *Without training*: Uses the k nearest neighbor algorithm to detect anomalies without training data

## Description

SOD is written in Go. It consists of several components. The data storage system uses etcd/bbolt, which is a key/value data store on disk. SOD uploads batches of data to disk. When running SOD, the data is expanded in memory. Anomaly recognition is based on the LOF - local outlier factor method. This method uses k nearest neighbors to detect anomalies. The algorithm for determining k nearest neighbors uses a kd tree. An important component of the SOD is a notifier. It sends a POST request with a warning about anomalies to the specified address.

## LOF white paper

![lof image](docs/images/lof.png)

[LOF agorithm white paper](https://www.dbs.ifi.lmu.de/Publikationen/Papers/LOF.pdf)

## Getting started

### Getting SOD

The simplest method to run is to run the docker image and throw the necessary environment variables.

### Running SOD

SOD uses the configuration method via environment variables. 
Running SOD from the repository

```bash
go run cmd/sod
```

or

```bash
go build cmd/sod
```

or 

```bash
docker build .
```

By default, the application runs on port 8787. SOD operates in two modes: collect and predict:

* Predict - To find out if the value is an outlier for non-saving data, send A post request to /predict
* Collect - To save the value in SOD, recognize it, and inform your applications about the outlier, send a POST request to the /collect address

### Predict request

 Your request will be mapped to the following structure
```json
{
  "entityId": "user-lives",
  "data": [
    {"vec": [1.1, 1.7], "extra": "player id 1 lives", "createdAt": "timestamp"}
  ] 
}
```

```bash
curl -X POST -d '{entityId: "user-lives", "data": [{"vec": [1.1, 1.7], "extra": "player id 1 lives", "createdAt": "timestamp"}]}' http://localhost:8787/predict
```

### Collect request

```json
{
  "entityId": "user-lives",
  "data": [
    {"vec": [1.1, 1.7], "extra": "player id 1 lives", "createdAt": "timestamp"}
  ] 
}
```

```bash
curl -X POST -d '{"entityId": "user-lives", "data": [{"vec": [1.1, 1.7], "extra": "player id 1 lives", "createdAt": "timestamp"}]}' http://localhost:8787/collect
```

### License

SOD is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
