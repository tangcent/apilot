package com.example;

import feign.Param;
import feign.RequestLine;
import java.util.List;
import com.example.model.Result;
import com.example.model.PageResult;
import com.example.client.BaseClient;

public interface UserClient extends BaseClient {

    @RequestLine("GET /users?name={name}&role={role}&status={status}&pageNum={pageNum}&pageSize={pageSize}")
    PageResult<UserVO> listUsers(@Param("name") String name, @Param("role") String role,
                                  @Param("status") Integer status,
                                  @Param("pageNum") int pageNum, @Param("pageSize") int pageSize);

    @RequestLine("POST /users")
    UserVO createUser(CreateUserReq req);

    @RequestLine("GET /users/{id}")
    UserVO getUser(@Param("id") String id);

    @RequestLine("PUT /users/{id}")
    UserVO updateUser(@Param("id") String id, UpdateUserReq req);

    @RequestLine("DELETE /users/{id}")
    void deleteUser(@Param("id") String id);

    @RequestLine("PATCH /users/{id}?name={name}")
    UserVO patchUser(@Param("id") String id, @Param("name") String name);

    @RequestLine("POST /users/login")
    Result<LoginResp> login(LoginReq req);

    @RequestLine("POST /users/register")
    Result<UserVO> register(RegisterReq req);

    @RequestLine("GET /users/{id}/profile")
    Result<UserProfileVO> getProfile(@Param("id") Long id);

    @RequestLine("PUT /users/{id}/profile")
    Result<UserProfileVO> updateProfile(@Param("id") Long id, UpdateProfileReq req);

    @RequestLine("GET /users/{id}/addresses")
    Result<List<AddressVO>> getAddresses(@Param("id") Long id);

    @RequestLine("POST /users/{id}/addresses")
    Result<AddressVO> addAddress(@Param("id") Long id, AddressReq req);

    @RequestLine("GET /users/{id}/favorites?pageNum={pageNum}&pageSize={pageSize}")
    Result<PageResult<ProductVO>> getFavorites(@Param("id") Long id,
                                                @Param("pageNum") int pageNum,
                                                @Param("pageSize") int pageSize);
}

class CreateUserReq {
    private String name;
    private String email;
    private String password;
    private String phone;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
    public String getPassword() { return password; }
    public void setPassword(String password) { this.password = password; }
    public String getPhone() { return phone; }
    public void setPhone(String phone) { this.phone = phone; }
}

class UpdateUserReq {
    private String name;
    private String email;
    private String phone;
    private String avatar;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
    public String getPhone() { return phone; }
    public void setPhone(String phone) { this.phone = phone; }
    public String getAvatar() { return avatar; }
    public void setAvatar(String avatar) { this.avatar = avatar; }
}

class UserVO {
    private Long id;
    private String name;
    private String email;
    private String phone;
    private String avatar;
    private Integer status;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
    public String getPhone() { return phone; }
    public void setPhone(String phone) { this.phone = phone; }
    public String getAvatar() { return avatar; }
    public void setAvatar(String avatar) { this.avatar = avatar; }
    public Integer getStatus() { return status; }
    public void setStatus(Integer status) { this.status = status; }
}

class LoginReq {
    private String email;
    private String password;

    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
    public String getPassword() { return password; }
    public void setPassword(String password) { this.password = password; }
}

class LoginResp {
    private String token;
    private Long userId;
    private String username;

    public String getToken() { return token; }
    public void setToken(String token) { this.token = token; }
    public Long getUserId() { return userId; }
    public void setUserId(Long userId) { this.userId = userId; }
    public String getUsername() { return username; }
    public void setUsername(String username) { this.username = username; }
}

class RegisterReq {
    private String username;
    private String email;
    private String password;
    private String phone;

    public String getUsername() { return username; }
    public void setUsername(String username) { this.username = username; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
    public String getPassword() { return password; }
    public void setPassword(String password) { this.password = password; }
    public String getPhone() { return phone; }
    public void setPhone(String phone) { this.phone = phone; }
}

class UserProfileVO {
    private Long id;
    private String username;
    private String email;
    private String phone;
    private String avatar;
    private String nickname;
    private String bio;
    private Integer gender;
    private String birthday;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getUsername() { return username; }
    public void setUsername(String username) { this.username = username; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
    public String getPhone() { return phone; }
    public void setPhone(String phone) { this.phone = phone; }
    public String getAvatar() { return avatar; }
    public void setAvatar(String avatar) { this.avatar = avatar; }
    public String getNickname() { return nickname; }
    public void setNickname(String nickname) { this.nickname = nickname; }
    public String getBio() { return bio; }
    public void setBio(String bio) { this.bio = bio; }
    public Integer getGender() { return gender; }
    public void setGender(Integer gender) { this.gender = gender; }
    public String getBirthday() { return birthday; }
    public void setBirthday(String birthday) { this.birthday = birthday; }
}

class UpdateProfileReq {
    private String nickname;
    private String bio;
    private Integer gender;
    private String birthday;
    private String avatar;

    public String getNickname() { return nickname; }
    public void setNickname(String nickname) { this.nickname = nickname; }
    public String getBio() { return bio; }
    public void setBio(String bio) { this.bio = bio; }
    public Integer getGender() { return gender; }
    public void setGender(Integer gender) { this.gender = gender; }
    public String getBirthday() { return birthday; }
    public void setBirthday(String birthday) { this.birthday = birthday; }
    public String getAvatar() { return avatar; }
    public void setAvatar(String avatar) { this.avatar = avatar; }
}

class AddressVO {
    private Long id;
    private String receiver;
    private String phone;
    private String province;
    private String city;
    private String district;
    private String detail;
    private Boolean isDefault;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getReceiver() { return receiver; }
    public void setReceiver(String receiver) { this.receiver = receiver; }
    public String getPhone() { return phone; }
    public void setPhone(String phone) { this.phone = phone; }
    public String getProvince() { return province; }
    public void setProvince(String province) { this.province = province; }
    public String getCity() { return city; }
    public void setCity(String city) { this.city = city; }
    public String getDistrict() { return district; }
    public void setDistrict(String district) { this.district = district; }
    public String getDetail() { return detail; }
    public void setDetail(String detail) { this.detail = detail; }
    public Boolean getIsDefault() { return isDefault; }
    public void setIsDefault(Boolean isDefault) { this.isDefault = isDefault; }
}

class AddressReq {
    private String receiver;
    private String phone;
    private String province;
    private String city;
    private String district;
    private String detail;
    private Boolean isDefault;

    public String getReceiver() { return receiver; }
    public void setReceiver(String receiver) { this.receiver = receiver; }
    public String getPhone() { return phone; }
    public void setPhone(String phone) { this.phone = phone; }
    public String getProvince() { return province; }
    public void setProvince(String province) { this.province = province; }
    public String getCity() { return city; }
    public void setCity(String city) { this.city = city; }
    public String getDistrict() { return district; }
    public void setDistrict(String district) { this.district = district; }
    public String getDetail() { return detail; }
    public void setDetail(String detail) { this.detail = detail; }
    public Boolean getIsDefault() { return isDefault; }
    public void setIsDefault(Boolean isDefault) { this.isDefault = isDefault; }
}

class ProductVO {
    private Long id;
    private String name;
    private String description;
    private java.math.BigDecimal price;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }
    public java.math.BigDecimal getPrice() { return price; }
    public void setPrice(java.math.BigDecimal price) { this.price = price; }
}
