#!/bin/sh
bin="go run sort-ip.go"
test_dir=target/test/
mkdir -p ${test_dir}
for i in tests/*.in; do
    name=$(basename $i)
    base=${test_dir}${name/.in/}
    $bin --range "$i" > ${base}.range
    $bin --cidr "$i" > ${base}.cidr
    $bin --range "${base}.cidr" > ${base}.cidr.range
    diff ${base}.range ${base}.cidr.range
done
