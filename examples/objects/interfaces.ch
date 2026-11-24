interface(Answerer) {
    // Caller sees no arguments: the requirement excludes self.
    requires(answer, signature(array(), "N"))
}
class(Answer) {
    field(value, "N", 42)
}
method(Answer, answer,
    // Stored method includes both the self parameter and its V contract.
    withSignature(func(self) { return(self.value) }, signature(array("V"), "N")))
setq(answer, new(Answer))
declare(bound, "F", answer.answer)
declare(object, "O", answer)
// Final result: [true, 42, 42, "F", "O"].
array(satisfies(object, Answerer), assertSatisfies(object, Answerer).answer(),
    call(bound), valueType(bound), valueType(object))
