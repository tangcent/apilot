package com.example.platformtypes;

import com.fasterxml.jackson.databind.node.*;
import java.sql.*;
import java.time.*;
import java.util.*;
import org.springframework.web.multipart.*;

// The same form as api-collector-kotlin/testdata/platform-types/PlatformWildcardForm.kt:
// every name is imported on demand, so none of them is written in full.
public class PlatformWildcardForm {
    private Timestamp stamp;
    private Time clock;
    private Month month;
    private DayOfWeek day;
    private List<Month> months;
    private Optional<String> note;
    private OptionalInt count;
    private Currency currency;
    private Locale locale;
    private ObjectNode node;
    private MultipartFile file;
}
