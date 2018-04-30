
## Usage

create new `yaml` config as below and save as `config.yaml` file:

```yaml
mode: develop
addr: localhost
port: 8080
enable: true
pubdir: static
lang: zh-tw
db:
  driver: MySQL
  Protocol: tcp
  host: localhost
  username: root
  password: 123456
  database: test
  enable: true
  port: 3306
  isTrue: "F"
  isFalse: 1
  average: 2.0
  coverage: 2.1
  gpa: "3.7"
  floatzero: 0.0
```

example.go

```go
c := &config.Engine{}
c.Load("config.yaml")

// Result: Mysql (string)
fmt.Println(c.Get("db.driver"))
// Result: 3306 (int)
fmt.Println(c.Get("db.port"))
// Result: true (bool)
fmt.Println(c.Get("db.enable"))
// Result: 2.0 (float64)
fmt.Println(c.Get("db.average"))

// Result: Mysql (string)
fmt.Println(c.GetString("db.driver"))
// Result: 3306 (string)
fmt.Println(c.GetString("db.port"))
// Result: true (string)
fmt.Println(c.GetString("db.enable"))
// Result: 2 (string)
fmt.Println(c.GetString("db.average"))
// Result: 2.1 (string)
fmt.Println(c.GetString("db.coverage"))

// Result: 0 (int)
fmt.Println(c.GetInt("db.driver"))
// Result: 3306 (int)
fmt.Println(c.GetInt("db.port"))
// Result: 1 (int)
fmt.Println(c.GetInt("db.enable"))
// Result: 2 (int)
fmt.Println(c.GetInt("db.average"))
// Result: 2 (int)
fmt.Println(c.GetInt("db.coverage"))

// Result: false (bool)
fmt.Println(c.GetBool("db.driver"))
// Result: true (bool)
fmt.Println(c.GetBool("db.port"))
// Result: true (bool)
fmt.Println(c.GetBool("db.enable"))
// Result: true (bool)
fmt.Println(c.GetBool("db.average"))
// Result: true (bool)
fmt.Println(c.GetBool("db.coverage"))
```

### Get Struct

```go
type SessionConfig struct {
	CookieName      string
	SessionIDLength int
	CookieLifeTime  int // cookie expire time
	ExpireTime      int // time for destroy from GC
	GCTime          int // GC frequency
	Domain          string
}
```

We can also defined in `config.yaml`, and use `GetStruct` function to get SessionConfig struct, example:

```yaml
mode: develop
addr: localhost
port: 8080
enable: true
pubdir: static
lang: zh-cn
db:
  driver: MySQL
  Protocol: tcp
  host: localhost
  username: root
  password: 123456
  database: test
  enable: true
  port: 3306
  isTrue: "F"
  isFalse: 1
sessions:
  CookieName: "test-cookie"
  SessionIDLength: 10
  CookieLifeTime: 3600
  ExpireTime: 3600
  GCTime: 10
  Domain: example.com
```

example.go

```go
c := &config.Engine{}
c.Load("config.yaml")

// Result: &{
//  CookieName:test-cookie
//  SessionIDLength:10
//  CookieLifeTime:3600
//  ExpireTime:3600
//  GCTime:10
//  Domain:example.com
//}
fmt.Printf("%+v", c.GetStruct("sessions", &sessions.SessionConfig{}))
```


