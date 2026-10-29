package com.example

import com.example.model.Result
import com.example.model.User
import org.springframework.web.bind.annotation.DeleteMapping
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/users")
class UserController {

    @GetMapping("/{id}")
    fun getUser(@PathVariable id: Long): Result<User> {
        return Result(0, "", null)
    }

    @DeleteMapping("/{id}")
    fun deleteUser(@PathVariable id: Long): Result<Void> {
        return Result(0, "", null)
    }

    @GetMapping("/{id}/profile")
    fun getProfile(@PathVariable id: Long): Result<User> {
        return Result(0, "", null)
    }

    @GetMapping("/check-username")
    fun checkUsername(@RequestParam username: String): Result<Boolean> {
        return Result(0, "", false)
    }
}
