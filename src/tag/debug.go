package tag

import (
	"sort"
)

// How well does a tagspec match a certain tag? MatchFailures counts which
// elemnts of a tagsepc (and how often) they do not match the tag.
type MatchFailures struct {
	Node                *Node
	Content             int
	ReqAttr, ForbAttr   int
	ReqClass, ForbClass int
	Sub, Deep           int
	Fail                []string
}

func (q MatchFailures) selfOkay() bool {
	return q.Content == 0 && q.ReqAttr == 0 && q.ForbAttr == 0 && q.ReqClass == 0 && q.ForbClass == 0
}
func (q MatchFailures) Total() int {
	return q.Content + q.ReqAttr + q.ForbAttr + q.ReqClass + q.ForbClass + q.Sub + q.Deep
}

// Allow sorting of MatchFailures slices
type QualityArray []MatchFailures

func (a QualityArray) Len() int           { return len(a) }
func (a QualityArray) Less(i, j int) bool { return a[i].Total() < a[j].Total() }
func (a QualityArray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// Calculate how much (on how many elements of ts) node does not match ts.
func Missmatch(ts *TagSpec, node *Node) (mq MatchFailures) {
	mq.Node = node

	// Tag Attributes
	for name, cntnt := range ts.Attr {
		if !containsAttr(node.Attr, name, cntnt) {
			mq.ReqAttr++
			mq.Fail = append(mq.Fail, "Required Attribute: "+name)
		}
	}
	for name, cntnt := range ts.XAttr {
		if containsAttr(node.Attr, name, cntnt) {
			mq.ForbAttr++
			mq.Fail = append(mq.Fail, "Forbidden Attribute: "+name)
		}
	}

	// Classes
	for _, c := range ts.Classes {
		if !containsClass(c, node.class) {
			mq.ReqClass++
			mq.Fail = append(mq.Fail, "Required Class: "+c)
		}
	}
	for _, c := range ts.XClasses {
		if containsClass(c, node.class) {
			mq.ForbClass++
			mq.Fail = append(mq.Fail, "Forbidden class: "+c)
		}
	}

	// Content
	if ts.Content != nil {
		if ts.Deep {
			if !ts.Content.Matches(node.Full) {
				mq.Content = 1
			}
			mq.Fail = append(mq.Fail, "Deep Content")
		} else {
			if !ts.Content.Matches(node.Text) {
				mq.Content = 2
			}
			mq.Fail = append(mq.Fail, "Direct Content")
		}
	}

	// Sub Tags
	ci := 0 // next child to test
	numChilds := len(node.Child)
	for si := 0; si < len(ts.Sub); si++ {
		var found bool = false
		last := ci
		for ; ci < numChilds; ci++ {
			if found = Matches(ts.Sub[si], node.Child[ci]); found {
				break
			}
		}
		if !found {
			// recheck nodes: no match by themself or just subnode mismatch?
			subless := ts.Sub[si].DeepCopy()
			subless.Sub = nil
			for j := last; j < numChilds; j++ {
				if Matches(subless, node.Child[j]) {
					mq.Deep++
				} else {
					mq.Sub++
				}
			}
		}
	}
	return
}

// Return a list of all (just same tag) nodes, sorted by amount of mismatch to ts.
func RankNodes(ts *TagSpec, node *Node) []MatchFailures {
	list := make([]MatchFailures, 0, 20)
	list = rankNodes(ts, node, list)
	sort.Sort(QualityArray(list))
	return list
}

// Real work part of RankNodes.
func rankNodes(ts *TagSpec, node *Node, best []MatchFailures) []MatchFailures {
	if best == nil {
	}
	if node.Name == ts.Name {
		q := Missmatch(ts, node)
		best = append(best, q)
	}
	for _, child := range node.Child {
		best = rankNodes(ts, child, best)
	}
	return best
}
