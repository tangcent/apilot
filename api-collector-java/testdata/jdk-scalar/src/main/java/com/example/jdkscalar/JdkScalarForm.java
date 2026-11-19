package com.example.jdkscalar;

import java.util.List;
import java.util.Map;

public class JdkScalarForm {
    private short primitiveShort;
    private Short boxedShort;
    private java.lang.Short qualifiedShort;
    private short[] shorts;
    private Short[] boxedShorts;
    private java.lang.Short[] qualifiedShorts;
    private List<Short> shortList;
    private Map<String, Short> shortsByName;

    /** A boxed type spelled with its package is the same type as the simple one. */
    private java.lang.Integer qualifiedInteger;
    private java.lang.Long qualifiedLong;
    private java.lang.Float qualifiedFloat;
    private java.lang.Double qualifiedDouble;
    private java.lang.Boolean qualifiedBoolean;

    /** Jackson writes any Number as a JSON number. */
    private Number number;
    private java.lang.Number qualifiedNumber;
}
