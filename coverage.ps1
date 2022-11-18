$cov_dir=".\coverage"
$env:WSL_UTF8=1
go.exe test -covermode=set -coverprofile="${cov_dir}\coverage.out" -coverpkg=./...
go.exe tool cover -o "${cov_dir}\index.html" -html="${cov_dir}\coverage.out"
