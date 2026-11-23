package com.example.nested

open class NestHolder {
    var own: Detail? = null
    var qualified: NestHolder.Detail? = null
    var other: NestOther.Detail? = null
    var others: List<NestOther.Detail> = emptyList()
    var deep: Mid.Deep? = null
    var badge: NestBadge? = null
    var title: String = ""

    class Detail {
        var code: String = ""
    }

    class NestBadge {
        var text: String = ""
    }

    class Mid {
        var detail: Detail? = null
        var deep: Deep? = null

        class Deep {
            var depth: Int = 0
        }
    }
}
