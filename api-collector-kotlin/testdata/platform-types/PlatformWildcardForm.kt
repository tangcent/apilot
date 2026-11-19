package com.example.platformtypes

import com.fasterxml.jackson.databind.node.*
import java.sql.*
import java.time.*
import java.util.*
import org.springframework.web.multipart.*

// Every name is imported on demand, so none of them is written in full.
class PlatformWildcardForm(
    val stamp: Timestamp,
    val clock: Time,
    val month: Month,
    val day: DayOfWeek,
    val months: List<Month>,
    val note: Optional<String>,
    val count: OptionalInt,
    val currency: Currency,
    val locale: Locale,
    val node: ObjectNode,
    val file: MultipartFile,
)
