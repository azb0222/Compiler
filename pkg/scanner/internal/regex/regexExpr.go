package regex

import (
	"asritha.dev/compiler/pkg/scanner/internal/fsm"
	_	"asritha.dev/compiler/pkg/scanner/internal/fsm"
	"fmt"
)

// separate out the types, printer, nfa converter into seperate files
const (
	epsilon rune = 0
)

// RExpr is implemented by all types automatically
type RExpr interface {
}

type String interface {
	String() string
}

// TODO add error handling?
type ASTPrinter interface {
	PrintNode(indent string) string
}

// TODO: write algorithm name in comment here, comment explaining what it does?
type NFAConverter interface {
	convertToNFA(idCounter *uint) (*fsm.NFAState, *fsm.NFAState, error) //start, end, create aliases?
}

type NFAPrinter interface {
	//printmermaidNFA()
}

type Const struct {
	value rune
}

func NewConst(value rune) *Const {
	return &Const{value: value}
}

func (c *Const) String() string {
	return fmt.Sprintf("%c", c.value)
}

func (c *Const) PrintNode(indent string) string {
	return fmt.Sprintf("%sConst { %c }", indent, c.value)
}

func (c *Const) convertToNFA(idCounter *uint) (*fsm.NFAState, *fsm.NFAState, error) {
	startState := fsm.NewNFAState(idCounter, false)
	endState := fsm.NewNFAState(idCounter, true)
	startState.AddTransition(c.value, endState)

	return startState, endState, nil
}

type Alternation struct { // left | right
	left  RExpr
	right RExpr
}

func NewAlternation(left RExpr, right RExpr) *Alternation {
	return &Alternation{left: left, right: right}
}

func (a *Alternation) String() string {
	return fmt.Sprintf("%s|%s", a.left, a.right)
}

func (a *Alternation) PrintNode(indent string) string {
	left, ok := a.left.(ASTPrinter)
	if !ok {
		return fmt.Sprintf("%sERROR PRINTING LEFT ALTERNATION", indent)
	}

	right, ok := a.right.(ASTPrinter)
	if(!ok){
		return fmt.Sprintf("%sERROR PRINTING RIGHT ALTERNATION", indent)
	}
	
	return fmt.Sprintf(
		"%sAlternation {\n%v,\n%v\n%s}",
		indent, left.PrintNode(indent+"  "),
		right.PrintNode(indent+"  "),
		indent,
	) 
}

// TODO add proper errors
func (a *Alternation) convertToNFA(idCounter *uint) (*fsm.NFAState, *fsm.NFAState, error) {
	left, ok := a.left.(NFAConverter)
	if(!ok){
		return nil, nil, fmt.Errorf("left fail")
	}
	right, ok := a.right.(NFAConverter)
	if(!ok){
		return nil, nil, fmt.Errorf("right fail")
	}
	
	leftNFAStartState, leftNFAEndState := left.convertToNFA(idCounter)
	rightNFAStartState, rightNFAEndState := right.convertToNFA(idCounter)
			startState := &fsm.NFAState{
				FAState: fsm.FAState{
					Id:          idCounter + 1,
					IsAccepting: false,
				},
				Transitions: map[rune][]*fsm.NFAState{
					epsilon: []*fsm.NFAState{
						leftNFAStartState,
						rightNFAStartState,
					},
				},
			}
			endState := &fsm.NFAState{
				FAState: fsm.FAState{
					Id:          idCounter + 1,
					IsAccepting: true,
				},
				Transitions: make(map[rune][]*fsm.NFAState),
			}
			rightNFAEndState.IsAccepting = false
			leftNFAEndState.IsAccepting = false

			rightNFAEndState.Transitions[epsilon] = append(rightNFAEndState.Transitions[epsilon], endState)
			leftNFAEndState.Transitions[epsilon] = append(leftNFAEndState.Transitions[epsilon], endState)

			return startState, endState
		}
	}
	return nil, nil
}

