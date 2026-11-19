package com.example.platformtypes;

import java.util.List;
import org.springframework.web.multipart.MultipartFile;

public class PlatformUploadForm {
    private String title;
    private MultipartFile cover;
    private List<MultipartFile> attachments;
    private MultipartFile[] extras;
}
