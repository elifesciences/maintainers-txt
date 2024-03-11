package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wcharczuk/go-chart/v2"
)

func graph() {
	report_file := "report.json"
	report_bytes, err := os.ReadFile(report_file)
	panicOnErr(err, "reading report.json")

	dest := map[string][]string{}
	json.Unmarshal(report_bytes, &dest)

	// each slice of the chart is a person,
	// the value of each slice is the sum of each percentage
	// of each repository they are responsible for.

	person_value_idx := map[string]float64{}
	for _, person_list := range dest {
		num_people := float64(len(person_list))
		for _, person := range person_list {
			person_value, present := person_value_idx[person]
			if !present {
				person_value = 0
			}
			person_value += 1.0 / num_people
			person_value_idx[person] = person_value
		}
	}

	vals := []chart.Value{}
	for person, value := range person_value_idx {
		vals = append(vals, chart.Value{Value: value, Label: person})
	}

	pie := chart.PieChart{
		Width:  512,
		Height: 512,
		Values: vals,
	}

	f, _ := os.Create("output.svg")
	defer f.Close()
	pie.Render(chart.SVG, f)
	fmt.Println("wrote output.svg")
}
