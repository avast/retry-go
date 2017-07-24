# retry

[![Linux Build Status](https://travis-ci.org/avast/retry-go.svg)](https://travis-ci.org/avast/retry-go)
[![Windows Build status](https://ci.appveyor.com/api/projects/status/fieg9gon3qlq0a9a?svg=true)](https://ci.appveyor.com/project/JaSei/retry-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/avast/retry-go)](https://goreportcard.com/report/github.com/avast/retry-go)
[![GoDoc](https://godoc.org/github.com/avast/retry-go?status.svg)](http://godoc.org/github.com/avast/retry-go)
[![Sourcegraph](https://sourcegraph.com/github.com/avast/retry-go/-/badge.svg)](https://sourcegraph.com/github.com/avast/retry-go?badge)
[![codecov.io](https://codecov.io/github/boennemann/badges/coverage.svg?branch=master)](https://codecov.io/github/avast/retry-go?branch=master)

Simple library for retry mechanism

slightly inspired by [Try::Tiny::Retry](https://metacpan.org/pod/Try::Tiny::Retry)

## EXAMPLE

http get with retry:

```go
url := "http://example.com"
var body []byte

err := retry.Retry(
	func() error {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return nil
	},
)

fmt.Println(body)
```

[next examples](examples)

## SEE ALSO
* [giantswarm/retry-go](https://github.com/giantswarm/retry-go) - slightly complicated interface.
* [sethgrid/pester](https://github.com/sethgrid/pester) - only http retry for http calls with retries and backoff
* [cenkalti/backoff](https://github.com/cenkalti/backoff) - Go port of the exponential backoff algorithm from Google's HTTP Client Library for Java. Really complicated interface.
