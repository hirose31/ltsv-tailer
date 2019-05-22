# ltsv-tailer

`ltsv-tailer` is the exporter for Prometheus that reads LTSV files like `tail -F` and exports metrics by given metrics configuration.

## How to run the example

```
./example/logger -o tailme -i 1 &

./ltsv-tailer -metrics example/metrics.yml -file tailme
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

## License

Licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at [http://www.apache.org/licenses/LICENSE-2.0].
