# GoWSL
This module ofers an idiomatic interface between your Go code and the Windows Subsystem for Linux (WSL) API (`wslApi.dll`  and ocasionally `wsl.exe`). It offers wrappers around common actions to manage WSL distros. 

## Aim
We aim not to extend the aforementioned API, but rather to provide a safe, idomatic, and easy-to-use wrapper around it. The goal is to enable the development of applications that build on top of it. 

## Requirements
- Windows Subsystem for Linux must be installed ([documentation](https://learn.microsoft.com/en-us/windows/wsl/install)) and enabled.
- Go version must be equal to or above 1.18.

## Development
This module is still in its infancy, but quickly reaching maturity. For developers wanting to test, you must first complete a couple of  steps of setup:
- Create a directory named `images` in the project root.
- Download the Ubuntu 22.04 LTS tarball from [here](https://cloud-images.ubuntu.com/wsl/jammy/current/), and store it as `.\images\jammy.tar.gz`.
- Create an empty file, rename it and store it as `.\images\empty.tar.gz`


Then you can run the tests:

```powershell
go.exe test
```
The tests take a few minutes to run due to the delay in registering distros and the fact that `wslApi.dll` is not thread-safe.

## Examples
You can see some example usage in the tests, as well as in [examples/demo.go](examples/demo.go). If you only want to run the example, use the following command while reading along the file to understand what it is doing.
```powershell
go.exe run .\examples\demo.go
```

## Contact
You are welcome to create a new issue on this repository if you find bugs or wish to make any feature request.