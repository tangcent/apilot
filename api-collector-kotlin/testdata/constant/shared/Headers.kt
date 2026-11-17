package com.example.constant.shared

interface Headers {
    companion object {
        /** API version the client speaks. */
        const val VERSION = "X-Api-Version"

        const val LANGUAGE = BaseHeaders.LANGUAGE
    }
}
