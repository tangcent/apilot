package com.example.controller

import com.example.model.Category
import com.example.model.CategoryTreeVO
import com.example.model.CreateCategoryReq
import com.example.model.PageResult
import com.example.model.Result
import com.example.model.Status
import org.springframework.web.bind.annotation.CookieValue
import org.springframework.web.bind.annotation.DeleteMapping
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.PutMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestHeader
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.multipart.MultipartFile

@RestController
@RequestMapping("/api/categories")
class CategoryController(private val service: CategoryService) {

    @GetMapping("/tree")
    fun getCategoryTree(): Result<List<CategoryTreeVO>> {
        return Result(0, "ok", emptyList())
    }

    @GetMapping("/{id}/products")
    fun getProductsByCategory(
        @PathVariable id: Long,
        @RequestParam(defaultValue = "1") pageNum: Int,
        @RequestParam(defaultValue = "10") pageSize: Int,
    ): Result<PageResult<Any>> {
        return Result(0, "ok", PageResult(0, 1, 10, emptyList()))
    }

    @PostMapping
    fun create(@RequestBody req: CreateCategoryReq): Result<Void> {
        return Result(0, "ok", null)
    }

    @PutMapping("/{id}/sort")
    fun updateSort(@PathVariable id: Long, @RequestParam sort: Int): Result<Category> {
        return Result(0, "ok", Category(id, ""))
    }

    @DeleteMapping("/{id}")
    fun delete(@PathVariable id: Long): Result<Void> {
        return Result(0, "ok", null)
    }

    @GetMapping("/search")
    fun search(
        @RequestParam keyword: String,
        @RequestParam(required = false) status: Status?,
        @RequestHeader("X-Tenant") tenant: String,
        @CookieValue sessionId: String,
    ): Result<List<Category>> {
        return Result(0, "ok", emptyList())
    }

    @PostMapping("/upload")
    fun upload(@RequestParam("file") file: MultipartFile): Result<String> {
        return Result(0, "ok", "")
    }
}
