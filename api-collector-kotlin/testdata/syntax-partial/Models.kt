package com.example.partial

class First {
    var a: String? = null
}

/**
 * One member the grammar cannot read; the rest of the class is intact.
 */
class Damaged {
    var kept: String? = null
    var broken: String? = )
    var alsoKept: Long? = null
}

/**
 * Parsed with the whole file, the grammar folds this class into the error
 * above; parsed on its own, it is intact.
 */
class Third {
    var c: String? = null
}
