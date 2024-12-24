# Kong Helpers for Go

A collection of utilities and helpers for working with Kong configuration in Go. This library provides convenient tools for generating YAML configuration templates and observing configuration file changes with robust error handling and debounce support.


---


## Features

### 1. YAML Template Generator

- Automatically generates YAML templates from Go structs with annotations.

- Supports `yaml` and `kong` struct tags.

- Handles nested structs, slices, and maps.

- Includes inline documentation via `help` tag.
  **Example Struct:**

```go
type Config struct {
   Host    string   `yaml:"host" default:"localhost" help:"The hostname"`
   Port    int      `yaml:"port" default:"8080" help:"The port number"`
   Options []string `yaml:"options" default:"1,2" help:"List of options"`
}
```
**Generated YAML:**

```yaml
host: "localhost" # The hostname
port: 8080        # The port number
options:          # List of options
  - 1
  - 2
```

### 2. Configuration File Watcher

- Observes configuration files for changes.

- Supports debounce to prevent frequent updates.

- Handles context-based shutdown.

- Customizable error handling and logging.
  **Example Usage:**

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

updates, err := ControlFileChanges(ctx, "./config.yaml", func() string {
    data, _ := os.ReadFile("./config.yaml")
    return string(data)
})

for event := range updates {
    fmt.Println("Old Config:", event.OldConfig)
    fmt.Println("New Config:", event.NewConfig)
}
```


---


## Installation


```sh
go get github.com/yourusername/kong-helpers
```


---


## Usage

### Generating YAML Template


```go
import "github.com/yourusername/kong-helpers/template"

template := template.GenerateYAMLTemplate(Config{})
fmt.Println(template)
```

### Watching Configuration Files


```go
import "github.com/yourusername/kong-helpers/watcher"

updates, err := watcher.ControlFileChanges(ctx, "./config.yaml", getConfigFn, watcher.WithDebounce(500*time.Millisecond))
```


---


## Customization

- **Debounce Duration:**  Prevents frequent updates during rapid file changes.

- **Error Handler:**  Custom callback for error handling.

- **Logger:**  Integrate your own logging solution.
  **Example:**

```go
watcher.ControlFileChanges(ctx, path, getConfigFn,
    watcher.WithDebounce(1*time.Second),
    watcher.WithLogger(myCustomLogger))
```



---


## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Authors
- **Vladislav Sysalov** - [vsysa](https://github.com/vsysa)
