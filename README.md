![GitHub](https://img.shields.io/github/license/hirose31/ltsv-tailer)
[![test](https://github.com/hirose31/ltsv-tailer/actions/workflows/test.yml/badge.svg)](https://github.com/hirose31/ltsv-tailer/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hirose31/ltsv-tailer?style=flat-square)](https://goreportcard.com/report/github.com/hirose31/ltsv-tailer)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/hirose31/ltsv-tailer)

# ltsv-tailer

`ltsv-tailer` is the exporter for Prometheus that reads LTSV files like `tail -F` and exports metrics by given metrics configuration.

## How to run the example

```
./example/logger -o tailme -i 1 &

./ltsv-tailer -metrics example/metrics.yml -file tailme -logtostderr &

curl http://127.0.0.1:9588/metrics
```

## Metrics Configuration

Example metrics configuration:

``` yaml
transform:
  tolower:
    - method
  tosec:
    - resptime: microsec
metrics:
  - name: ltsv_http_request_count_total
    kind: counter
    value_key: COUNTER
    help: http request count total
    labels:
      - vhost
      - method
      - code
  - name: ltsv_http_response_bytes_total
    kind: counter
    value_key: size
    help: http response bytes total
    labels:
      - vhost
      - method
      - code
  - name: ltsv_http_response_seconds
    kind: histogram
    value_key: resptime
    help: http response seconds
    buckets:
      - 0.1
      - 0.5
      - 1.0
      - 2.0
      - 4.0
      - 8.0
    labels:
      - vhost
      - method
      - code
      - path
      - host
```
