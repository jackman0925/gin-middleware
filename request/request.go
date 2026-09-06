// Package request 提供 HTTP 请求绑定、默认值填充和校验的辅助函数。
package request

import (
	"bytes"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	ginjson "github.com/gin-gonic/gin/codec/json"
)

var (
	// ErrInvalidTarget 表示目标不是指向 struct 的非 nil 指针，因而无法填充默认值。
	ErrInvalidTarget = errors.New("request target must be a non-nil pointer to a struct")
)

var timeType = reflect.TypeFor[time.Time]()
var durationType = reflect.TypeFor[time.Duration]()

// BindJSON 按“JSON 解码 → default 标签填充 → Gin 校验”的顺序处理请求。
// BindJSON decodes JSON, applies default tags, then invokes Gin validation.
//
// 与 c.ShouldBindJSON 不同，默认值会在校验前写入，因此字段可以同时使用
// default:"..." 和 binding:"required"。JSON 中明确传入的 false、0 和空字符串
// 会被保留，不会被默认值覆盖。请求体会缓存到 Gin 的 BodyBytesKey；如有需要，
// 后续仍可通过 c.ShouldBindBodyWithJSON 再次绑定。
func BindJSON(c *gin.Context, obj any) error {
	if err := validateTarget(obj); err != nil {
		return err
	}

	body, err := bodyBytes(c)
	if err != nil {
		return err
	}
	if err := decodeJSON(body, obj); err != nil {
		return err
	}

	present, err := jsonObject(body)
	if err != nil {
		return err
	}
	if err := setDefaults(reflect.ValueOf(obj).Elem(), present, ""); err != nil {
		return err
	}
	return binding.Validator.ValidateStruct(obj)
}

// SetReqDefaults 为 obj 中的零值字段应用 default 标签。
//
// 该函数适合未通过 JSON 填充的值。由于无法判断某个 JSON 字段是否由客户端传入，
// 它会将零值视作未设置。JSON 请求建议优先使用 BindJSON，因为 BindJSON 会保留
// 客户端明确传入的 0、false 和空字符串。
func SetReqDefaults(obj any) error {
	if err := validateTarget(obj); err != nil {
		return err
	}
	return setDefaults(reflect.ValueOf(obj).Elem(), nil, "")
}

func bodyBytes(c *gin.Context) ([]byte, error) {
	// 复用 Gin 的缓存，避免同一请求体被重复读取。
	if cached, exists := c.Get(gin.BodyBytesKey); exists {
		if body, ok := cached.([]byte); ok {
			return body, nil
		}
	}
	if c.Request == nil || c.Request.Body == nil {
		return nil, errors.New("invalid request body")
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Set(gin.BodyBytesKey, body)
	return body, nil
}

func decodeJSON(body []byte, obj any) error {
	// 对齐 Gin JSON binder 的全局解码选项。
	decoder := ginjson.API.NewDecoder(bytes.NewReader(body))
	if binding.EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if binding.EnableDecoderDisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	return decoder.Decode(obj)
}

func jsonObject(body []byte) (map[string]stdjson.RawMessage, error) {
	var object map[string]stdjson.RawMessage
	if err := stdjson.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	return object, nil
}

func validateTarget(obj any) error {
	value := reflect.ValueOf(obj)
	if !value.IsValid() || value.Kind() != reflect.Ptr || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return ErrInvalidTarget
	}
	return nil
}

func setDefaults(value reflect.Value, present map[string]stdjson.RawMessage, path string) error {
	typeOfValue := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldType := typeOfValue.Field(i)
		// 跳过未导出字段和不可写字段。
		if fieldType.PkgPath != "" || !field.CanSet() {
			continue
		}

		name, include := jsonFieldName(fieldType)
		if !include {
			continue
		}
		fieldPath := name
		if path != "" {
			fieldPath = path + "." + name
		}
		raw, supplied := lookupRaw(present, name)

		// 仅为未在 JSON 中出现且仍是零值的字段写入默认值。default:"-"
		// 是显式跳过标记。
		if defaultValue, hasDefault := fieldType.Tag.Lookup("default"); hasDefault && defaultValue != "-" && !supplied && field.IsZero() {
			if err := setDefault(field, defaultValue); err != nil {
				return fmt.Errorf("invalid default for field %s: %w", fieldPath, err)
			}
			continue
		}

		// 递归处理嵌套 struct；time.Time 作为标量值单独处理。
		if field.Kind() == reflect.Struct && field.Type() != timeType {
			child, err := rawObject(raw, supplied)
			if err != nil {
				return fmt.Errorf("invalid JSON value for field %s: %w", fieldPath, err)
			}
			if err := setDefaults(field, child, fieldPath); err != nil {
				return err
			}
		}
		if field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct && field.Elem().Type() != timeType {
			child, err := rawObject(raw, supplied)
			if err != nil {
				return fmt.Errorf("invalid JSON value for field %s: %w", fieldPath, err)
			}
			if err := setDefaults(field.Elem(), child, fieldPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	tag := strings.Split(field.Tag.Get("json"), ",")[0]
	if tag == "-" {
		return "", false
	}
	if tag != "" {
		return tag, true
	}
	return field.Name, true
}

func lookupRaw(values map[string]stdjson.RawMessage, name string) (stdjson.RawMessage, bool) {
	if values == nil {
		return nil, false
	}
	if raw, ok := values[name]; ok {
		return raw, true
	}
	for key, raw := range values {
		if strings.EqualFold(key, name) {
			return raw, true
		}
	}
	return nil, false
}

func rawObject(raw stdjson.RawMessage, supplied bool) (map[string]stdjson.RawMessage, error) {
	if !supplied || len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var object map[string]stdjson.RawMessage
	if err := stdjson.Unmarshal(raw, &object); err != nil {
		return nil, err
	}
	return object, nil
}

func setDefault(field reflect.Value, defaultValue string) error {
	// 指针默认值会先分配底层值，再按其实际类型解析。
	if field.Kind() == reflect.Ptr {
		field.Set(reflect.New(field.Type().Elem()))
		return setDefault(field.Elem(), defaultValue)
	}

	// time.Time 使用 RFC3339；time.Duration 使用 Go duration 格式，例如 5s。
	if field.Type() == timeType {
		value, err := time.Parse(time.RFC3339, defaultValue)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(value))
		return nil
	}
	if field.Type() == durationType {
		value, err := time.ParseDuration(defaultValue)
		if err != nil {
			return err
		}
		field.SetInt(int64(value))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultValue)
	case reflect.Bool:
		value, err := strconv.ParseBool(defaultValue)
		if err != nil {
			return err
		}
		field.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(defaultValue, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(defaultValue, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(value)
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(defaultValue, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(value)
	case reflect.Slice, reflect.Map, reflect.Array, reflect.Struct:
		// 复杂类型的默认值使用 JSON 字面量，例如 default:"[\"admin\"]"。
		if err := stdjson.Unmarshal([]byte(defaultValue), field.Addr().Interface()); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type %s", field.Type())
	}
	return nil
}