type Concatenation struct { // left right
	Left  RExpr
	Right RExpr
}

func NewConcatenation(left RExpr, right RExpr) *Concatenation {
	return &Concatenation{Left: left, Right: right}
}

func (c *Concatenation) String() string {
	return fmt.Sprintf("%s%s", c.Left, c.Right)
}

func (c *Concatenation) PrintNode(indent string) string {
	if left, ok := c.Left.(ASTPrinter); ok {
		if right, ok := c.Right.(ASTPrinter); ok {
			return fmt.Sprintf(
				"%sConcatenation {\n%v,\n%v\n%s}",
				indent, left.PrintNode(indent+"  "),
				right.PrintNode(indent+"  "),
				indent,
			)
		}
	}
	return fmt.Sprintf("%sERROR PRINTING CONCATENATION", indent)
}

func (c *Concatenation) convertToNFA(idCounter uint) (*scanner.NFAState, *scanner.NFAState) {
	if left, ok := c.Left.(NFAConverter); ok {
		if right, ok := c.Right.(NFAConverter); ok {
			//TODO: change the leftNFAEndState to not be final here, and in each corresponding function?
			leftNFAStartState, leftNFAEndState := left.convertToNFA(idCounter) //TODO: the idCounter is not being handled properly, return it everywhere and then increment by 1?
			rightNFAStartState, rightNFAEndState := right.convertToNFA(idCounter)
			leftNFAStartState.IsAccepting = false
			rightNFAEndState.IsAccepting = true
			leftNFAEndState.Transitions[epsilon] = append(leftNFAStartState.Transitions[epsilon], rightNFAStartState)

			return leftNFAStartState, rightNFAEndState
		} else {
			fmt.Println("Right part is invalid")
			//TODO: put in better error handling everywhere instead of just printing it
		}
	} else {
		fmt.Println("Left part is invalid")
	}
	return nil, nil
}

type KleeneStar struct { // left*
	Left RExpr
}

func NewKleeneStar(left RExpr) *KleeneStar {
	return &KleeneStar{Left: left}
}

func (ks *KleeneStar) String() string {
	return fmt.Sprintf("(%s)*", ks.Left)
}

func (ks *KleeneStar) PrintNode(indent string) string {
	if left, ok := ks.Left.(ASTPrinter); ok {
		return fmt.Sprintf(
			"%sKleeneStar {\n%v\n%s}",
			indent, left.PrintNode(indent+"  "),
			indent,
		)
	}
	return fmt.Sprintf("%sERROR PRINTING KLEENE_STAR", indent)
}

func (ks *KleeneStar) convertToNFA(idCounter uint) (*scanner.NFAState, *scanner.NFAState) {
	if left, ok := ks.Left.(NFAConverter); ok {
		leftNFAStartState, leftNFAEndState := left.convertToNFA(idCounter)

		leftNFAEndState.IsAccepting = false

		startState := &scanner.NFAState{
			FAState: scanner.FAState{
				Id:          idCounter + 1,
				IsAccepting: false,
			},
			Transitions: map[rune][]*scanner.NFAState{
				epsilon: []*scanner.NFAState{
					leftNFAStartState,
				},
			},
		}
		endState := &scanner.NFAState{
			FAState: scanner.FAState{
				Id:          idCounter + 1,
				IsAccepting: true,
			},
			Transitions: make(map[rune][]*scanner.NFAState),
		}

		//TODO: the below two prolly dont have to be separate, can blend together
		startState.Transitions[epsilon] = append(startState.Transitions[epsilon], endState)
		endState.Transitions[epsilon] = append(endState.Transitions[epsilon], startState)

		leftNFAEndState.Transitions[epsilon] = append(leftNFAEndState.Transitions[epsilon], endState)

		return startState, endState
	}
	return nil, nil
}

//TODO: make a mermaid js thing for this
