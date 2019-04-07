# checkem
Checks REMAX mapping json for common errors


Get largest folders (for benchmarking):


```
du -xhS | sort -h | tail -n15 | tac
```

Compare memory usage to checkem.sh:

```
/usr/bin/time -f '%e seconds %P CPU (%M max)k' ~/dev/ops/apps/runner/maptests/checkem.sh gjara && /usr/bin/time -f '%e seconds %P CPU (%M max)k' ./checkem gjara
```


Compare time to checkem.sh:

```
time ~/dev/ops/apps/runner/maptests/checkem.sh gjara && time ./checkem gjara
```

Check *ALL* the mappings:

```
ls ~/dev/ops/apps/runner/mappings | xargs -n1 ./checkem
```
