package models

import "fmt"

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

func MergeConfigModels(c1 *ConfigModel, c2 *ConfigModel) ConfigModel {
	var mergedConfig ConfigModel

	ruleMap := make(map[string]bool)
	routeMap := make(map[string]bool)
	vlanMap := make(map[string]bool)
	tableHardSyncMap := make(map[int]bool)

	// Helper function to add unique rules
	addRules := func(rules []RuleModel) {
		for _, rule := range rules {
			key := fmt.Sprintf("%s-%d", rule.From, rule.Table)
			if !ruleMap[key] {
				mergedConfig.Rules = append(mergedConfig.Rules, rule)
				ruleMap[key] = true
			}
		}
	}

	// Helper function to add unique table hard sync settings
	addTableHardSync := func(syncs []int) {
		for _, sync := range syncs {
			if !tableHardSyncMap[sync] {
				mergedConfig.Settings.TableHardSync = append(mergedConfig.Settings.TableHardSync, sync)
				tableHardSyncMap[sync] = true
			}
		}
	}

	// Helper function to add unique routes
	addRoutes := func(routes []RouteModel) {
		for _, route := range routes {
			key := fmt.Sprintf("%s-%s-%d-%s-%s-%t-%s", route.To, route.Via, route.Table, route.Dev, route.Protocol, route.OnLink, route.Scope)
			if !routeMap[key] {
				mergedConfig.Routes = append(mergedConfig.Routes, route)
				routeMap[key] = true
			}
		}
	}

	// Helper function to add unique VLANs
	addVlans := func(vlans []VlanModel) {
		for _, vlan := range vlans {
			key := fmt.Sprintf("%s-%s-%d-%s", vlan.Name, vlan.Link, vlan.ID, vlan.Protocol)
			if !vlanMap[key] {
				mergedConfig.Vlans = append(mergedConfig.Vlans, vlan)
				vlanMap[key] = true
			}
		}
	}

	// Add unique elements from c1
	addRules(c1.Rules)
	addTableHardSync(c1.Settings.TableHardSync)
	addRoutes(c1.Routes)
	addVlans(c1.Vlans)

	// Add unique elements from c2
	addRules(c2.Rules)
	addTableHardSync(c2.Settings.TableHardSync)
	addRoutes(c2.Routes)
	addVlans(c2.Vlans)

	return mergedConfig
}
