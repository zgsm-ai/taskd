package task

import "fmt"

// Metric Define metric type
type Metric map[string]any

// Add Add metric
func (m *Metric) Add(name string, value any) {
	(*m)[name] = value
}

// Remove Remove metric
func (m *Metric) Remove(name string) {
	delete(*m, name)
}

// Get Get metric value
func (m *Metric) Get(name string) any {
	return (*m)[name]
}

// ToString View all metric values
func (m *Metric) ToString() string {
	result := ""
	for k, v := range *m {
		result = fmt.Sprintf("%v\n%v=%v", result, k, v)
	}
	return result
}
