# Java/Kotlin 源文件解析策略研究

> Date: 2026-04-10
> Status: Research completed
> Decision: 推荐使用 Tree-sitter + go-tree-sitter (带 CGO)

---

## 问题背景

APilot 需要在 Go 中解析 Java/Kotlin 源文件，提取 Spring MVC、JAX-RS、Feign 等 Web 框架的 API 端点信息。当前 `api-collector-java` 模块所有解析器都是 TODO 状态。

---

## 评估方案

### 方案 1: Tree-sitter-java + go-tree-sitter (CGO)

**实现方式:**
```bash
go get github.com/tree-sitter/go-tree-sitter
go get github.com/tree-sitter/tree-sitter/java
```

**优点:**
- ✅ 官方 Go 绑定支持：[tree-sitter/go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter)
- ✅ 完整的 Java 语法树，包括注释、类型推导
- ✅ 增量解析能力（虽然 APilot 不需要）
- ✅ 广泛使用在 Neovim、GitHub Code Search 等工具中
- ✅ Kotlin 支持（tree-sitter-kotlin 有 Go 绑定）
- ✅ 纯运行时依赖，无需 Java 环境或 javac
- ✅ 静态编译时已链接，解析速度快
- ✅ 与 Go 生态系统（gopls 等）的实践一致

**缺点:**
- ⚠️ 需要 CGO（C 绑定），影响交叉编译
- ⚠️ Go 二进制文件体积增大
- ⚠️ CGO 调试比纯 Go 更复杂
- ⚠️ 内存管理需要手动调用 Close（见文档）

**技术细节:**
- 需要正确释放资源：Parser、Tree、TreeCursor、Query、QueryCursor、LookaheadIterator
- CGO 交叉编译支持：需要目标平台的 C 编译器和 tree-sitter 库
- 分层编译策略见下方 "交叉编译解决方案"

**复杂度:** 中等（CGO 配置，但有成熟文档和示例）

---

### 方案 2: ANTLR4 Go 运行时

**实现方式:**
```bash
go get github.com/antlr/antlr4/v4  # 语法文件
go get github.com/antlr4-go/antlr   # Go 运行时
go get github.com/antlr/antlr4/grammars/java  # 预编译 Java 语法
```

**优点:**
- ✅ 纯 Go 运行时（antlr4-go）
- ✅ 预编译的语法可用（antlr4-grammars/java）
- ✅ 功能强大的解析器，支持复杂语法
- ✅ Go 运行时无 CGO 依赖

**缺点:**
- ⚠️ 语法文件仍需要 ANTLR 工具链（Java/Python）生成
- ⚠️ 代码生成环节可能需要 CGO 或外部工具
- ⚠️ 维护自定义语法时需要 ANTLR 工具链
- ⚠️ 学习曲线较陡峭
- ⚠️ Kotlin 支持需要额外语法文件

**技术细节:**
- 预编译语法：antlr4-grammars 项目提供 Go 版本的常用语法
- 运行时：antlr4-go 提供 Go 目标的解析器
- 代码生成：使用 ANTLR 工具从 .g4 文件生成 .go 文件

**复杂度:** 中高（依赖 ANTLR 工具链）

---

### 方案 3: Subprocess javac/kt 编译器

**实现方式:**
```go
exec.Command("javac", "-proc:only", "-Xprint:ast", sourceFile).Output()
// 或使用 com.sun.source.tree API
```

**优点:**
- ✅ 100% 准确（使用 javac 内部解析器）
- ✅ 支持所有 Java/Kotlin 语言特性
- ✅ 无 CGO 依赖

**缺点:**
- ❌ 需要 Java/JDK 运行时环境
- ❌ 跨平台兼容性差（依赖 JDK 安装）
- ❌ 性能开销（进程启动）
- ❌ 错误处理复杂（Java 异常到 Go 错误转换）
- ❌ 用户环境复杂性增加

**复杂度:** 低（简单 subprocess 调用，但运行时要求高）

---

### 方案 4: Pure Go regex/heuristic parser

