# fib.cs — recursion, if/else, and a while loop.
const N = 15

fn fib(n)
    if n < 2
        return n
    else
        return fib(n - 1) + fib(n - 2)
    end
end

log "Fibonacci sequence:"
let i = 0
while i < N
    log "fib(" + i + ") = " + fib(i)
    do i = i + 1
end
