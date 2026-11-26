package com.example.override;

import com.fasterxml.jackson.annotation.JacksonAnnotationsInside;
import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonValue;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

interface Wire {
    @JsonValue
    String code();
}

interface Base {
    @JsonValue
    String code();
}

interface Mid extends Base {
    @JsonValue(false)
    String code();
}

interface Two {
    @JsonValue
    String code();

    @JsonValue
    String wire();
}

interface Typed<T> {
    @JsonValue
    T value();
}

interface TrueWire {
    @JsonValue
    String code();
}

interface FalseWire {
    @JsonValue(false)
    String code();
}

interface DefaultedWire {
    @JsonValue
    default String code() {
        return "x";
    }
}

@JsonFormat(shape = JsonFormat.Shape.NUMBER)
interface AsNum {}

@JsonFormat(shape = JsonFormat.Shape.OBJECT)
interface AsObj {}

@JsonFormat(shape = JsonFormat.Shape.ARRAY)
interface AsArr {}

@JsonFormat(shape = JsonFormat.Shape.STRING)
interface AsStr {}

@JsonSerialize(using = Lower.class)
interface AsSer {}

@JsonFormat(shape = JsonFormat.Shape.NUMBER)
interface NumBase {}

interface NumMid extends NumBase {}

@Retention(RetentionPolicy.RUNTIME)
@JacksonAnnotationsInside
@JsonValue
@interface AsValue {}

@Retention(RetentionPolicy.RUNTIME)
@JacksonAnnotationsInside
@JsonValue(false)
@interface NotValue {}

@Retention(RetentionPolicy.RUNTIME)
@JacksonAnnotationsInside
@JsonFormat(shape = JsonFormat.Shape.NUMBER)
@interface AsNumber {}

@JacksonAnnotationsInside
@JsonValue
@interface AsValueClass {}

interface BundleWire {
    @AsValue
    String code();
}

enum Plain {
    A,
    B
}

enum Off implements Wire {
    A,
    B;

    @JsonValue(false)
    public String code() {
        return "x";
    }
}

enum Inherit implements Wire {
    A,
    B;

    public String code() {
        return "x";
    }
}

enum ExplicitOn implements Wire {
    A,
    B;

    @JsonValue(true)
    public String code() {
        return "x";
    }
}

enum MidOff implements Mid {
    A,
    B;

    public String code() {
        return "x";
    }
}

enum Reopen implements Mid {
    A,
    B;

    @JsonValue(true)
    public String code() {
        return "x";
    }
}

enum OwnOff {
    A,
    B;

    @JsonValue(false)
    public String code() {
        return "x";
    }
}

enum OneOff implements Two {
    A,
    B;

    @JsonValue(false)
    public String code() {
        return "x";
    }

    public String wire() {
        return "y";
    }
}

enum Overload implements Wire {
    A,
    B;

    @JsonValue(false)
    public String code(int ignored) {
        return "n";
    }

    public String code() {
        return "x";
    }
}

enum GenericOn implements Typed<String> {
    A,
    B;

    public String value() {
        return "x";
    }
}

enum GenericOff implements Typed<String> {
    A,
    B;

    @JsonValue(false)
    public String value() {
        return "x";
    }
}

enum FirstTrue implements TrueWire, FalseWire {
    A,
    B;

    public String code() {
        return "x";
    }
}

enum FirstFalse implements FalseWire, TrueWire {
    A,
    B;

    public String code() {
        return "x";
    }
}

enum Defaulted implements DefaultedWire {
    A,
    B
}

enum PrivateVal {
    A,
    B;

    @JsonValue
    private String code() {
        return "x";
    }
}

enum IfaceNum implements AsNum {
    A,
    B
}

enum IfaceObj implements AsObj {
    A,
    B
}

enum IfaceArr implements AsArr {
    A,
    B
}

enum IfaceStr implements AsStr {
    A,
    B
}

enum IfaceSer implements AsSer {
    A,
    B
}

@JsonFormat(shape = JsonFormat.Shape.STRING)
enum StringOver implements AsNum {
    A,
    B
}

@JsonSerialize(using = com.fasterxml.jackson.databind.JsonSerializer.None.class)
enum NoneClears implements AsSer {
    A,
    B
}

enum ThroughNum implements NumMid {
    A,
    B
}

enum BundleVal {
    A,
    B;

    @AsValue
    public String code() {
        return "x";
    }
}

enum BundleOff implements Wire {
    A,
    B;

    @NotValue
    public String code() {
        return "x";
    }
}

@AsNumber
enum BundleNum {
    A,
    B
}

enum BundleIface implements BundleWire {
    A,
    B;

    public String code() {
        return "x";
    }
}

enum ClassRet {
    A,
    B;

    @AsValueClass
    public String code() {
        return "x";
    }
}

enum ConstBody {
    A {
        @JsonValue
        public String extra() {
            return "x";
        }
    },
    B
}

enum BodyFalse {
    A {
        @JsonValue(false)
        public String extra() {
            return "x";
        }
    },
    B
}

enum BodyPlain {
    A {
        public String label() {
            return "aa";
        }
    },
    B
}

enum BodyKeeps implements Wire {
    A {
        @JsonValue(false)
        public String code() {
            return "x";
        }
    },
    B;

    public String code() {
        return "x";
    }
}

class OverrideForm {
    private Plain namePlain;
    private Off nameOff;
    private MidOff nameMid;
    private OwnOff nameOwnOff;
    private GenericOff nameGenericOff;
    private StringOver nameStringOver;
    private NoneClears nameNone;
    private BundleOff nameBundleOff;
    private ClassRet nameClassRet;
    private IfaceStr nameIfaceString;
    private FirstFalse nameFirstFalse;
    private ConstBody limitConst;
    private BodyFalse nameBodyFalse;
    private BodyPlain nameBodyPlain;
    private BodyKeeps valueBodyKeeps;

    private Inherit valueInherit;
    private ExplicitOn valueExplicit;
    private Reopen valueReopen;
    private OneOff valueOneOff;
    private Overload valueOverload;
    private GenericOn valueGeneric;
    private IfaceNum valueNumber;
    private IfaceObj valueObject;
    private IfaceArr valueArray;
    private IfaceSer valueSerialize;
    private ThroughNum valueThrough;
    private BundleVal valueBundle;
    private BundleNum valueBundleNum;
    private BundleIface valueBundleIface;
    private Defaulted valueDefault;
    private PrivateVal valuePrivate;
    private FirstTrue valueFirstTrue;
}

@RestController
@RequestMapping("/override")
class OverrideController {
    @PostMapping
    public void create(@RequestBody OverrideForm form) {
    }

    @GetMapping
    public void page(OverrideForm query) {
    }
}
