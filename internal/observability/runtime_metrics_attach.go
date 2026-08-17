package observability

func (m *Metrics) NewRuntimeMetrics() *RuntimeMetrics {
	if m == nil {
		return nil
	}
	return NewRuntimeMetrics(m.registry)
}
