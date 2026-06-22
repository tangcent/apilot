package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Notification;
import com.example.model.Result;
import com.example.model.PageResult;
import java.util.List;

public interface NotificationClient extends BaseCrudClient<CreateNotificationReq, Notification> {

    @RequestLine("GET /api/notifications/user/{userId}?type={type}&read={read}&pageNum={pageNum}&pageSize={pageSize}")
    Result<PageResult<Notification>> getUserNotifications(@Param("userId") Long userId,
                                                           @Param("type") String type,
                                                           @Param("read") Boolean read,
                                                           @Param("pageNum") Integer pageNum,
                                                           @Param("pageSize") Integer pageSize);

    @RequestLine("PUT /api/notifications/{id}/read")
    Result<Void> markAsRead(@Param("id") Long id);

    @RequestLine("PUT /api/notifications/user/{userId}/read-all")
    Result<Void> markAllAsRead(@Param("userId") Long userId);

    @RequestLine("POST /api/notifications/send")
    Result<Void> sendNotification(SendNotificationReq req);

    @RequestLine("GET /api/notifications/templates")
    Result<List<NotificationTemplateVO>> getTemplates();

    @RequestLine("POST /api/notifications/templates")
    Result<NotificationTemplateVO> createTemplate(CreateTemplateReq req);

    @RequestLine("GET /api/notifications/user/{userId}/unread-count")
    Result<Long> getUnreadCount(@Param("userId") Long userId);
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
