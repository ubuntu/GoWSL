# GoWSL

This is a Work In Process. It will become a Go module that wraps around WSL syscalls on windows.

# How to use
To consume it:
- This project runs on Windows 10 or above.
- WSL must be enabled or installed from the store.
- You need to have Go installed.

To run the tests and the examples:
- Create a directory named `images` in the project root.
- Download the jammy tarball from [here](https://cloud-images.ubuntu.com/wsl/jammy/current/), and store it as `images\jammy.tar.gz`.
- Create an empty file, rename it and store it as `images\empty.tar.gz`