setq(x, 100)

switch(x) {
    case(100) {
        log('x equals 100')
    }
    case(200) {
        log('x equals 200')
    }
    default() {
        log('x is neither 100 nor 200')
    }
}

switch() {
    case(equal(x, 100)) {
        log('x equals 100')
    }
    case(equal(x, 200)) {
        log('x equals 200')
    }
    default() {
        log('x is neither 100 nor 200', 'info', x)
    }
}
x