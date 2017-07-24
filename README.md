# retry

[![Build Status](https://travis-ci.org/JaSei/pathutil-go.svg?branch=implementation)](https://travis-ci.org/JaSei/pathutil-go)
[![Build status](https://ci.appveyor.com/api/projects/status/urj0cf370sr5hjw4?svg=true)](https://ci.appveyor.com/project/JaSei/pathutil-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/JaSei/pathutil-go)](https://goreportcard.com/report/github.com/JaSei/pathutil-go)
[![GoDoc](https://godoc.org/github.com/JaSei/pathutil-go?status.svg)](http://godoc.org/github.com/jasei/pathutil-go)
[![Sourcegraph](https://sourcegraph.com/github.com/jasei/pathutil-go/-/badge.svg)](https://sourcegraph.com/github.com/jasei/pathutil-go?badge)
[![codecov.io](https://codecov.io/github/boennemann/badges/coverage.svg?branch=implementation)](https://codecov.io/github/jasei/pathutil-go?branch=implementation)

Simple library for retry mechanism

slightly inspired by [Try::Tiny::Retry](https://metacpan.org/pod/Try::Tiny::Retry)

## example

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

[next examples](examples)
