# fizzbuzz.cs — for-in over a range, nested if/else.
fn fizzbuzz(n)
    if n % 15 == 0
        return "FizzBuzz"
    else
        if n % 3 == 0
            return "Fizz"
        else
            if n % 5 == 0
                return "Buzz"
            else
                return String(n)
            end
        end
    end
end

for i in Array.from({length: 20}, (_, k) => k + 1)
    log fizzbuzz(i)
end
