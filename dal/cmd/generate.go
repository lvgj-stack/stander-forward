package main

import (
	"strings"

	"github.com/heyhip/frog"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"

	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/model"
)

// generate code
func main() {
	var DB *gorm.DB

	model.InitMysql(&config.Database{
		Username: "root",
		Password: "Lgj0873967111...",
		Addr:     "alihk.byte.gs:3306",
		DBName:   "stander",
	})
	DB = model.GetDb()
	// specify the output directory (default: "./query")
	// ### if you want to query without context constrain, set mode gen.WithoutContext ###
	g := gen.NewGenerator(gen.Config{
		Mode:         gen.WithDefaultQuery,
		OutPath:      "../../pkg/model/dal",
		ModelPkgPath: "../../pkg/model/entity",
		/* Mode: gen.WithoutContext,*/
		// if you want the nullable field generation property to be pointer type, set FieldNullable true
		FieldNullable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})
	g.WithJSONTagNameStrategy(func(columnName string) (tagContent string) {
		f := func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToLower(s[:1]) + s[1:]
		}

		return f(frog.Case2Camel(columnName))
	})

	// reuse the database connection in Project or create a connection here
	// if you want to use GenerateModel/GenerateModelAs, UseDB is necessary, or it will panic
	g.UseDB(DB)
	node := g.GenerateModel("nodes")
	trafficPlan := g.GenerateModel("traffic_plan")
	udt := g.GenerateModel("user_daily_traffic")
	ncm := g.GenerateModel("node_chain_mappings")
	chain := g.GenerateModel("chains", gen.FieldRelate(field.HasOne, "Node", node,
		&field.RelateConfig{
			GORMTag: field.GormTag{"references": []string{"NodeID"}, "foreignKey": []string{"ID"}},
		}))
	user := g.GenerateModel("user", gen.FieldRelate(field.HasOne, "TrafficPlan", trafficPlan,
		&field.RelateConfig{
			GORMTag: field.GormTag{"references": []string{"PlanID"}, "foreignKey": []string{"ID"}},
		}))
	rule := g.GenerateModel("rules",
		gen.FieldRelate(field.HasOne, "Node", node,
			&field.RelateConfig{
				GORMTag: field.GormTag{"references": []string{"NodeID"}, "foreignKey": []string{"ID"}},
			}),
		gen.FieldRelate(field.HasOne, "Chain", chain,
			&field.RelateConfig{
				GORMTag: field.GormTag{"references": []string{"ChainID"}, "foreignKey": []string{"ID"}},
			}),
	)
	urcm := g.GenerateModel("user_role_chain_mappings", gen.FieldRelate(field.HasOne, "Chain", chain,
		&field.RelateConfig{
			GORMTag: field.GormTag{"references": []string{"ChainID"}, "foreignKey": []string{"ID"}},
		}))
	urnm := g.GenerateModel("user_role_node_mappings", gen.FieldRelate(field.HasOne, "Node", node,
		&field.RelateConfig{
			GORMTag: field.GormTag{"references": []string{"NodeID"}, "foreignKey": []string{"ID"}},
		}))
	chainGroups := g.GenerateModel("chain_groups", gen.FieldRelate(field.HasOne, "Chain", chain,
		&field.RelateConfig{
			GORMTag: field.GormTag{"references": []string{"ChainID"}, "foreignKey": []string{"ID"}},
		}), gen.FieldType("backup", "bool"), gen.FieldGenType("backup", "Bool"))
	g.ApplyBasic(
		chain,
		node,
		rule,
		urcm,
		urnm,
		udt,
		trafficPlan,
		user,
		ncm,
		chainGroups,
	)

	g.Execute()
}
