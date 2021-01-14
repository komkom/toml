package toml

type Map struct {
	Var Var
	m   map[string]Map
}

func (m Map) Set(key []string, v Var) bool {

	currentMap := m
	for idx, sk := range key {

		subMap, ok := currentMap.m[sk]

		if subMap.Var == BasicVar {
			return false
		}

		if idx == len(key)-1 {

			if ok {
				if subMap.Var == ImplicitTableVar && v == TableVar {
					subMap.Var = TableVar

				} else if subMap.Var != ArrayVar {
					return false
				}
			} else {
				subMap.Var = v
			}

			currentMap.m[sk] = subMap
			return true
		}

		if !ok {
			subMap.Var = ImplicitTableVar
		}

		if subMap.m == nil {
			subMap.m = make(map[string]Map)
		}

		currentMap.m[sk] = subMap
		currentMap = subMap
	}

	return true
}

func (m Map) Get(key []string) (Map, bool) {

	currentMap := m
	for _, sk := range key {

		if currentMap.m == nil {
			return Map{}, false
		}

		subMap, ok := currentMap.m[sk]
		if !ok {
			return Map{}, false
		}

		currentMap = subMap
	}
	return currentMap, true
}

func (m Map) Clear(key []string) bool {

	currentMap := m
	for idx, sk := range key {

		if currentMap.m == nil {
			return false
		}

		subMap, ok := currentMap.m[sk]
		if !ok {
			return false
		}

		if idx == len(key)-1 {
			for k := range subMap.m {
				delete(subMap.m, k)
			}
			return true
		}

		currentMap = subMap
	}
	return false
}
