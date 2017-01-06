#!/bin/bash

MAC_TARGET=server
WIN_TARGET=server.exe

rm -vr $MAC_TARGET $WIN_TARGET emc*

export GOOS=windows
export GOARCH=386
while [ ! -f "$WIN_TARGET" ]
do
	go build -o $WIN_TARGET
done
echo "$WIN_TARGET build success, next step..."
cp $WIN_TARGET ~/Dropbox/tmp/server/

export GOOS=darwin
export GOARCH=amd64
go build -o $MAC_TARGET

if [ -f "$MAC_TARGET" ]; then
    echo " ===== $MAC_TARGET build success, start service ====="
	./$MAC_TARGET
else
    echo " ===== Build fail... ====="
fi

