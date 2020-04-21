#!/bin/sh
set -e
go build
bin="./cidr-merger"
test_dir=target/test
mkdir -p "$test_dir"

doTest() {
    for i in tests/*.in; do
        name="${i##*/}"
        echo "running $name"
        base="$test_dir/${name%.in}"
        $bin --range "$i" > "$base.range"
        $bin --cidr "$i" > "$base.cidr"
        $bin --range "$base.cidr" > "$base.cidr.range"
        diff "$base.range" "$base.cidr.range"
    done
}

doTest