**实现方式:**
```go
// 基于 @RestController, @GetMapping 等注解的简单模式匹配
re.MustCompile(`@GetMapping\("([^"]+)"\)`)
```

**优点:**
- ✅ 无 CGO 依赖
- ✅ 无外部依赖（JDK、javac 等）
- ✅ 简单、快速实现
- ✅ 跨平台编译无忧

**缺点:**
- ❌ 准确度有限（无法处理复杂语法、嵌套注解、类型信息）
- ❌ 维护困难（正则表达式复杂、难以理解）
- ❌ 容易遗漏边缘情况
- ❌ 无法提取完整信息（方法签名、参数类型、返回类型等）
- ❌ 不适合生产环境

**复杂度:** 低（实现简单，但功能有限）

---

### 方案 5: JavaParser 子进程

**实现方式:**
```go
exec.Command("java", "-jar", "javaparser-cli.jar", sourceFile).Output()
```

**优点:**
- ✅ 准确解析（使用 javaparser 或 JavaParser 库）
- ✅ 无 CGO 依赖

**缺点:**
- ❌ 需要 Java 运行时
- ❌ 需要分发额外 jar 文件
- ❌ 性能开销（进程启动）
- ❌ 增加部署复杂性

**复杂度:** 中等（需要构建 Java 侧工具）

---

## 决策矩阵

| 维度 | Tree-sitter | ANTLR4 | javac subprocess | Regex/heuristic | JavaParser subprocess |
|------|------------|---------|------------------|-----------------|---------------------|
| **准确度** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Go 纯度** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **性能** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **依赖复杂性** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Kotlin 支持** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ |
| **交叉编译** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **维护性** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ | ⭐⭐ | ⭐⭐⭐ |
| **学习曲线** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |

**总分:**
1. Tree-sitter: 38/40 (⭐⭐⭐⭐⭐ 首选)
2. ANTLR4: 35/40
3. javac subprocess: 32/40
4. JavaParser subprocess: 31/40
5. Regex/heuristic: 28/40

---

## 推荐方案：Tree-sitter-java + go-tree-sitter

### 推荐理由

1. **准确性与性能兼得**：完整的语法树，静态链接，快速解析
2. **Kotlin 一致支持**：tree-sitter-kotlin 有相同 Go 绑定
3. **工具链成熟**：在 Neovim、GitHub Code Search 等广泛使用
4. **运行时依赖最小**：只需 C 库，无 JDK/javac 需求
5. **社区支持**：丰富的文档和示例

### CGO 处理策略

**交叉编译解决方案：**

对于静态构建（如 Linux/Windows），使用分层编译：

```bash
# 1. 原生构建（macOS/ARM64，当前开发环境）
GOOS=darwin GOARCH=arm64 go build -o bin/apilot-darwin-arm64 ./apilot-cli

# 2. 容器构建（Linux AMD64）
docker run --rm -v $PWD:/app -w /app golang:1.22 bash -c "
  apt-get update && apt-get install -y build-essential
  go build -o bin/apilot-linux-amd64 ./apilot-cli
"

# 3. Windows 构建（使用 mingw）
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
CC=x86_64-w64-mingw32-gcc \
go build -o bin/apilot-windows-amd64.exe ./apilot-cli
```

**内存管理示例：**

```go
parser := tree_sitter.NewParser(unsafe.Pointer(lang))
defer parser.Close()  // 必须调用 Close

tree := parser.ParseString(sourceCode)
defer tree.Close()  // 必须调用 Close

cursor := tree.Walk()
defer cursor.Close()  // // 必须调用 Close
```

### 实现步骤

#### 1. 添加依赖

```bash
# api-collector-java/go.mod
go get github.com/tree-sitter/go-tree-sitter@latest
go get github.com/tree-sitter/tree-sitter/java@latest
go get github.com/tree-sitter/tree-sitter/kotlin@latest
```

#### 2. 创建解析器适配层

```go
// api-collector-java/parser/parser.go
package parser

import tree_sitter "github.com/tree-sitter/go-tree-sitter"

// Parser 封装 tree-sitter 解析
type Parser struct {
    lang      *tree_sitter.Language
    parser    *tree_sitter.Parser
}

func NewParser() (*Parser, error) {
    // 初始化 Java/Kotlin 语言
    // ... 错误处理
}

func (p *Parser) ParseFile(path string) (*tree_sitter.Tree, error) {
    // 读取文件并解析
    // ... 资源管理 defer tree.Close()
}

func (p *Parser) ExtractEndpoints(node *tree_sitter.Node) []ApiEndpoint {
    // 遍历 AST 提取 API 端点
    // 查找 @RestController, @GetMapping 等注解
}

func (p *Parser) Close() {
    p.parser.Close()
}
```

#### 3. 集成到框架解析器

```go
// api-collector-java/springmvc/parser.go
package springmvc

import "github.com/tangcent/apilot/api-collector-java/parser"

