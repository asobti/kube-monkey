package config

import (
	"testing"

	"kube-monkey/internal/pkg/config/param"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestValidateConfigs(t *testing.T) {
	SetDefaults()

	assert.Nil(t, ValidateConfigs())

	viper.Set(param.RunHour, 24)
	assert.EqualError(t, ValidateConfigs(), "RunHour: "+param.RunHour+" is outside valid range of [0,23]")
	viper.Set(param.RunHour, 23)

	viper.Set(param.StartHour, 24)
	assert.EqualError(t, ValidateConfigs(), "StartHour: "+param.StartHour+" is outside valid range of [0,23]")
	viper.Set(param.StartHour, 23)

	viper.Set(param.EndHour, 24)
	assert.EqualError(t, ValidateConfigs(), "EndHour: "+param.EndHour+" is outside valid range of [0,23]")
	viper.Set(param.EndHour, 23)

	viper.Set(param.StartHour, 23)
	assert.EqualError(t, ValidateConfigs(), "StartHour: "+param.StartHour+" must be less than "+param.EndHour)
	viper.Set(param.StartHour, 22)

	viper.Set(param.RunHour, 23)
	assert.EqualError(t, ValidateConfigs(), "RunHour: "+param.RunHour+" should be less than "+param.StartHour)

}

func TestIsValidHour(t *testing.T) {
	for i := 0; i <= 23; i++ {
		assert.True(t, IsValidHour(i))
	}
	assert.False(t, IsValidHour(24))
}

func TestIsValidHeader(t *testing.T) {
	header := "header1Key:header1Value"
	assert.True(t, isValidHeader(header))

	header = "header1/Key:header1/Value"
	assert.True(t, isValidHeader(header))

	header = "header1:{$env:VARIABLE_NAME}"
	assert.True(t, isValidHeader(header))

	header = "header1Key"
	assert.False(t, isValidHeader(header))

	header = "header1Key:"
	assert.False(t, isValidHeader(header))
}
