oldGOPATH=$GOPATH
nowPATH=`pwd`
export GOPATH=$oldGOPATH:$nowPATH
echo "GOPATH = $GOPATH"

export GOBIN=$nowPATH/bin
echo "GOBIN = $GOBIN"