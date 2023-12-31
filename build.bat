@echo off
setlocal

rem Get the directory of the script
set "repoDir=%~dp0"
set "outDir=%repoDir%out"

rem Create the output directory
mkdir "%outDir%"

rem Build the Go application
go build -C "%repoDir%src" -o %outDir%

endlocal

