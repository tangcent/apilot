package com.example.controller;

import org.springframework.web.bind.annotation.*;
import com.example.model.Notification;
import com.example.model.Result;
import com.example.model.PageResult;
import java.util.List;

@RestController
@RequestMapping("/api/notifications")
public class NotificationController extends BaseCrudController<CreateNotificationReq, Notification> {

    @GetMapping("/user/{userId}")
    public Result<PageResult<Notification>> getUserNotifications(
            @PathVariable Long userId,
            @RequestParam(required = false) String type,
            @RequestParam(required = false) Boolean read,
            @RequestParam(defaultValue = "1") Integer pageNum,
            @RequestParam(defaultValue = "10") Integer pageSize) {
        return Result.success(new PageResult<>());
    }

    @PutMapping("/{id}/read")
    public Result<Void> markAsRead(@PathVariable Long id) {
        return Result.success(null);
    }

    @PutMapping("/user/{userId}/read-all")
    public Result<Void> markAllAsRead(@PathVariable Long userId) {
        return Result.success(null);
    }

    @PostMapping("/send")
    public Result<Void> sendNotification(@RequestBody SendNotificationReq req) {
        return Result.success(null);
    }

    @GetMapping("/templates")
    public Result<List<NotificationTemplateVO>> getTemplates() {
        return Result.success(List.of());
    }

    @PostMapping("/templates")
    public Result<NotificationTemplateVO> createTemplate(@RequestBody CreateTemplateReq req) {
        return Result.success(new NotificationTemplateVO());
    }

    @GetMapping("/user/{userId}/unread-count")
    public Result<Long> getUnreadCount(@PathVariable Long userId) {
        return Result.success(0L);
    }
}

class CreateNotificationReq {
    private Long userId;
    private String type;
    private String title;
    private String content;

    public Long getUserId() { return userId; }
    public void setUserId(Long userId) { this.userId = userId; }
    public String getType() { return type; }
    public void setType(String type) { this.type = type; }
    public String getTitle() { return title; }
    public void setTitle(String title) { this.title = title; }
    public String getContent() { return content; }
    public void setContent(String content) { this.content = content; }
}

class SendNotificationReq {
    private List<Long> userIds;
    private String type;
    private String templateCode;
    private Object data;

    public List<Long> getUserIds() { return userIds; }
    public void setUserIds(List<Long> userIds) { this.userIds = userIds; }
    public String getType() { return type; }
    public void setType(String type) { this.type = type; }
    public String getTemplateCode() { return templateCode; }
    public void setTemplateCode(String templateCode) { this.templateCode = templateCode; }
    public Object getData() { return data; }
    public void setData(Object data) { this.data = data; }
}

class NotificationTemplateVO {
    private Long id;
    private String code;
    private String name;
    private String type;
    private String titleTemplate;
    private String contentTemplate;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getCode() { return code; }
    public void setCode(String code) { this.code = code; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getType() { return type; }
    public void setType(String type) { this.type = type; }
    public String getTitleTemplate() { return titleTemplate; }
    public void setTitleTemplate(String titleTemplate) { this.titleTemplate = titleTemplate; }
    public String getContentTemplate() { return contentTemplate; }
    public void setContentTemplate(String contentTemplate) { this.contentTemplate = contentTemplate; }
}

class CreateTemplateReq {
    private String code;
    private String name;
    private String type;
    private String titleTemplate;
    private String contentTemplate;

    public String getCode() { return code; }
    public void setCode(String code) { this.code = code; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getType() { return type; }
    public void setType(String type) { this.type = type; }
    public String getTitleTemplate() { return titleTemplate; }
    public void setTitleTemplate(String titleTemplate) { this.titleTemplate = titleTemplate; }
    public String getContentTemplate() { return contentTemplate; }
    public void setContentTemplate(String contentTemplate) { this.contentTemplate = contentTemplate; }
}
