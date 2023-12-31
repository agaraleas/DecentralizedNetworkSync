currentDir="$(pwd)"
repoDir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$repoDir/src/"
go test ./...
cd $currentDir