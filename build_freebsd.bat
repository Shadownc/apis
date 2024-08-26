@echo off
REM 设置目标操作系统和架构
set GOOS=freebsd
set GOARCH=amd64

REM 定义源文件和输出文件名
set SOURCE=main.go
set OUTPUT=imgapi

REM 检查源文件是否存在
if not exist %SOURCE% (
    echo Source file %SOURCE% does not exist.
    exit /b 1
)

REM 编译Go程序
go build -o %OUTPUT% %SOURCE%

REM 检查是否编译成功
if "%ERRORLEVEL%"=="0" (
    echo Build successful: %OUTPUT%
) else (
    echo Build failed
    exit /b 1
)
