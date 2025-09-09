# GTClock
<!-- markdownlint-disable MD034 -->
<!-- markdownlint-disable MD013 -->
[![Go Report Card](https://goreportcard.com/badge/github.com/karasz/gtclock)](https://goreportcard.com/report/github.com/karasz/gtclock)
[![Unlicensed](https://img.shields.io/badge/license-Unlicense-blue.svg)](https://github.com/karasz/gnocco/blob/master/UNLICENSE)

A Go implementation of TAICLOCK protocol.

gtclock is a multi binary so after installing create the following symlinks:

* gtclocd - called by this name gtclock will run a TAIN time server
* gntpclock - called bt this name gtclock will run a SNTP client
* gtailocal - called this way gtclock will read from its standard input and
  write to standard output replacing TAI or TAIN labels with RFC3399 timestamps
