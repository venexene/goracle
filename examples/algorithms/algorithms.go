package algorithms

import (
	"container/heap"
	"errors"
)

type Edge struct {
	To     int
	Weight int
}

type queueItem struct{ vertex, distance int }
type priorityQueue []queueItem

func (q priorityQueue) Len() int           { return len(q) }
func (q priorityQueue) Less(i, j int) bool { return q[i].distance < q[j].distance }
func (q priorityQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *priorityQueue) Push(x any)        { *q = append(*q, x.(queueItem)) }
func (q *priorityQueue) Pop() any {
	old := *q
	x := old[len(old)-1]
	*q = old[:len(old)-1]
	return x
}

func Dijkstra(graph [][]Edge, start int) ([]int, error) {
	if start < 0 || start >= len(graph) {
		return nil, errors.New("начальная вершина вне графа")
	}
	const infinity = int(^uint(0) >> 1)
	distances := make([]int, len(graph))
	for i := range distances {
		distances[i] = infinity
	}
	distances[start] = 0
	queue := &priorityQueue{{vertex: start}}
	heap.Init(queue)
	for queue.Len() > 0 {
		item := heap.Pop(queue).(queueItem)
		if item.distance != distances[item.vertex] {
			continue
		}
		for _, edge := range graph[item.vertex] {
			if edge.Weight < 0 || edge.To < 0 || edge.To >= len(graph) {
				return nil, errors.New("некорректное ребро")
			}
			candidate := item.distance + edge.Weight
			if candidate < distances[edge.To] {
				distances[edge.To] = candidate
				heap.Push(queue, queueItem{edge.To, candidate})
			}
		}
	}
	return distances, nil
}

func TopologicalSort(graph [][]int) ([]int, error) {
	indegree := make([]int, len(graph))
	for _, edges := range graph {
		for _, to := range edges {
			if to < 0 || to >= len(graph) {
				return nil, errors.New("ребро вне графа")
			}
			indegree[to]++
		}
	}
	queue := make([]int, 0, len(graph))
	for vertex, degree := range indegree {
		if degree == 0 {
			queue = append(queue, vertex)
		}
	}
	order := make([]int, 0, len(graph))
	for len(queue) > 0 {
		vertex := queue[0]
		queue = queue[1:]
		order = append(order, vertex)
		for _, to := range graph[vertex] {
			indegree[to]--
			if indegree[to] == 0 {
				queue = append(queue, to)
			}
		}
	}
	if len(order) != len(graph) {
		return nil, errors.New("граф содержит цикл")
	}
	return order, nil
}

type DSU struct{ parent, size []int }

func NewDSU(n int) *DSU {
	d := &DSU{parent: make([]int, n), size: make([]int, n)}
	for i := range n {
		d.parent[i], d.size[i] = i, 1
	}
	return d
}

func (d *DSU) Find(v int) int {
	if d.parent[v] != v {
		d.parent[v] = d.Find(d.parent[v])
	}
	return d.parent[v]
}

func (d *DSU) Union(a, b int) bool {
	a, b = d.Find(a), d.Find(b)
	if a == b {
		return false
	}
	if d.size[a] < d.size[b] {
		a, b = b, a
	}
	d.parent[b] = a
	d.size[a] += d.size[b]
	return true
}
