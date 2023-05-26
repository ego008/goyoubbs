package model

type Map map[string]interface{}

func (m Map) Update(m2 Map) {
	// del key not in m2
	for k := range m {
		if _, ok := m2[k]; ok {
			continue
		}
		delete(m, k)
	}
	// add/update key
	for k, v := range m2 {
		m[k] = v
	}
}
