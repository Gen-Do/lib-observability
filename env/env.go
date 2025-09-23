// Package env предоставляет утилиты для работы с переменными окружения
package env

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ParseableType интерфейс для типов, которые можно парсить из строки
type ParseableType interface {
	~int | ~int64 | ~float64 | ~bool | ~string
}

// EnvName представляет тип окружения
type EnvName string

const (
	// EnvProd продакшн окружение
	EnvProd EnvName = "prod"
	// EnvStaging тестовое окружение
	EnvStaging EnvName = "staging"
	// EnvLocal локальное окружение
	EnvLocal EnvName = "local"
)

// LoadEnvFiles загружает переменные окружения из файлов в указанном порядке.
// Каждый последующий файл переопределяет значения из предыдущих.
// Отсутствие файлов не вызывает ошибку.
// Функция не использует логгер чтобы избежать циклической зависимости,
// так как логгер сам может зависеть от переменных окружения.
func LoadEnvFiles() {
	envFiles := []string{".env.paas", ".env.override"}

	for i, filename := range envFiles {
		var err error
		if i == 0 {
			// Первый файл загружаем обычным способом
			err = godotenv.Load(filename)
		} else {
			// Последующие файлы загружаем с перезаписью
			err = godotenv.Overload(filename)
		}

		// Молчаливо игнорируем ошибки - отсутствие файлов не критично
		_ = err
	}
}

// Get возвращает значение переменной окружения указанного типа или дефолтное значение
// Поддерживает типы: int, int64, float64, bool, string
func Get[T ParseableType](key string, defaultValue T) T {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	var result any
	var err error

	switch any(defaultValue).(type) {
	case string:
		result = value
	case bool:
		switch strings.ToLower(value) {
		case "true", "1", "yes", "on":
			result = true
		case "false", "0", "no", "off":
			result = false
		default:
			return defaultValue
		}
	case int:
		result, err = strconv.Atoi(value)
	case int64:
		result, err = strconv.ParseInt(value, 10, 64)
	case float64:
		result, err = strconv.ParseFloat(value, 64)
	default:
		return defaultValue
	}

	if err != nil {
		return defaultValue
	}

	return result.(T)
}

// GetString возвращает значение строковой переменной окружения или дефолтное значение
// Использует generic функцию Get[string]
func GetString(key, defaultValue string) string {
	return Get(key, defaultValue)
}

// GetBool возвращает значение булевой переменной окружения или дефолтное значение
// Поддерживает значения: true/false, 1/0, yes/no, on/off (регистронезависимо)
// Использует generic функцию Get[bool]
func GetBool(key string, defaultValue bool) bool {
	return Get(key, defaultValue)
}

// GetInt возвращает значение целочисленной переменной окружения или дефолтное значение
// Использует generic функцию Get[int]
func GetInt(key string, defaultValue int) int {
	return Get(key, defaultValue)
}

// GetInt64 возвращает значение целочисленной переменной окружения или дефолтное значение
// Использует generic функцию Get[int64]
func GetInt64(key string, defaultValue int64) int64 {
	return Get(key, defaultValue)
}

// GetFloat64 возвращает значение переменной окружения типа float64 или дефолтное значение
// Использует generic функцию Get[float64]
func GetFloat64(key string, defaultValue float64) float64 {
	return Get(key, defaultValue)
}

// GetDuration возвращает значение переменной окружения типа time.Duration или дефолтное значение
// Поддерживает форматы: "300ms", "1.5h", "2h45m", etc.
func GetDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed
	}
	return defaultValue
}

// GetServiceName возвращает имя сервиса из переменной окружения SERVICE_NAME
func GetServiceName() string {
	return GetString("SERVICE_NAME", "")
}

// GetServiceVersion возвращает версию сервиса из переменной окружения SERVICE_VERSION
// По умолчанию возвращает "v0.0.1"
func GetServiceVersion() string {
	return GetString("SERVICE_VERSION", "v0.0.1")
}

// GetEnvName возвращает тип окружения из переменной ENV_NAME
// По умолчанию возвращает EnvLocal
func GetEnvName() EnvName {
	envName := GetString("ENV_NAME", string(EnvLocal))
	switch EnvName(strings.ToLower(envName)) {
	case EnvProd:
		return EnvProd
	case EnvStaging:
		return EnvStaging
	case EnvLocal:
		return EnvLocal
	default:
		return EnvLocal
	}
}
