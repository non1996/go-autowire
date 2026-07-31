# go-autowire

`go-autowire` 是一个基于反射的 Go 依赖注入容器，提供结构体组件、接口实现选择、配置属性、环境变量、条件组件、配置派生 Bean，以及已有对象注入。

运行完整示例：

```bash
go run ./example
```

验证所有公开功能与并发安全：

```bash
go test -race ./...
```

## 基础组件

在包的 `init` 中注册组件。`Component[C]` 的泛型参数必须是结构体值类型；容器实际保存和注入 `*C`。

```go
type Repository interface {
    Find(id string) string
}

type repository struct{}

func (*repository) Find(id string) string { return id }

func init() {
    autowire.Register(
        autowire.Component[repository]().
            Implement(autowire.TypeOf[Repository]()).
            Register(),
    )
}
```

使用 `GetComponent[T]()` 按类型取唯一生效组件，或使用 `GetComponentByName[T]("alias")` 按别名取得组件。缺省为必填；传入 `false` 时未找到会返回 `T` 的零值。

```go
repo := autowire.GetComponent[Repository]()
optional := autowire.GetComponentByName[*repository]("missing", false)
_ = optional
_ = repo
```

同类型有多个候选时，`Primary(true)`（默认）决定优先候选；需要多个非主候选时应使用 `Alias` 和 `qualifier` 显式选择。`Implement` 的目标必须是接口，且组件指针必须实现它。

## 字段注入

支持 `autowire`、`value`、`env` 三类字段标签，每个字段只能声明其中一种。字段可以是非导出字段。

```go
type Service struct {
    repo     Repository `autowire:"true"`
    readonly Repository `autowire:"true" qualifier:"readonlyRepo" required:"false"`
    port     int        `value:"server/Port"`
    region   string     `env:"APP_REGION" default:"cn"`
}
```

- `autowire:"true"`：按字段类型注入组件；`qualifier` 指定组件别名。
- `value:"scope/key"`：注入配置属性；也接受 `scope.key`，其中首个分隔符前是 scope。
- `env:"NAME"`：注入环境变量。`default:"value"` 在环境变量缺失时使用默认值，并视为可满足的依赖。
- `required` 缺省为 `true`。`required:"false"` 的缺失组件/属性保持字段零值。
- 属性同类型值直接赋值；字符串可转换到 `string`、布尔、整数、无符号整数、浮点数和逗号分隔的切片。不可转换会报出字段和来源。

也可以注入未注册的已有对象：

```go
app := &Service{}
autowire.Inject(app)
```

## 配置、属性与 Bean

只有标记了 `Configuration()` 的组件可以声明 `Properties` 和 `Beans`。属性对象会在首次访问指定 scope 时构建一次；对象的导出字段会被展开，嵌套字段以点连接。

```go
type Settings struct {
    Port int
    Feature struct {
        Enabled bool
    }
}

type Config struct {
    settings Settings
}

type Greeting interface {
    Value() string
}

type greeting struct{}

func (greeting) Value() string { return "hello" }

func registerConfig() {
    autowire.Register(
        autowire.Component[Config]().
            Configuration().
            Properties(autowire.Property[Config]("server", func(c *Config) Settings {
                return c.settings
            })).
            Beans(autowire.Bean[Config, Greeting]("greeting", func(*Config) Greeting {
                return greeting{}
            })).
            PostConstruct(func(c *Config) error {
                c.settings.Port = 8080
                c.settings.Feature.Enabled = true
                return nil
            }).
            Register(),
    )
}
```

上例可通过 `value:"server/Port"` 和 `value:"server/Feature.Enabled"` 注入配置。每个容器中同一个属性 scope 只能注册一次；重复注册会 panic，而不会静默覆盖。

`Bean` 可使用 `PrimaryBean`、`ImplementBean` 和 `ConditionBean` 调整候选优先级、声明接口实现和设置条件。

## 条件组件

`Condition("scope/key=value")` 仅在属性值匹配时启用组件；按类型查询、按别名查询和作为其他组件依赖时使用同一规则。未满足条件的必填依赖会按“未找到组件”处理，可选依赖则为零值。

```go
autowire.Component[FeatureService]().
    Condition("server/Feature.Enabled=true").
    Register()
```

## 容器与生命周期

`Context()` 返回默认容器。使用 `NewContext()` 可创建隔离容器，适合测试或多应用配置：

```go
ctx := autowire.NewContext()
ctx.Register(autowire.Component[repository]().Register())
repo := autowire.GetComponentFrom[*repository](ctx)
_ = repo
```

组件采用惰性单例：首次构建会执行字段注入，再执行 `PostConstruct`。容器将注册过程与查询过程同步，且首次构建只会运行一次；构建失败会缓存带组件别名的错误，后续查询返回同一失败，而不会误报循环依赖。
