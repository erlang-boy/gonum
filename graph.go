package discrete

import (
	"math"
	"sort"
)

// A Graph implements all methods necessary to run graph-specific algorithms on it. Strictly speaking EdgeList() and NodeList() would be sufficient, but the others (such as Successors)
// are required to make the algorithms more efficient and/or easier to write and maintain.
type Graph interface {
	Successors(node int) []int                // Gives the nodes connected by OUTBOUND edges, if the graph is an undirected graph, this set is equal to Predecessors
	IsSuccessor(node, successor int) bool     // If successor shows up in the list returned by Successors(node), then it's a successor. If node doesn't exist, this should always return false
	Predecessors(node int) []int              // Gives the nodes connected by INBOUND edges, if the graph is an undirected graph, this set is equal to Successors
	IsPredecessor(node, predecessor int) bool // If predecessor shows up in the list returned by Predecessors(node), then it's a predecessor. If node doesn't exist, this should always return false
	IsAdjacent(node, neighbor int) bool       // IsSuccessor || IsPredecessor
	NodeExists(node int) bool                 // Returns whether a node with the given ID is currently in the graph
	Degree(node int) int                      // Degree is equivalent to len(Successors(node)) + len(Predecessors(node)); this means that reflexive edges are counted twice
	EdgeList() [][2]int                       // Returns a list of all edges in the graph. In the case of an directed graph edge[0] goes TO edge[1]. In an undirected graph, provide both directions as separate edges
	NodeList() []int                          // Returns a list of all node IDs in no particular order, useful for determining things like if a graph is fully connected. The caller is free to modify this list (so don't pass a reference to your own list)
	IsDirected() bool
}

// A Graph that implements Coster has an actual cost between adjacent nodes, also known as a weighted graph. If a graph implements coster and a function needs to read cost (e.g. A*), this function will
// take precedence over the Uniform Cost function (all weights are 1) if "nil" is passed in for the function argument
//
// Coster only need worry about the case when an edge from node 1 to node 2 exists (i.e. node2 is a successor to node1) -- asking for the weight in any other case is considered undefined behavior
type Coster interface {
	Cost(node1, node2 int) float64
}

// A graph that implements HeuristicCoster implements a heuristic between any two given nodes. Like Coster, if a graph implements this and a function needs a heuristic cost (e.g. A*), this function will
// take precedence over the Null Heuristic (always returns 0) if "nil" is passed in for the function argument
type HeuristicCoster interface {
	Coster
	HeuristicCost(node1, node2 int) float64 // If HeuristicCost is not intended to be used, it can be implemented as the null heuristic (always returns 0)
}

// A Mutable Graph is a graph that can be changed in an arbitrary way. It is useful for several algorithms; for instance, Johnson's Algorithm requires adding a temporary node and changing edge weights.
// Another case where this is used is computing minimum spanning trees. Since trees are graphs, a minimum spanning tree can be created using this interface.
//
// Note that just because a graph does not implement MutableGraph does not mean that this package expects it to be invariant (though even a MutableGraph should be treated as invariant while an algorithm
// is operating on it), it simply means that without this interface this package can not properly handle the graph in order to, say, fill it with a minimum spanning tree.
//
// In functions that take a MutableGraph as an argument, it should not be the same as the Graph argument as concurrent modification will likely cause problems in most cases.
type MutableGraph interface {
	Graph
	NewNode(successors []int) int               // Adds a node with an arbitrary ID, and returns the new, unique ID used
	AddNode(id int, successors []int)           // The graph itself is responsible for adding reciprocal edges if it's undirected. Likewise, the graph itself must add any non-existant nodes listed in successors.
	AddEdge(node1, node2 int)                   // For a digraph, adds node1->node2; the graph is free to initialize this to any value it wishes. Node1 must exist, or it will result in undefined behavior, node2 must be created by the function if absent
	SetEdgeCost(node1, node2 int, cost float64) // The behavior is undefined if the edge has not been created with AddEdge (or the edge was removed before this function was called). For a directed graph only sets node1->node2
	RemoveNode(node int)                        // The graph is reponsible for removing edges to a node that is removed
	RemoveEdge(node1, node2 int)                // The graph is responsible for removing reciprocal edges if it's undirected
	EmptyGraph()                                // Clears the graph of all nodes and edges
	SetDirected(bool)                           // This package will only call SetDirected on an empty graph, so there's no need to worry about the case where a graph suddenly becomes (un)directed
}

