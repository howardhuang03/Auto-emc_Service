#!/bin/bash

TARGET=server

rm -vr $TARGET emc*

export GOOS=windows
export GOARCH=386
go build -o $TARGET.exe
cp $TARGET.exe ~/Dropbox/tmp/server/
export GOOS=darwin
export GOARCH=amd64
go build -o $TARGET

if [ -f "$TARGET" ]; then
    echo " ===== Build success, start service ====="
	./$TARGET
else
    echo " ===== Build fail... ====="
fi

