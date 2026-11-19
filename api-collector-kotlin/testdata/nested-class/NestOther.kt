package com.example.nested

class NestOther {
    var own: Detail? = null
    var badge: NestBadge? = null

    class Detail {
        var amount: Long = 0
    }
}