func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
    p, err := parser.NewParser()
    if err != nil {
        return nil, err
    }
    defer p.Close()

    // 遍历 .java 文件
    // 调用 p.ParseFile 和 p.ExtractEndpoints
}
```

#### 4. 更新 collector.go

```go
func (c *JavaCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
    // 1. 发现 .java / .kt 文件
    // 2. 初始化 parser.Parser
    // 3. 按框架委托解析
}
```

---

## 备选方案：ANTLR4 Go 运行时

如果 Tree-sitter 遇到 CGO 交叉编译障碍，可切换到 ANTLR4：

**实现步骤：**
1. `go get github.com/antlr/antlr4/v4`
2. `go get github.com/antlr/antlr4/grammars/java`
3. 使用预编译的 Java 语法构建解析器
4. 实现监听器提取注解和方法信息

**优点：** 无 CGO，纯 Go
**缺点：** 需要 ANTLR 工具链生成语法文件

---

## 次选方案：Pure Go Regex Parser（仅限 MVP）

如果时间紧迫，可先实现基于正则的简单解析器，后续升级到 Tree-sitter：

**适用场景：**
- 快速原型验证
- 资源受限项目
- MVP 阶段

**限制：**
- 仅提取基础信息（HTTP 方法、路径）
- 不处理复杂语法
- 不适合生产环境

---

## 实现路线图

### 阶段 1: 基础设施（Tree-sitter 集成）
- [ ] 添加 go-tree-sitter 依赖
- [ ] 创建 parser/parser.go 适配层
- [ ] 实现内存管理（defer Close）
- [ ] 添加 Java/Karvin 语言初始化

### 阶段 2: Spring MVC 解析器
- [ ] 实现 springmvc/parser.go
- [ ] 提取 @RestController 类
- [ ] 提取 @GetMapping/@PostMapping 等路由注解
- [ ] 提取方法参数、返回类型
- [ ] 添加测试用例

### 阶段 3: JAX-RS 解析器
- [ ] 实现 jaxrs/parser.go
- [ ] 提取 @Path, @GET/@POST 等注解
- [ ] 添加测试用例

### 阶段 4: Feign 客户端解析器
- [ ] 实现 feign/parser.go
- [ ] 提取 @FeignClient 接口
- [ ] 提取方法签名映射到 HTTP 童点
- [ ] 添加测试用例

### 阶段 5: Kotlin 支持
- [ ] 集成 tree-sitter-kotlin
- [ ] 测试 Kotlin Spring Boot 项目
- [ ] 验证类型推导准确性

### 阶段 6: Maven 集成
- [ ] 调用 maven-indexer-cli 依赖解析
- [ ] 解析 pom.xml/build.gradle
- [ ] 解析导入类型到完整类名

### 阶段 7: 交叉编译配置
- [ ] 配置 GitHub Actions 构建矩阵
- [ ] 添加容器构建（Linux/Windows）
- [ ] 验证所有平台二进制

---

## 相关资源

### Tree-sitter 相关
- [tree-sitter/go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter) - Go 绑定
- [tree-sitter/tree-sitter-java](https://github.com/tree-sitter/tree-sitter-java) - Java 语法
- [tree-sitter/tree-sitter-kotlin](https://github.com/tree-sitter/tree-sitter-kotlin) - Kotlin 语法
- [Tree-sitter Go 使用指南](https://dev.to/shrsv/tinkering-with-tree-sitter-using-go-4d8n)

### ANTLR4 相关
- [antlr4-go/antlr](https://github.com/antlr4-go/antlr) - Go 运行时
- [antlr4-grammars](https://gitee.com/runningnoob/antlr4-grammars) - 预编译语法
- [ANTLR Getting Started (Go)](https://deepwiki.com/antlr4-go/antlr/1.2-getting-started-with-antlr-go)

### Java/Kotlin 解析
- [javaparser](https://javaparser.org/) - Java 解析库
- [kotlinx/ast](https://github.com/kotlinx/ast) - Kotlin AST 库（仅 JVM）

### APilot 相关
- [architecture.md](./architecture.md) - 架构文档
- [contributing-collectors.md](./contributing-collectors.md) - Collector 贡献指南
- [api-collector-java/collector.go](../api-collector-java/collector.go) - Java Collector 骨架

---

## 开放问题

1. **CGO 交叉编译验证**：需要在实际 CI 环境验证所有平台构建
2. **Maven indexer 集成时序**：依赖解析与源文件解析的协调策略
3. **类型信息精度**：tree-sitter 的类型推导对泛型、继承的支持范围
4. **性能基线**：大型项目（1000+ Java 文）的解析性能

---

## 结论

**推荐方案：** Tree-sitter-java + go-tree-sitter（带 CGO）

**下一步：**
1. 实现 parser/parser.go 适配层
2. 编写 Spring MVC 解析器 PoC
3. 验证 CGO 交叉编译
4. 完善错误处理和测试覆盖

**预计时间线：**
- 阶段 1-2（MVP）：2-3 天
- 阶段 3-4（完整功能）：2-3 天
- 阶段 5-7（生产就绪）：3-4 天

**风险缓解：**
- CGO 跨平台构建：使用容器构建
- 性能问题：实现文件级缓存和并行解析
- 准确度问题：与 javaparser 子进程结果对比校验
