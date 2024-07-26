package models

type ConfigModel struct {
	Rules    []RuleModel   `json:"rules,omitempty" yaml:"rules,omitempty"`
	Settings SettingsModel `json:"settings,omitempty" yaml:"settings,omitempty"`
	Routes   []RouteModel  `json:"routes,omitempty" yaml:"routes,omitempty"`
	Vlans    []VlanModel   `json:"vlans,omitempty" yaml:"vlans,omitempty"`
}

type SettingsModel struct {
	TableHardSync []int `json:"table-hard-sync,omitempty" yaml:"table-hard-sync,omitempty"`
}

type RouteModel struct {
	To       string `json:"to,omitempty" yaml:"to,omitempty"`
	Via      string `json:"via,omitempty" yaml:"via,omitempty"`
	Table    int    `json:"table,omitempty" yaml:"table,omitempty"`
	Dev      string `json:"dev,omitempty" yaml:"dev,omitempty"`
	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	OnLink   bool   `json:"on-link,omitempty" yaml:"on-link,omitempty"`
	Scope    string `json:"scope,omitempty" yaml:"scope,omitempty"`
}

type RuleModel struct {
	From  string `json:"from,omitempty" yaml:"from,omitempty"`
	Table int    `json:"table,omitempty" yaml:"table,omitempty"`
}

type VlanModel struct {
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
	Link     string `json:"link,omitempty" yaml:"link,omitempty"`
	ID       int    `json:"id,omitempty" yaml:"id,omitempty"`
	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

func (in *ConfigModel) DeepCopyInto(out *ConfigModel) {
	*out = *in
}

// Merge c2 into C1
func (c1 *ConfigModel) Merge(c2 *ConfigModel) {

}
