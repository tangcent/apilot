package com.example.controller

import com.example.model.CreateUserReq
import com.example.model.PageResult
import com.example.model.Result
import com.example.model.UpdateUserReq
import com.example.model.User
import org.springframework.web.bind.annotation.DeleteMapping
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.PutMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/users")
class UserController {

    @GetMapping
    fun listUsers(
        @RequestParam(required = false) keyword: String?,
        @RequestParam(required = false) status: Int?,
        @RequestParam(defaultValue = "1") pageNum: Int,
        @RequestParam(defaultValue = "20") pageSize: Int,
    ): Result<PageResult<User>> {
        return Result(0, "ok", PageResult(0, 1, 20, emptyList()))
    }

    @GetMapping("/{id}")
    fun getUser(@PathVariable id: Long): Result<User> {
        return Result(0, "ok", null)
    }

    @PostMapping
    fun createUser(@RequestBody req: CreateUserReq): Result<User> {
        return Result(0, "ok", null)
    }

    @PutMapping("/{id}")
    fun updateUser(@PathVariable id: Long, @RequestBody req: UpdateUserReq): Result<User> {
        return Result(0, "ok", null)
    }

    @DeleteMapping("/{id}")
    fun deleteUser(@PathVariable id: Long): Result<Void> {
        return Result(0, "ok", null)
    }

    @GetMapping("/{id}/orders")
    fun listOrders(
        @PathVariable id: Long,
        @RequestParam(required = false) status: Int?,
        @RequestParam(defaultValue = "1") pageNum: Int,
        @RequestParam(defaultValue = "10") pageSize: Int,
    ): Result<PageResult<Any>> {
        return Result(0, "ok", PageResult(0, 1, 10, emptyList()))
    }

    @GetMapping("/check-username")
    fun checkUsername(@RequestParam username: String): Result<Boolean> {
        return Result(0, "ok", true)
    }
}
