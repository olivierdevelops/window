# closures.cs — `do` is the escape hatch for any raw JS expression, so
# higher-order functions and arrow callbacks pass straight through.
fn makeCounter(start)
    do let n = start
    do return () => { n = n + 1; return n; }
end

const next = makeCounter(10)
log "first:  " + next()
log "second: " + next()
log "third:  " + next()

const doubled = [1, 2, 3, 4].map(x => x * 2)
log "doubled: " + doubled.join(", ")