// A package that contains an edge (as from EdgeList), and a Weight (as if Cost(Edge[0], Edge[1]) had been called)
type WeightedEdge struct {
	Edge   [2]int
	Weight float64
}

/* Basic Graph tests */

// Also known as Tarjan's Strongly Connected Components Algorithm. This returns all the strongly connected components in the graph.
//
// A strongly connected component of a graph is a set of vertices where it's possible to reach any vertex in the set from any other (meaning there's a cycle between them)
//
// Generally speaking, a directed graph where the number of strongly connected components is equal to the number of nodes is acyclic, unless you count reflexive edges as a cycle (which requires only a little extra testing)
//
// An undirected graph should end up with as many SCCs as there are "islands" (or subgraphs) of connections, meaning having more than one strongly connected component implies that your graph is not fully connected.
func Tarjan(graph Graph) (sccs [][]int) {
	index := 0
	vStack := &Stack{}
	stackSet := NewSet()
	sccs = make([][]int, 0)

	nodes := graph.NodeList()
	lowlinks := make(map[int]int, len(nodes))
	indices := make(map[int]int, len(nodes))

	var strongconnect func(int) []int

	strongconnect = func(node int) []int {
		indices[node] = index
		lowlinks[node] = index
		index += 1

		vStack.Push(node)
		stackSet.Add(node)

		for _, succ := range graph.Successors(node) {
			if _, ok := indices[succ]; !ok {
				strongconnect(succ)
				lowlinks[node] = int(math.Min(float64(lowlinks[node]), float64(lowlinks[succ])))
			} else if stackSet.Contains(succ) {
				lowlinks[node] = int(math.Min(float64(lowlinks[node]), float64(lowlinks[succ])))
			}
		}

		if lowlinks[node] == indices[node] {
			scc := make([]int, 0)
			for {
				v, _ := vStack.Pop()
				stackSet.Remove(v.(int))
				scc = append(scc, v.(int))
				if v.(int) == node {
					return scc
				}
			}
		}

		return nil
	}

	for _, n := range nodes {
		if _, ok := indices[n]; !ok {
			sccs = append(sccs, strongconnect(n))
		}
	}

	return sccs
}

// Returns true if, starting at path[0] and ending at path[len(path)-1], all nodes between are valid neighbors. That is, for each element path[i], path[i+1] is a valid successor
//
// Special case: a nil or zero length path is considered valid (true), a path of length 1 (only one node) is the trivial case, but only if the node listed in path exists.
func IsPath(path []int, graph Graph) bool {
	if path == nil || len(path) == 0 {
		return true
	} else if len(path) == 1 {
		return graph.NodeExists(path[0])
	}

	for i := 0; i < len(path)-1; i++ {
		if !graph.IsSuccessor(path[i], path[i+1]) {
			return false
		}
	}

	return true
}

/* Implements minimum-spanning tree algorithms; puts the resulting minimum spanning tree in the dst graph */

// Generates a minimum spanning tree with sets.
//
// As with other algorithms that use Cost, the order of precedence is Argument > Interface > UniformCost
func Prim(dst MutableGraph, graph Graph, Cost func(int, int) float64) {
	if Cost == nil {
		if cgraph, ok := graph.(Coster); ok {
			Cost = cgraph.Cost
		} else {
			Cost = UniformCost
		}
	}
	dst.EmptyGraph()
	dst.SetDirected(false)

	nlist := graph.NodeList()

	if nlist == nil || len(nlist) == 0 {
		return
	}

	dst.AddNode(nlist[0], nil)
	remainingNodes := NewSet()
	for _, node := range nlist[1:] {
		remainingNodes.Add(node)
	}

	edgeList := graph.EdgeList()
	for remainingNodes.Cardinality() != 0 {
		edgeWeights := make(edgeSorter, 0)
		for _, edge := range edgeList {
			if dst.NodeExists(edge[0]) && remainingNodes.Contains(edge[1]) {
				edgeWeights = append(edgeWeights, WeightedEdge{edge, Cost(edge[0], edge[1])})
			}
		}

		sort.Sort(edgeWeights)
		myEdge := edgeWeights[0]

		if !dst.NodeExists(myEdge.Edge[0]) {
			dst.AddNode(myEdge.Edge[0], []int{myEdge.Edge[1]})
		} else {
			dst.AddEdge(myEdge.Edge[0], myEdge.Edge[1])
		}
		dst.SetEdgeCost(myEdge.Edge[0], myEdge.Edge[1], myEdge.Weight)

		remainingNodes.Remove(myEdge.Edge[1])
	}

}

