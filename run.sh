#!/bin/bash

TARGET=server

rm -vr $TARGET emc*
env GOOS=darwin GOARCH=386|grep GO
go build -o $TARGET

if [ -f "$TARGET" ]; then
    echo " ===== Build success, start service ====="
	./$TARGET
else
    echo " ===== Build fail... ====="
fi

