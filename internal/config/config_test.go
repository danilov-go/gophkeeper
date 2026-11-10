package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	optionSet  string = "Set"
	optionText string = "UnmarshalText"
	optionJSON string = "Unmarshal"
)

func TestNetAddress(t *testing.T) {
	type want struct {
		err  bool
		host string
		port int
		str  string
	}
	tests := []struct {
		name   string
		option string
		input  string
		exp    want
	}{
		{
			name:   "Set: положительный тест",
			option: optionSet,
			input:  "localhost:8080",
			exp: want{
				err:  false,
				host: "localhost",
				port: 8080,
				str:  "localhost:8080",
			},
		},
		{
			name:   "Set: нет порта",
			option: optionSet,
			input:  "localhost",
			exp: want{
				err: true,
			},
		},
		{
			name:   "Set: порт не число",
			option: optionSet,
			input:  "localhost:invalid",
			exp: want{
				err: true,
			},
		},

		{
			name:   "UnmarshalText: положительный тест",
			option: optionText,
			input:  "10.1.0.1:8080",
			exp: want{
				err:  false,
				host: "10.1.0.1",
				port: 8080,
				str:  "10.1.0.1:8080",
			},
		},
		{
			name:   "UnmarshalJSON: невалидный формат адреса внутри JSON",
			option: optionJSON,
			input:  `"invalid-address"`,
			exp: want{
				err: true,
			},
		},
		{
			name:   "UnmarshalJSON: тип не строка",
			option: optionJSON,
			input:  `12345`,
			exp: want{
				err: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var test NetAddress
			var err error
			switch tt.option {
			case optionSet:
				err = test.Set(tt.input)
			case optionText:
				err = test.UnmarshalText([]byte(tt.input))
			case optionJSON:
				err = json.Unmarshal([]byte(tt.input), &test)
			}
			if tt.exp.err {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.exp.host, test.Host)
			assert.Equal(t, tt.exp.port, test.Port)
			if tt.exp.str != "" {
				assert.Equal(t, tt.exp.str, test.String())
			}
		})
	}
}
