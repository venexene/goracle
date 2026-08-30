package algorithms

import "testing"

func TestDijkstra(t *testing.T) {
	graph := [][]Edge{{{1, 4}, {2, 1}}, {{3, 1}}, {{1, 2}, {3, 5}}, {}}
	got, err := Dijkstra(graph, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{0, 3, 1, 4}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("distance[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestTopologicalSortRejectsCycle(t *testing.T) {
	if _, err := TopologicalSort([][]int{{1}, {0}}); err == nil {
		t.Fatal("cycle must be rejected")
	}
}

func TestDSU(t *testing.T) {
	d := NewDSU(4)
	if !d.Union(0, 1) || !d.Union(1, 2) || d.Union(0, 2) {
		t.Fatal("unexpected union result")
	}
	if d.Find(0) != d.Find(2) || d.Find(0) == d.Find(3) {
		t.Fatal("unexpected components")
	}
}
