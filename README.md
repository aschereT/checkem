# checkem
Checks REMAX mapping json for common errors

New and improved, now supports new schema!

TODO:
- Read metadata


Compilation:

```Shell
go build -ldflags="-s -w"
```


Get largest folders (for benchmarking):


```Shell
du -xhS | sort -h | tail -n15 | tac
```

Compare memory usage to checkem.sh:

```Shell
/usr/bin/time -f '%e seconds %P CPU (%M max)k' ~/dev/ops/apps/runner/maptests/checkem.sh gjara && /usr/bin/time -f '%e seconds %P CPU (%M max)k' ./checkem gjara
```


Compare time to checkem.sh:

```Shell
time ~/dev/ops/apps/runner/maptests/checkem.sh gjara && time ./checkem gjara
```

Check *ALL* the mappings:

```Shell
ls ~/dev/ops/apps/runner/mappings | xargs -n1 ./checkem
```
