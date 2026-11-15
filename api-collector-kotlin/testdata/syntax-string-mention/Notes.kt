package com.example.mention

annotation class Marker(val groups: Array<String> = [])

class Note {
    // Mentions of @RestController in a comment or a string are not annotations.
    var text: String? = "@RestController"
}
