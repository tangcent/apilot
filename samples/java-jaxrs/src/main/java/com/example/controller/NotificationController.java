package com.example.controller;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import com.example.model.Notification;
import com.example.model.Result;
import com.example.model.PageResult;
import java.util.List;

@Path("/api/notifications")
public class NotificationController extends BaseCrudController<CreateNotificationReq, Notification> {

    @GET
    @Path("/user/{userId}")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<PageResult<Notification>> getUserNotifications(
            @PathParam("userId") Long userId,
            @QueryParam("type") String type,
            @QueryParam("read") Boolean read,
            @DefaultValue("1") @QueryParam("pageNum") Integer pageNum,
            @DefaultValue("10") @QueryParam("pageSize") Integer pageSize) {
        return Result.success(new PageResult<>());
    }

    @PUT
    @Path("/{id}/read")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Void> markAsRead(@PathParam("id") Long id) {
        return Result.success(null);
    }

    @PUT
    @Path("/user/{userId}/read-all")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Void> markAllAsRead(@PathParam("userId") Long userId) {
        return Result.success(null);
    }

    @POST
    @Path("/send")
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Void> sendNotification(SendNotificationReq req) {
        return Result.success(null);
    }

    @GET
    @Path("/templates")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<NotificationTemplateVO>> getTemplates() {
        return Result.success(List.of());
    }

    @POST
    @Path("/templates")
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public Result<NotificationTemplateVO> createTemplate(CreateTemplateReq req) {
        return Result.success(new NotificationTemplateVO());
    }

    @GET
    @Path("/user/{userId}/unread-count")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Long> getUnreadCount(@PathParam("userId") Long userId) {
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
