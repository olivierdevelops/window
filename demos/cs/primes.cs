# primes.cs — trial-division sieve using while + if and a raw `do` break.
fn isPrime(n)
    if n < 2
        return false
    end
    let d = 2
    while d * d <= n
        if n % d == 0
            return false
        end
        do d = d + 1
    end
    return true
end

log "Primes under 50:"
let line = ""
for n in Array.from({length: 50}, (_, k) => k)
    if isPrime(n)
        do line = line + n + " "
    end
end
log line
