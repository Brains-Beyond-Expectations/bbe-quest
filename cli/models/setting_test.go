package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ChartSetting_EntryLabel_IsTheUniqueField(t *testing.T) {
	setting := ChartSetting{Fields: []SettingField{{Name: "email"}, {Name: "username", Unique: true}}}

	assert.Equal(t, "alice", setting.EntryLabel(map[string]interface{}{"username": "alice", "email": "alice@example.com"}))
}

func Test_ChartSetting_EntryLabel_IsTheNameWithoutFields(t *testing.T) {
	setting := ChartSetting{Catalog: &SettingCatalog{Type: "prowlarr-indexers"}}

	assert.Equal(t, "1337x", setting.EntryLabel(map[string]interface{}{"name": "1337x", "definition": "1337x"}))
}
