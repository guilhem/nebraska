package nebraska_test

import (
	"context"
	"fmt"

	nebraska "github.com/kinvolk/nebraska/go-sdk"
	"github.com/labstack/gommon/log"
)

func ExampleNebraska() {
	// Output:CompileTest!

	cnf := nebraska.Config{
		ServerURL: "http://localhost:8000",
	}
	n, err := nebraska.New(cnf)
	if err != nil {
		log.Error("client setup error:", err)
		return
	}
	ctx := context.Background()

	app, err := n.CreateApp(ctx, nebraska.AppConfig{
		Name: "testApp123",
	}, nil)
	if err != nil {
		log.Error("create app error:", err)
		return
	}
	fmt.Printf("App:\n%+v\n", app.Props())

	app, err = app.Update(ctx, nebraska.AppConfig{
		Name: "testAppXYZ",
	})
	if err != nil {
		log.Error("update app error:", err)
	}
	fmt.Printf("Updated App:\n%+v\n", app.Props())

	count := 0
	for {
		fmt.Println(count)
		count += 1
		group, err := app.CreateGroup(ctx, nebraska.GroupConfig{
			Name:                      fmt.Sprintf("Group %d", count),
			PolicyMaxUpdatesPerPeriod: count,
			PolicyPeriodInterval:      "1 hours",
			PolicyUpdateTimeout:       "1 hours",
		})
		if err != nil {
			fmt.Println("create group error", count, err)
			return
		}
		if count == 20 {
			break
		}
		fmt.Println("Group:", group.Props().Name)
	}

	aGroups, err := app.Groups(ctx)
	if err != nil {
		fmt.Println("app groups error", err)
		return
	}
	for _, grp := range aGroups {
		_, err = grp.Update(ctx, nebraska.GroupConfig{
			Name:                      "test " + grp.Props().Name,
			PolicyMaxUpdatesPerPeriod: grp.Props().PolicyMaxUpdatesPerPeriod,
			PolicyPeriodInterval:      grp.Props().PolicyPeriodInterval,
			PolicyUpdateTimeout:       grp.Props().PolicyUpdateTimeout,
		})
		if err != nil {
			return
		}
	}
}
