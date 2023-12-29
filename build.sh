repoDir=$(dirname "$0")
mkdir -p "$repoDir/out"
go build -C "$repoDir/src" -o "../out/DecentralizedNetworkSync"
