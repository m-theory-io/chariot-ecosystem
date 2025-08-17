// Chariot Script Example

// 1. declare a function variable
declare(plus100, 'F', func(n) {
    declare(x, 'N', n)
    setq(result, add(x, 100))
    result
})
// 2. call the function
call(plus100, 500)
