@echo off
setlocal

rem Get the directory of the script
set "repoDir=%~dp0"

rem Change directory to the root of the Go module
cd "%repoDir%src"

rem Run go test
go test ./...

endlocal