// Generates a minimum spanning tree for a graph using discrete.DisjointSet
//
// As with other algorithms with Cost, the precedence goes Argument > Interface > UniformCost
func Kruskal(dst MutableGraph, graph Graph, Cost func(int, int) float64) {
	if Cost == nil {
		if cgraph, ok := graph.(Coster); ok {
			Cost = cgraph.Cost
		} else {
			Cost = UniformCost
		}
	}
	dst.EmptyGraph()
	dst.SetDirected(false)

	edgeList := graph.EdgeList()
	edgeWeights := make(edgeSorter, 0, len(edgeList))
	for _, edge := range edgeList {
		edgeWeights = append(edgeWeights, WeightedEdge{edge, Cost(edge[0], edge[1])})
	}

	sort.Sort(edgeWeights)

	ds := NewDisjointSet()
	for _, node := range graph.NodeList() {
		ds.MakeSet(node)
	}

	for _, edge := range edgeWeights {
		if s1, s2 := ds.Find(edge.Edge[0]), ds.Find(edge.Edge[1]); s1 != s2 {
			ds.Union(s1, s2)
			if !dst.NodeExists(edge.Edge[0]) {
				dst.AddNode(edge.Edge[0], []int{edge.Edge[1]})
			} else {
				dst.AddEdge(edge.Edge[0], edge.Edge[1])
			}
			dst.SetEdgeCost(edge.Edge[0], edge.Edge[1], edge.Weight)
		}
	}
}

/* Control flow graph stuff */

// A dominates B if and only if the only path through B travels through A
//
// This returns all possible dominators for all nodes, it does not prune for strict dominators, immediate dominators etc
func Dominators(start int, graph Graph) map[int]*Set {
	allNodes := NewSet()
	nlist := graph.NodeList()
	dominators := make(map[int]*Set, len(nlist))
	for _, node := range nlist {
		allNodes.Add(node)
	}

	for _, node := range nlist {
		dominators[node] = NewSet()
		if node == start {
			dominators[node].Add(start)
		} else {
			dominators[node].Copy(allNodes)
		}
	}

	for somethingChanged := true; somethingChanged; {
		somethingChanged = false
		for _, node := range nlist {
			if node == start {
				continue
			}
			preds := graph.Predecessors(node)
			if len(preds) == 0 {
				continue
			}
			tmp := NewSet().Copy(dominators[preds[0]])
			for _, pred := range preds[1:] {
				tmp.Intersection(tmp, dominators[pred])
			}

			dom := NewSet()
			dom.Add(node)

			dom.Union(dom, tmp)
			if !Equal(dom, dominators[node]) {
				dominators[node] = dom
				somethingChanged = true
			}
		}
	}

	return dominators
}

// A Postdominates B if and only if all paths from B travel through A
//
// This returns all possible post-dominators for all nodes, it does not prune for strict postdominators, immediate postdominators etc
func PostDominators(end int, graph Graph) map[int]*Set {
	allNodes := NewSet()
	nlist := graph.NodeList()
	dominators := make(map[int]*Set, len(nlist))
	for _, node := range nlist {
		allNodes.Add(node)
	}

	for _, node := range nlist {
		dominators[node] = NewSet()
		if node == end {
			dominators[node].Add(end)
		} else {
			dominators[node].Copy(allNodes)
		}
	}

	for somethingChanged := true; somethingChanged; {
		somethingChanged = false
		for _, node := range nlist {
			if node == end {
				continue
			}
			succs := graph.Successors(node)
			if len(succs) == 0 {
				continue
			}
			tmp := NewSet().Copy(dominators[succs[0]])
			for _, succ := range succs[1:] {
				tmp.Intersection(tmp, dominators[succ])
			}

			dom := NewSet()
			dom.Add(node)

			dom.Union(dom, tmp)
			if !Equal(dom, dominators[node]) {
				dominators[node] = dom
				somethingChanged = true
			}
		}
	}

	return dominators
}

/* Purely internal data structures and functions (mostly for sorting) */

/** Sorts a list of edges by weight, agnostic to repeated edges as well as direction **/

type edgeSorter []WeightedEdge

func (el edgeSorter) Len() int {
	return len(el)
}

func (el edgeSorter) Less(i, j int) bool {
	return el[i].Weight < el[j].Weight
}

func (el edgeSorter) Swap(i, j int) {
	el[i], el[j] = el[j], el[i]
}
