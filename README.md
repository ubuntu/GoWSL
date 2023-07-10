# GoWSL
This module offers an idiomatic interface between your Go code and the Windows Subsystem for Linux (WSL) API (`wslApi.dll`  and occasionally `wsl.exe`). It offers wrappers around common actions to manage WSL distros.

[![Code quality](https://github.com/ubuntu/GoWSL/actions/workflows/qa-azure.yaml/badge.svg)](https://github.com/ubuntu/GoWSL/actions/workflows/qa-azure.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ubuntu/gowsl.svg)](https://pkg.go.dev/github.com/ubuntu/gowsl)
[![Go Report Card](https://goreportcard.com/badge/ubuntu/gowsl)](https://goreportcard.com/report/ubuntu/gowsl)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/ubuntu/gowsl/blob/main/LICENSE)

## Aim

We aim not to extend the aforementioned API, but rather to provide a safe, idiomatic, and easy-to-use wrapper around it. The goal is to enable the development of applications that build on top of it.

## Requirements

- Windows Subsystem for Linux must be installed ([documentation](https://learn.microsoft.com/en-us/windows/wsl/install)) and enabled.
- Go version must be equal to or above 1.20.

## Development

Your help would be very much appreciated! Check out the [CONTRIBUTING](./CONTRIBUTING.md) document to see how you could collaborate.

## Contact

You are welcome to create a new issue on this repository if you find bugs or wish to make any feature request.
