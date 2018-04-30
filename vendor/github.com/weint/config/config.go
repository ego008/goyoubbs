package config

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
)

// Engine inclue data store interface.
type Engine struct {
	data map[interface{}]interface{}
}

// New object as default config interface.
func New(defaultConfig ...map[interface{}]interface{}) *Engine {
	if len(defaultConfig) > 0 {
		return &Engine{data: defaultConfig[0]}
	}

	return &Engine{}
}

// Load config file
func (c *Engine) Load(path string) error {
	if c.data == nil {
		c.data = make(map[interface{}]interface{})
	}

	ext := c.guessFileType(path)

	if ext == "" {
		return errors.New("Can not load " + path + " config")
	}

	return c.loadFromYaml(path)
}

func (c *Engine) guessFileType(path string) string {
	// guess from file extension
	s := strings.Split(path, ".")
	ext := s[len(s)-1]

	switch ext {
	case "yaml", "yml":
		return "yaml"
	}

	return ""
}

func (c *Engine) loadFromYaml(path string) error {

	yamlS, readErr := ioutil.ReadFile(path)

	if readErr != nil {
		return readErr
	}

	err := yaml.Unmarshal(yamlS, &c.data)

	if err != nil {
		return errors.New("Can not parse " + path + " config")
	}

	return nil
}

// Get value from config file.
func (c *Engine) Get(name string) interface{} {
	path := strings.Split(name, ".")

	data := c.data
	for key, value := range path {
		v, ok := data[value]

		if !ok {
			break
		}

		if (key + 1) == len(path) {
			return v
		}

		if reflect.TypeOf(v).String() == "map[interface {}]interface {}" {
			data = v.(map[interface{}]interface{})
		}
	}

	return nil
}

// GetString value from config file and return string format.
func (c *Engine) GetString(name string) string {
	value := c.Get(name)

	switch value := value.(type) {
	case string:
		return value
	case bool, float64, int:
		return fmt.Sprint(value)
	default:
		return ""
	}
}

// GetInt value from config file and return integer format.
func (c *Engine) GetInt(name string) int {
	value := c.Get(name)

	switch value := value.(type) {
	case string:
		i, _ := strconv.Atoi(value)
		return i
	case int:
		return value
	case bool:
		if value {
			return 1
		}
		return 0
	case float64:
		return int(value)
	default:
		return 0
	}
}

// GetBool value from config file and return boolean format.
func (c *Engine) GetBool(name string) bool {
	value := c.Get(name)

	switch value := value.(type) {
	case string:
		str, _ := strconv.ParseBool(value)
		return str
	case int:
		if value != 0 {
			return true
		}

		return false
	case bool:
		return value
	case float64:
		if value != 0.0 {
			return true
		}

		return false
	default:
		return false
	}
}

// GetFloat64 value from config file and return float64 format.
func (c *Engine) GetFloat64(name string) float64 {
	value := c.Get(name)

	switch value := value.(type) {
	case string:
		str, _ := strconv.ParseFloat(value, 64)
		return str
	case int:
		return float64(value)
	case bool:
		if value {
			return float64(1)
		}

		return float64(0)
	case float64:
		return value
	default:
		return float64(0)
	}
}

// GetStruct support user defined structure
func (c *Engine) GetStruct(name string, s interface{}) interface{} {
	d := c.Get(name)

	switch d.(type) {
	case string:
		c.setField(s, name, d)
	case map[interface{}]interface{}:
		c.mapToStruct(d.(map[interface{}]interface{}), s)
	}

	return s
}

func (c *Engine) mapToStruct(m map[interface{}]interface{}, s interface{}) interface{} {

	for key, value := range m {
		switch key.(type) {
		case string:
			c.setField(s, key.(string), value)
		}
	}

	return s
}

func (c *Engine) setField(obj interface{}, name string, value interface{}) error {
	structValue := reflect.Indirect(reflect.ValueOf(obj))
	structFieldValue := structValue.FieldByName(name)

	if !structFieldValue.IsValid() {
		return fmt.Errorf("No such field: %s in obj", name)
	}

	if !structFieldValue.CanSet() {
		return fmt.Errorf("Cannot set %s field value", name)
	}

	structFieldType := structFieldValue.Type()
	val := reflect.ValueOf(value)

	if structFieldType.Kind() == reflect.Struct && val.Kind() == reflect.Map {
		vint := val.Interface()

		switch vint.(type) {
		case map[interface{}]interface{}:
			for key, value := range vint.(map[interface{}]interface{}) {
				c.setField(structFieldValue.Addr().Interface(), key.(string), value)
			}
		case map[string]interface{}:
			for key, value := range vint.(map[string]interface{}) {
				c.setField(structFieldValue.Addr().Interface(), key, value)
			}
		}

	} else {
		if structFieldType != val.Type() {
			return errors.New("Provided value type didn't match obj field type")
		}

		structFieldValue.Set(val)
	}

	return nil
}
