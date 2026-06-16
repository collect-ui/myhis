go build -ldflags="-s -w" -o windows/main.exe -v -x main.go
# 如果有upx 可以进行压缩一下 没有可以跳过
F:/go/upx-5.0.1-win64/upx.exe F:/go/auto-desk/windows/main.exe