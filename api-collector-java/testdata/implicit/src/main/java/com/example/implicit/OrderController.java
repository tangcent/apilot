package com.example.implicit;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonProperty;
import io.swagger.v3.oas.annotations.media.Schema;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotNull;
import java.time.LocalDate;
import java.util.List;
import org.springframework.data.domain.Pageable;
import org.springframework.data.domain.Sort;
import org.springframework.format.annotation.DateTimeFormat;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.ModelAttribute;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

// The same handlers as api-collector-kotlin/testdata/implicit/OrderController.kt.

enum Status { ACTIVE, CLOSED }

class Address {
    private String city;
}

class OrderQuery {
    /** Order number. */
    private String orderNo;
    private List<String> regionList;
    private Address address;
    @JsonProperty("display_name")
    private String displayName;
    @JsonIgnore
    private String secret;

    /** Read-only: Spring cannot bind it. */
    public String getSummary() {
        return orderNo;
    }
}

class OrderFilter {
    private String name;
}

class Operator {
    private Long id;
}

@RestController
@RequestMapping("/orders")
public class OrderController {

    @GetMapping
    public String page(@Valid OrderQuery query, Pageable pageable, HttpServletRequest request,
                       @NotNull String keyword, List<Long> ids,
                       @Schema(defaultValue = "ACTIVE") Status status,
                       @DateTimeFormat(iso = DateTimeFormat.ISO.DATE) LocalDate from) {
        return "";
    }

    @PostMapping("/form")
    public String submit(OrderQuery form, List<MultipartFile> attachments) {
        return "";
    }

    @PostMapping("/json")
    public String json(@RequestBody OrderQuery body, String trace) {
        return "";
    }

    @GetMapping("/me")
    public String me(@CurrentUser Operator operator, Long id) {
        return "";
    }

    @GetMapping("/search")
    public String search(@Audit @ModelAttribute OrderFilter filter, Sort sort) {
        return "";
    }
}
