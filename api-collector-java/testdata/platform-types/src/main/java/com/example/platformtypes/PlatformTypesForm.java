package com.example.platformtypes;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.node.ArrayNode;
import com.fasterxml.jackson.databind.node.ObjectNode;
import java.sql.Time;
import java.sql.Timestamp;
import java.time.DayOfWeek;
import java.time.Month;
import java.time.ZoneId;
import java.util.Currency;
import java.util.List;
import java.util.Locale;
import java.util.Optional;
import java.util.OptionalDouble;
import java.util.OptionalInt;
import java.util.OptionalLong;
import java.util.TimeZone;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.LongAdder;
import org.springframework.web.multipart.MultipartFile;

// The same form as api-collector-kotlin/testdata/platform-types/PlatformTypesController.kt.
public class PlatformTypesForm {
    // A collection or a map spelled with its package.
    private java.util.List<String> qualifiedList;
    private java.util.Set<Long> qualifiedSet;
    private java.util.Collection<Integer> qualifiedCollection;
    private java.lang.Iterable<String> qualifiedIterable;
    private java.util.Map<String, Long> qualifiedMap;
    private Iterable<String> iterable;
    private java.util.LinkedHashSet<String> linkedSet;
    private java.util.SortedMap<String, Integer> sortedMap;

    // A value that may be absent is written as the value, or left out.
    private Optional<String> optionalName;
    private java.util.Optional<Long> qualifiedOptional;
    private Optional<PlatformItem> optionalItem;
    private Optional<List<String>> optionalTags;
    private OptionalInt optionalInt;
    private OptionalLong optionalLong;
    private OptionalDouble optionalDouble;

    // JDK values.
    private CharSequence text;
    private Locale locale;
    private Currency currency;
    private TimeZone timeZone;
    private ZoneId zoneId;
    private DayOfWeek day;
    private Month month;
    private AtomicInteger atomicInteger;
    private AtomicLong atomicLong;
    private AtomicBoolean atomicBoolean;
    private LongAdder longAdder;
    private Timestamp timestamp;
    private java.sql.Date sqlDate;
    private Time sqlTime;

    // Library classes.
    private JsonNode jsonNode;
    private ObjectNode objectNode;
    private ArrayNode arrayNode;
    private MultipartFile file;
}
