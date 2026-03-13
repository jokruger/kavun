export {
    fn: func(a, b, c) {
        four := import("./two/four/four.gs")
        return four.fn(a, b, c, "three")
    }
}