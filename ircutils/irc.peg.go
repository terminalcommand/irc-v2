package ircutils

//go:generate c:\Users\Erkin\go\bin\peg.exe irc.peg

import (
	"fmt"
	"github.com/pointlander/peg/tree"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleMessage
	rulePrefix
	ruleServer
	rulePerson
	ruleNick
	ruleUser
	ruleHost
	ruleCommand
	ruleParams
	ruleMiddle
	ruleTrailing
	ruleNonTerminating
	ruleSquareBrackets
	ruleCurlyBrackets
	ruleBacktick
	ruleAlphaNum
	ruleLetter
	ruleNum
	ruleCRLF
	ruleS
	ruleAction0
)

var rul3s = [...]string{
	"Unknown",
	"Message",
	"Prefix",
	"Server",
	"Person",
	"Nick",
	"User",
	"Host",
	"Command",
	"Params",
	"Middle",
	"Trailing",
	"NonTerminating",
	"SquareBrackets",
	"CurlyBrackets",
	"Backtick",
	"AlphaNum",
	"Letter",
	"Num",
	"CRLF",
	"S",
	"Action0",
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Printf(" ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Printf("%v %v\n", rule, quote)
			} else {
				fmt.Printf("\x1B[34m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(buffer string) {
	node.print(false, buffer)
}

func (node *node32) PrettyPrint(buffer string) {
	node.print(true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	if tree := t.tree; int(index) >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	t.tree[index] = token32{
		pegRule: rule,
		begin:   begin,
		end:     end,
	}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type IRCParser struct {
	*tree.Tree

	Buffer string
	buffer []rune
	rules  [22]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *IRCParser) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *IRCParser) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *IRCParser
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *IRCParser) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *IRCParser) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for _, token := range p.Tokens() {
		switch token.pegRule {

		case ruleAction0:
			fmt.Println(buffer[begin:end])

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *IRCParser) Init() {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := tokens32{tree: make([]token32, math.MaxInt16)}
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 Message <- <(Prefix? Command Params? CRLF Action0)> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
				{
					position2, tokenIndex2 := position, tokenIndex
					if !_rules[rulePrefix]() {
						goto l2
					}
					goto l3
				l2:
					position, tokenIndex = position2, tokenIndex2
				}
			l3:
				if !_rules[ruleCommand]() {
					goto l0
				}
				{
					position4, tokenIndex4 := position, tokenIndex
					if !_rules[ruleParams]() {
						goto l4
					}
					goto l5
				l4:
					position, tokenIndex = position4, tokenIndex4
				}
			l5:
				if !_rules[ruleCRLF]() {
					goto l0
				}
				if !_rules[ruleAction0]() {
					goto l0
				}
				add(ruleMessage, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 Prefix <- <(':' (Server / Person))> */
		func() bool {
			position6, tokenIndex6 := position, tokenIndex
			{
				position7 := position
				if buffer[position] != rune(':') {
					goto l6
				}
				position++
				{
					position8, tokenIndex8 := position, tokenIndex
					if !_rules[ruleServer]() {
						goto l9
					}
					goto l8
				l9:
					position, tokenIndex = position8, tokenIndex8
					if !_rules[rulePerson]() {
						goto l6
					}
				}
			l8:
				add(rulePrefix, position7)
			}
			return true
		l6:
			position, tokenIndex = position6, tokenIndex6
			return false
		},
		/* 2 Server <- <((AlphaNum / SquareBrackets / (' ' / '_' / [ - ] / '.' / ' ' / '|' / ' ' / '*' / ' '))+ !(User / Host) S)> */
		func() bool {
			position10, tokenIndex10 := position, tokenIndex
			{
				position11 := position
				{
					position14, tokenIndex14 := position, tokenIndex
					if !_rules[ruleAlphaNum]() {
						goto l15
					}
					goto l14
				l15:
					position, tokenIndex = position14, tokenIndex14
					if !_rules[ruleSquareBrackets]() {
						goto l16
					}
					goto l14
				l16:
					position, tokenIndex = position14, tokenIndex14
					{
						position17, tokenIndex17 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l18
						}
						position++
						goto l17
					l18:
						position, tokenIndex = position17, tokenIndex17
						if buffer[position] != rune('_') {
							goto l19
						}
						position++
						goto l17
					l19:
						position, tokenIndex = position17, tokenIndex17
						if c := buffer[position]; c < rune(' ') || c > rune(' ') {
							goto l20
						}
						position++
						goto l17
					l20:
						position, tokenIndex = position17, tokenIndex17
						if buffer[position] != rune('.') {
							goto l21
						}
						position++
						goto l17
					l21:
						position, tokenIndex = position17, tokenIndex17
						if buffer[position] != rune(' ') {
							goto l22
						}
						position++
						goto l17
					l22:
						position, tokenIndex = position17, tokenIndex17
						if buffer[position] != rune('|') {
							goto l23
						}
						position++
						goto l17
					l23:
						position, tokenIndex = position17, tokenIndex17
						if buffer[position] != rune(' ') {
							goto l24
						}
						position++
						goto l17
					l24:
						position, tokenIndex = position17, tokenIndex17
						if buffer[position] != rune('*') {
							goto l25
						}
						position++
						goto l17
					l25:
						position, tokenIndex = position17, tokenIndex17
						if buffer[position] != rune(' ') {
							goto l10
						}
						position++
					}
				l17:
				}
			l14:
			l12:
				{
					position13, tokenIndex13 := position, tokenIndex
					{
						position26, tokenIndex26 := position, tokenIndex
						if !_rules[ruleAlphaNum]() {
							goto l27
						}
						goto l26
					l27:
						position, tokenIndex = position26, tokenIndex26
						if !_rules[ruleSquareBrackets]() {
							goto l28
						}
						goto l26
					l28:
						position, tokenIndex = position26, tokenIndex26
						{
							position29, tokenIndex29 := position, tokenIndex
							if buffer[position] != rune(' ') {
								goto l30
							}
							position++
							goto l29
						l30:
							position, tokenIndex = position29, tokenIndex29
							if buffer[position] != rune('_') {
								goto l31
							}
							position++
							goto l29
						l31:
							position, tokenIndex = position29, tokenIndex29
							if c := buffer[position]; c < rune(' ') || c > rune(' ') {
								goto l32
							}
							position++
							goto l29
						l32:
							position, tokenIndex = position29, tokenIndex29
							if buffer[position] != rune('.') {
								goto l33
							}
							position++
							goto l29
						l33:
							position, tokenIndex = position29, tokenIndex29
							if buffer[position] != rune(' ') {
								goto l34
							}
							position++
							goto l29
						l34:
							position, tokenIndex = position29, tokenIndex29
							if buffer[position] != rune('|') {
								goto l35
							}
							position++
							goto l29
						l35:
							position, tokenIndex = position29, tokenIndex29
							if buffer[position] != rune(' ') {
								goto l36
							}
							position++
							goto l29
						l36:
							position, tokenIndex = position29, tokenIndex29
							if buffer[position] != rune('*') {
								goto l37
							}
							position++
							goto l29
						l37:
							position, tokenIndex = position29, tokenIndex29
							if buffer[position] != rune(' ') {
								goto l13
							}
							position++
						}
					l29:
					}
				l26:
					goto l12
				l13:
					position, tokenIndex = position13, tokenIndex13
				}
				{
					position38, tokenIndex38 := position, tokenIndex
					{
						position39, tokenIndex39 := position, tokenIndex
						if !_rules[ruleUser]() {
							goto l40
						}
						goto l39
					l40:
						position, tokenIndex = position39, tokenIndex39
						if !_rules[ruleHost]() {
							goto l38
						}
					}
				l39:
					goto l10
				l38:
					position, tokenIndex = position38, tokenIndex38
				}
				if !_rules[ruleS]() {
					goto l10
				}
				add(ruleServer, position11)
			}
			return true
		l10:
			position, tokenIndex = position10, tokenIndex10
			return false
		},
		/* 3 Person <- <(Nick User? Host? S)> */
		func() bool {
			position41, tokenIndex41 := position, tokenIndex
			{
				position42 := position
				if !_rules[ruleNick]() {
					goto l41
				}
				{
					position43, tokenIndex43 := position, tokenIndex
					if !_rules[ruleUser]() {
						goto l43
					}
					goto l44
				l43:
					position, tokenIndex = position43, tokenIndex43
				}
			l44:
				{
					position45, tokenIndex45 := position, tokenIndex
					if !_rules[ruleHost]() {
						goto l45
					}
					goto l46
				l45:
					position, tokenIndex = position45, tokenIndex45
				}
			l46:
				if !_rules[ruleS]() {
					goto l41
				}
				add(rulePerson, position42)
			}
			return true
		l41:
			position, tokenIndex = position41, tokenIndex41
			return false
		},
		/* 4 Nick <- <(AlphaNum / SquareBrackets / CurlyBrackets / Backtick / (' ' / '_' / [ - ] / '^' / ' ' / '|' / ' ' / '\\' / ' '))+> */
		func() bool {
			position47, tokenIndex47 := position, tokenIndex
			{
				position48 := position
				{
					position51, tokenIndex51 := position, tokenIndex
					if !_rules[ruleAlphaNum]() {
						goto l52
					}
					goto l51
				l52:
					position, tokenIndex = position51, tokenIndex51
					if !_rules[ruleSquareBrackets]() {
						goto l53
					}
					goto l51
				l53:
					position, tokenIndex = position51, tokenIndex51
					if !_rules[ruleCurlyBrackets]() {
						goto l54
					}
					goto l51
				l54:
					position, tokenIndex = position51, tokenIndex51
					if !_rules[ruleBacktick]() {
						goto l55
					}
					goto l51
				l55:
					position, tokenIndex = position51, tokenIndex51
					{
						position56, tokenIndex56 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l57
						}
						position++
						goto l56
					l57:
						position, tokenIndex = position56, tokenIndex56
						if buffer[position] != rune('_') {
							goto l58
						}
						position++
						goto l56
					l58:
						position, tokenIndex = position56, tokenIndex56
						if c := buffer[position]; c < rune(' ') || c > rune(' ') {
							goto l59
						}
						position++
						goto l56
					l59:
						position, tokenIndex = position56, tokenIndex56
						if buffer[position] != rune('^') {
							goto l60
						}
						position++
						goto l56
					l60:
						position, tokenIndex = position56, tokenIndex56
						if buffer[position] != rune(' ') {
							goto l61
						}
						position++
						goto l56
					l61:
						position, tokenIndex = position56, tokenIndex56
						if buffer[position] != rune('|') {
							goto l62
						}
						position++
						goto l56
					l62:
						position, tokenIndex = position56, tokenIndex56
						if buffer[position] != rune(' ') {
							goto l63
						}
						position++
						goto l56
					l63:
						position, tokenIndex = position56, tokenIndex56
						if buffer[position] != rune('\\') {
							goto l64
						}
						position++
						goto l56
					l64:
						position, tokenIndex = position56, tokenIndex56
						if buffer[position] != rune(' ') {
							goto l47
						}
						position++
					}
				l56:
				}
			l51:
			l49:
				{
					position50, tokenIndex50 := position, tokenIndex
					{
						position65, tokenIndex65 := position, tokenIndex
						if !_rules[ruleAlphaNum]() {
							goto l66
						}
						goto l65
					l66:
						position, tokenIndex = position65, tokenIndex65
						if !_rules[ruleSquareBrackets]() {
							goto l67
						}
						goto l65
					l67:
						position, tokenIndex = position65, tokenIndex65
						if !_rules[ruleCurlyBrackets]() {
							goto l68
						}
						goto l65
					l68:
						position, tokenIndex = position65, tokenIndex65
						if !_rules[ruleBacktick]() {
							goto l69
						}
						goto l65
					l69:
						position, tokenIndex = position65, tokenIndex65
						{
							position70, tokenIndex70 := position, tokenIndex
							if buffer[position] != rune(' ') {
								goto l71
							}
							position++
							goto l70
						l71:
							position, tokenIndex = position70, tokenIndex70
							if buffer[position] != rune('_') {
								goto l72
							}
							position++
							goto l70
						l72:
							position, tokenIndex = position70, tokenIndex70
							if c := buffer[position]; c < rune(' ') || c > rune(' ') {
								goto l73
							}
							position++
							goto l70
						l73:
							position, tokenIndex = position70, tokenIndex70
							if buffer[position] != rune('^') {
								goto l74
							}
							position++
							goto l70
						l74:
							position, tokenIndex = position70, tokenIndex70
							if buffer[position] != rune(' ') {
								goto l75
							}
							position++
							goto l70
						l75:
							position, tokenIndex = position70, tokenIndex70
							if buffer[position] != rune('|') {
								goto l76
							}
							position++
							goto l70
						l76:
							position, tokenIndex = position70, tokenIndex70
							if buffer[position] != rune(' ') {
								goto l77
							}
							position++
							goto l70
						l77:
							position, tokenIndex = position70, tokenIndex70
							if buffer[position] != rune('\\') {
								goto l78
							}
							position++
							goto l70
						l78:
							position, tokenIndex = position70, tokenIndex70
							if buffer[position] != rune(' ') {
								goto l50
							}
							position++
						}
					l70:
					}
				l65:
					goto l49
				l50:
					position, tokenIndex = position50, tokenIndex50
				}
				add(ruleNick, position48)
			}
			return true
		l47:
			position, tokenIndex = position47, tokenIndex47
			return false
		},
		/* 5 User <- <('!' (AlphaNum / SquareBrackets / (' ' / '_' / [ - ] / '=' / ' ' / '.' / ' ' / '~' / ' ' / '^' / ' ' / '\\' / ' ' / '`' / ' '))+)> */
		func() bool {
			position79, tokenIndex79 := position, tokenIndex
			{
				position80 := position
				if buffer[position] != rune('!') {
					goto l79
				}
				position++
				{
					position83, tokenIndex83 := position, tokenIndex
					if !_rules[ruleAlphaNum]() {
						goto l84
					}
					goto l83
				l84:
					position, tokenIndex = position83, tokenIndex83
					if !_rules[ruleSquareBrackets]() {
						goto l85
					}
					goto l83
				l85:
					position, tokenIndex = position83, tokenIndex83
					{
						position86, tokenIndex86 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l87
						}
						position++
						goto l86
					l87:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune('_') {
							goto l88
						}
						position++
						goto l86
					l88:
						position, tokenIndex = position86, tokenIndex86
						if c := buffer[position]; c < rune(' ') || c > rune(' ') {
							goto l89
						}
						position++
						goto l86
					l89:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune('=') {
							goto l90
						}
						position++
						goto l86
					l90:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune(' ') {
							goto l91
						}
						position++
						goto l86
					l91:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune('.') {
							goto l92
						}
						position++
						goto l86
					l92:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune(' ') {
							goto l93
						}
						position++
						goto l86
					l93:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune('~') {
							goto l94
						}
						position++
						goto l86
					l94:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune(' ') {
							goto l95
						}
						position++
						goto l86
					l95:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune('^') {
							goto l96
						}
						position++
						goto l86
					l96:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune(' ') {
							goto l97
						}
						position++
						goto l86
					l97:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune('\\') {
							goto l98
						}
						position++
						goto l86
					l98:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune(' ') {
							goto l99
						}
						position++
						goto l86
					l99:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune('`') {
							goto l100
						}
						position++
						goto l86
					l100:
						position, tokenIndex = position86, tokenIndex86
						if buffer[position] != rune(' ') {
							goto l79
						}
						position++
					}
				l86:
				}
			l83:
			l81:
				{
					position82, tokenIndex82 := position, tokenIndex
					{
						position101, tokenIndex101 := position, tokenIndex
						if !_rules[ruleAlphaNum]() {
							goto l102
						}
						goto l101
					l102:
						position, tokenIndex = position101, tokenIndex101
						if !_rules[ruleSquareBrackets]() {
							goto l103
						}
						goto l101
					l103:
						position, tokenIndex = position101, tokenIndex101
						{
							position104, tokenIndex104 := position, tokenIndex
							if buffer[position] != rune(' ') {
								goto l105
							}
							position++
							goto l104
						l105:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune('_') {
								goto l106
							}
							position++
							goto l104
						l106:
							position, tokenIndex = position104, tokenIndex104
							if c := buffer[position]; c < rune(' ') || c > rune(' ') {
								goto l107
							}
							position++
							goto l104
						l107:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune('=') {
								goto l108
							}
							position++
							goto l104
						l108:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune(' ') {
								goto l109
							}
							position++
							goto l104
						l109:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune('.') {
								goto l110
							}
							position++
							goto l104
						l110:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune(' ') {
								goto l111
							}
							position++
							goto l104
						l111:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune('~') {
								goto l112
							}
							position++
							goto l104
						l112:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune(' ') {
								goto l113
							}
							position++
							goto l104
						l113:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune('^') {
								goto l114
							}
							position++
							goto l104
						l114:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune(' ') {
								goto l115
							}
							position++
							goto l104
						l115:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune('\\') {
								goto l116
							}
							position++
							goto l104
						l116:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune(' ') {
								goto l117
							}
							position++
							goto l104
						l117:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune('`') {
								goto l118
							}
							position++
							goto l104
						l118:
							position, tokenIndex = position104, tokenIndex104
							if buffer[position] != rune(' ') {
								goto l82
							}
							position++
						}
					l104:
					}
				l101:
					goto l81
				l82:
					position, tokenIndex = position82, tokenIndex82
				}
				add(ruleUser, position80)
			}
			return true
		l79:
			position, tokenIndex = position79, tokenIndex79
			return false
		},
		/* 6 Host <- <('@' (AlphaNum / SquareBrackets / (' ' / '_' / [ - ] / '.' / ' ' / '|' / ' ' / ':' / ' ' / '/' / ' '))+)> */
		func() bool {
			position119, tokenIndex119 := position, tokenIndex
			{
				position120 := position
				if buffer[position] != rune('@') {
					goto l119
				}
				position++
				{
					position123, tokenIndex123 := position, tokenIndex
					if !_rules[ruleAlphaNum]() {
						goto l124
					}
					goto l123
				l124:
					position, tokenIndex = position123, tokenIndex123
					if !_rules[ruleSquareBrackets]() {
						goto l125
					}
					goto l123
				l125:
					position, tokenIndex = position123, tokenIndex123
					{
						position126, tokenIndex126 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l127
						}
						position++
						goto l126
					l127:
						position, tokenIndex = position126, tokenIndex126
						if buffer[position] != rune('_') {
							goto l128
						}
						position++
						goto l126
					l128:
						position, tokenIndex = position126, tokenIndex126
						if c := buffer[position]; c < rune(' ') || c > rune(' ') {
							goto l129
						}
						position++
						goto l126
					l129:
						position, tokenIndex = position126, tokenIndex126
						if buffer[position] != rune('.') {
							goto l130
						}
						position++
						goto l126
					l130:
						position, tokenIndex = position126, tokenIndex126
						if buffer[position] != rune(' ') {
							goto l131
						}
						position++
						goto l126
					l131:
						position, tokenIndex = position126, tokenIndex126
						if buffer[position] != rune('|') {
							goto l132
						}
						position++
						goto l126
					l132:
						position, tokenIndex = position126, tokenIndex126
						if buffer[position] != rune(' ') {
							goto l133
						}
						position++
						goto l126
					l133:
						position, tokenIndex = position126, tokenIndex126
						if buffer[position] != rune(':') {
							goto l134
						}
						position++
						goto l126
					l134:
						position, tokenIndex = position126, tokenIndex126
						if buffer[position] != rune(' ') {
							goto l135
						}
						position++
						goto l126
					l135:
						position, tokenIndex = position126, tokenIndex126
						if buffer[position] != rune('/') {
							goto l136
						}
						position++
						goto l126
					l136:
						position, tokenIndex = position126, tokenIndex126
						if buffer[position] != rune(' ') {
							goto l119
						}
						position++
					}
				l126:
				}
			l123:
			l121:
				{
					position122, tokenIndex122 := position, tokenIndex
					{
						position137, tokenIndex137 := position, tokenIndex
						if !_rules[ruleAlphaNum]() {
							goto l138
						}
						goto l137
					l138:
						position, tokenIndex = position137, tokenIndex137
						if !_rules[ruleSquareBrackets]() {
							goto l139
						}
						goto l137
					l139:
						position, tokenIndex = position137, tokenIndex137
						{
							position140, tokenIndex140 := position, tokenIndex
							if buffer[position] != rune(' ') {
								goto l141
							}
							position++
							goto l140
						l141:
							position, tokenIndex = position140, tokenIndex140
							if buffer[position] != rune('_') {
								goto l142
							}
							position++
							goto l140
						l142:
							position, tokenIndex = position140, tokenIndex140
							if c := buffer[position]; c < rune(' ') || c > rune(' ') {
								goto l143
							}
							position++
							goto l140
						l143:
							position, tokenIndex = position140, tokenIndex140
							if buffer[position] != rune('.') {
								goto l144
							}
							position++
							goto l140
						l144:
							position, tokenIndex = position140, tokenIndex140
							if buffer[position] != rune(' ') {
								goto l145
							}
							position++
							goto l140
						l145:
							position, tokenIndex = position140, tokenIndex140
							if buffer[position] != rune('|') {
								goto l146
							}
							position++
							goto l140
						l146:
							position, tokenIndex = position140, tokenIndex140
							if buffer[position] != rune(' ') {
								goto l147
							}
							position++
							goto l140
						l147:
							position, tokenIndex = position140, tokenIndex140
							if buffer[position] != rune(':') {
								goto l148
							}
							position++
							goto l140
						l148:
							position, tokenIndex = position140, tokenIndex140
							if buffer[position] != rune(' ') {
								goto l149
							}
							position++
							goto l140
						l149:
							position, tokenIndex = position140, tokenIndex140
							if buffer[position] != rune('/') {
								goto l150
							}
							position++
							goto l140
						l150:
							position, tokenIndex = position140, tokenIndex140
							if buffer[position] != rune(' ') {
								goto l122
							}
							position++
						}
					l140:
					}
				l137:
					goto l121
				l122:
					position, tokenIndex = position122, tokenIndex122
				}
				add(ruleHost, position120)
			}
			return true
		l119:
			position, tokenIndex = position119, tokenIndex119
			return false
		},
		/* 7 Command <- <(Letter+ / ([0-9] [0-9] [0-9]))> */
		func() bool {
			position151, tokenIndex151 := position, tokenIndex
			{
				position152 := position
				{
					position153, tokenIndex153 := position, tokenIndex
					if !_rules[ruleLetter]() {
						goto l154
					}
				l155:
					{
						position156, tokenIndex156 := position, tokenIndex
						if !_rules[ruleLetter]() {
							goto l156
						}
						goto l155
					l156:
						position, tokenIndex = position156, tokenIndex156
					}
					goto l153
				l154:
					position, tokenIndex = position153, tokenIndex153
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l151
					}
					position++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l151
					}
					position++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l151
					}
					position++
				}
			l153:
				add(ruleCommand, position152)
			}
			return true
		l151:
			position, tokenIndex = position151, tokenIndex151
			return false
		},
		/* 8 Params <- <(S Middle* Trailing?)> */
		func() bool {
			position157, tokenIndex157 := position, tokenIndex
			{
				position158 := position
				if !_rules[ruleS]() {
					goto l157
				}
			l159:
				{
					position160, tokenIndex160 := position, tokenIndex
					if !_rules[ruleMiddle]() {
						goto l160
					}
					goto l159
				l160:
					position, tokenIndex = position160, tokenIndex160
				}
				{
					position161, tokenIndex161 := position, tokenIndex
					if !_rules[ruleTrailing]() {
						goto l161
					}
					goto l162
				l161:
					position, tokenIndex = position161, tokenIndex161
				}
			l162:
				add(ruleParams, position158)
			}
			return true
		l157:
			position, tokenIndex = position157, tokenIndex157
			return false
		},
		/* 9 Middle <- <((!':' !S NonTerminating)+ S?)> */
		func() bool {
			position163, tokenIndex163 := position, tokenIndex
			{
				position164 := position
				{
					position167, tokenIndex167 := position, tokenIndex
					if buffer[position] != rune(':') {
						goto l167
					}
					position++
					goto l163
				l167:
					position, tokenIndex = position167, tokenIndex167
				}
				{
					position168, tokenIndex168 := position, tokenIndex
					if !_rules[ruleS]() {
						goto l168
					}
					goto l163
				l168:
					position, tokenIndex = position168, tokenIndex168
				}
				if !_rules[ruleNonTerminating]() {
					goto l163
				}
			l165:
				{
					position166, tokenIndex166 := position, tokenIndex
					{
						position169, tokenIndex169 := position, tokenIndex
						if buffer[position] != rune(':') {
							goto l169
						}
						position++
						goto l166
					l169:
						position, tokenIndex = position169, tokenIndex169
					}
					{
						position170, tokenIndex170 := position, tokenIndex
						if !_rules[ruleS]() {
							goto l170
						}
						goto l166
					l170:
						position, tokenIndex = position170, tokenIndex170
					}
					if !_rules[ruleNonTerminating]() {
						goto l166
					}
					goto l165
				l166:
					position, tokenIndex = position166, tokenIndex166
				}
				{
					position171, tokenIndex171 := position, tokenIndex
					if !_rules[ruleS]() {
						goto l171
					}
					goto l172
				l171:
					position, tokenIndex = position171, tokenIndex171
				}
			l172:
				add(ruleMiddle, position164)
			}
			return true
		l163:
			position, tokenIndex = position163, tokenIndex163
			return false
		},
		/* 10 Trailing <- <(':' (NonTerminating / S)*)> */
		func() bool {
			position173, tokenIndex173 := position, tokenIndex
			{
				position174 := position
				if buffer[position] != rune(':') {
					goto l173
				}
				position++
			l175:
				{
					position176, tokenIndex176 := position, tokenIndex
					{
						position177, tokenIndex177 := position, tokenIndex
						if !_rules[ruleNonTerminating]() {
							goto l178
						}
						goto l177
					l178:
						position, tokenIndex = position177, tokenIndex177
						if !_rules[ruleS]() {
							goto l176
						}
					}
				l177:
					goto l175
				l176:
					position, tokenIndex = position176, tokenIndex176
				}
				add(ruleTrailing, position174)
			}
			return true
		l173:
			position, tokenIndex = position173, tokenIndex173
			return false
		},
		/* 11 NonTerminating <- <(!(' ' / 'U' / '+' / '0' / '0' / '0' / 'A' / ' ' / 'U' / '+' / '0' / '0' / '0' / 'D' / ' ') .)> */
		func() bool {
			position179, tokenIndex179 := position, tokenIndex
			{
				position180 := position
				{
					position181, tokenIndex181 := position, tokenIndex
					{
						position182, tokenIndex182 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l183
						}
						position++
						goto l182
					l183:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('U') {
							goto l184
						}
						position++
						goto l182
					l184:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('+') {
							goto l185
						}
						position++
						goto l182
					l185:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('0') {
							goto l186
						}
						position++
						goto l182
					l186:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('0') {
							goto l187
						}
						position++
						goto l182
					l187:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('0') {
							goto l188
						}
						position++
						goto l182
					l188:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('A') {
							goto l189
						}
						position++
						goto l182
					l189:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune(' ') {
							goto l190
						}
						position++
						goto l182
					l190:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('U') {
							goto l191
						}
						position++
						goto l182
					l191:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('+') {
							goto l192
						}
						position++
						goto l182
					l192:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('0') {
							goto l193
						}
						position++
						goto l182
					l193:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('0') {
							goto l194
						}
						position++
						goto l182
					l194:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('0') {
							goto l195
						}
						position++
						goto l182
					l195:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune('D') {
							goto l196
						}
						position++
						goto l182
					l196:
						position, tokenIndex = position182, tokenIndex182
						if buffer[position] != rune(' ') {
							goto l181
						}
						position++
					}
				l182:
					goto l179
				l181:
					position, tokenIndex = position181, tokenIndex181
				}
				if !matchDot() {
					goto l179
				}
				add(ruleNonTerminating, position180)
			}
			return true
		l179:
			position, tokenIndex = position179, tokenIndex179
			return false
		},
		/* 12 SquareBrackets <- <(' ' / '[' / ' ' / 'U' / '+' / '0' / '0' / '5' / 'D' / ' ')> */
		func() bool {
			position197, tokenIndex197 := position, tokenIndex
			{
				position198 := position
				{
					position199, tokenIndex199 := position, tokenIndex
					if buffer[position] != rune(' ') {
						goto l200
					}
					position++
					goto l199
				l200:
					position, tokenIndex = position199, tokenIndex199
					if buffer[position] != rune('[') {
						goto l201
					}
					position++
					goto l199
				l201:
					position, tokenIndex = position199, tokenIndex199
					if buffer[position] != rune(' ') {
						goto l202
					}
					position++
					goto l199
				l202:
					position, tokenIndex = position199, tokenIndex199
					if buffer[position] != rune('U') {
						goto l203
					}
					position++
					goto l199
				l203:
					position, tokenIndex = position199, tokenIndex199
					if buffer[position] != rune('+') {
						goto l204
					}
					position++
					goto l199
				l204:
					position, tokenIndex = position199, tokenIndex199
					if buffer[position] != rune('0') {
						goto l205
					}
					position++
					goto l199
				l205:
					position, tokenIndex = position199, tokenIndex199
					if buffer[position] != rune('0') {
						goto l206
					}
					position++
					goto l199
				l206:
					position, tokenIndex = position199, tokenIndex199
					if buffer[position] != rune('5') {
						goto l207
					}
					position++
					goto l199
				l207:
					position, tokenIndex = position199, tokenIndex199
					if buffer[position] != rune('D') {
						goto l208
					}
					position++
					goto l199
				l208:
					position, tokenIndex = position199, tokenIndex199
					if buffer[position] != rune(' ') {
						goto l197
					}
					position++
				}
			l199:
				add(ruleSquareBrackets, position198)
			}
			return true
		l197:
			position, tokenIndex = position197, tokenIndex197
			return false
		},
		/* 13 CurlyBrackets <- <(' ' / '{' / ' ' / '}' / ' ')> */
		func() bool {
			position209, tokenIndex209 := position, tokenIndex
			{
				position210 := position
				{
					position211, tokenIndex211 := position, tokenIndex
					if buffer[position] != rune(' ') {
						goto l212
					}
					position++
					goto l211
				l212:
					position, tokenIndex = position211, tokenIndex211
					if buffer[position] != rune('{') {
						goto l213
					}
					position++
					goto l211
				l213:
					position, tokenIndex = position211, tokenIndex211
					if buffer[position] != rune(' ') {
						goto l214
					}
					position++
					goto l211
				l214:
					position, tokenIndex = position211, tokenIndex211
					if buffer[position] != rune('}') {
						goto l215
					}
					position++
					goto l211
				l215:
					position, tokenIndex = position211, tokenIndex211
					if buffer[position] != rune(' ') {
						goto l209
					}
					position++
				}
			l211:
				add(ruleCurlyBrackets, position210)
			}
			return true
		l209:
			position, tokenIndex = position209, tokenIndex209
			return false
		},
		/* 14 Backtick <- <(' ' / 'U' / '+' / '0' / '0' / '6' / '0' / ' ')> */
		func() bool {
			position216, tokenIndex216 := position, tokenIndex
			{
				position217 := position
				{
					position218, tokenIndex218 := position, tokenIndex
					if buffer[position] != rune(' ') {
						goto l219
					}
					position++
					goto l218
				l219:
					position, tokenIndex = position218, tokenIndex218
					if buffer[position] != rune('U') {
						goto l220
					}
					position++
					goto l218
				l220:
					position, tokenIndex = position218, tokenIndex218
					if buffer[position] != rune('+') {
						goto l221
					}
					position++
					goto l218
				l221:
					position, tokenIndex = position218, tokenIndex218
					if buffer[position] != rune('0') {
						goto l222
					}
					position++
					goto l218
				l222:
					position, tokenIndex = position218, tokenIndex218
					if buffer[position] != rune('0') {
						goto l223
					}
					position++
					goto l218
				l223:
					position, tokenIndex = position218, tokenIndex218
					if buffer[position] != rune('6') {
						goto l224
					}
					position++
					goto l218
				l224:
					position, tokenIndex = position218, tokenIndex218
					if buffer[position] != rune('0') {
						goto l225
					}
					position++
					goto l218
				l225:
					position, tokenIndex = position218, tokenIndex218
					if buffer[position] != rune(' ') {
						goto l216
					}
					position++
				}
			l218:
				add(ruleBacktick, position217)
			}
			return true
		l216:
			position, tokenIndex = position216, tokenIndex216
			return false
		},
		/* 15 AlphaNum <- <(Letter / Num)> */
		func() bool {
			position226, tokenIndex226 := position, tokenIndex
			{
				position227 := position
				{
					position228, tokenIndex228 := position, tokenIndex
					if !_rules[ruleLetter]() {
						goto l229
					}
					goto l228
				l229:
					position, tokenIndex = position228, tokenIndex228
					if !_rules[ruleNum]() {
						goto l226
					}
				}
			l228:
				add(ruleAlphaNum, position227)
			}
			return true
		l226:
			position, tokenIndex = position226, tokenIndex226
			return false
		},
		/* 16 Letter <- <(' ' / [a-z] / ' ' / (' ' / [A-Z] / ' ') / ':' / '@')> */
		func() bool {
			position230, tokenIndex230 := position, tokenIndex
			{
				position231 := position
				{
					position232, tokenIndex232 := position, tokenIndex
					if buffer[position] != rune(' ') {
						goto l233
					}
					position++
					goto l232
				l233:
					position, tokenIndex = position232, tokenIndex232
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l234
					}
					position++
					goto l232
				l234:
					position, tokenIndex = position232, tokenIndex232
					if buffer[position] != rune(' ') {
						goto l235
					}
					position++
					goto l232
				l235:
					position, tokenIndex = position232, tokenIndex232
					{
						position237, tokenIndex237 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l238
						}
						position++
						goto l237
					l238:
						position, tokenIndex = position237, tokenIndex237
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l239
						}
						position++
						goto l237
					l239:
						position, tokenIndex = position237, tokenIndex237
						if buffer[position] != rune(' ') {
							goto l236
						}
						position++
					}
				l237:
					goto l232
				l236:
					position, tokenIndex = position232, tokenIndex232
					if buffer[position] != rune(':') {
						goto l240
					}
					position++
					goto l232
				l240:
					position, tokenIndex = position232, tokenIndex232
					if buffer[position] != rune('@') {
						goto l230
					}
					position++
				}
			l232:
				add(ruleLetter, position231)
			}
			return true
		l230:
			position, tokenIndex = position230, tokenIndex230
			return false
		},
		/* 17 Num <- <(' ' / [0-9] / ' ')> */
		func() bool {
			position241, tokenIndex241 := position, tokenIndex
			{
				position242 := position
				{
					position243, tokenIndex243 := position, tokenIndex
					if buffer[position] != rune(' ') {
						goto l244
					}
					position++
					goto l243
				l244:
					position, tokenIndex = position243, tokenIndex243
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l245
					}
					position++
					goto l243
				l245:
					position, tokenIndex = position243, tokenIndex243
					if buffer[position] != rune(' ') {
						goto l241
					}
					position++
				}
			l243:
				add(ruleNum, position242)
			}
			return true
		l241:
			position, tokenIndex = position241, tokenIndex241
			return false
		},
		/* 18 CRLF <- <((' ' / 'U' / '+' / '0' / '0' / '0' / 'D' / ' ') (' ' / 'U' / '+' / '0' / '0' / '0' / 'A' / ' '))> */
		func() bool {
			position246, tokenIndex246 := position, tokenIndex
			{
				position247 := position
				{
					position248, tokenIndex248 := position, tokenIndex
					if buffer[position] != rune(' ') {
						goto l249
					}
					position++
					goto l248
				l249:
					position, tokenIndex = position248, tokenIndex248
					if buffer[position] != rune('U') {
						goto l250
					}
					position++
					goto l248
				l250:
					position, tokenIndex = position248, tokenIndex248
					if buffer[position] != rune('+') {
						goto l251
					}
					position++
					goto l248
				l251:
					position, tokenIndex = position248, tokenIndex248
					if buffer[position] != rune('0') {
						goto l252
					}
					position++
					goto l248
				l252:
					position, tokenIndex = position248, tokenIndex248
					if buffer[position] != rune('0') {
						goto l253
					}
					position++
					goto l248
				l253:
					position, tokenIndex = position248, tokenIndex248
					if buffer[position] != rune('0') {
						goto l254
					}
					position++
					goto l248
				l254:
					position, tokenIndex = position248, tokenIndex248
					if buffer[position] != rune('D') {
						goto l255
					}
					position++
					goto l248
				l255:
					position, tokenIndex = position248, tokenIndex248
					if buffer[position] != rune(' ') {
						goto l246
					}
					position++
				}
			l248:
				{
					position256, tokenIndex256 := position, tokenIndex
					if buffer[position] != rune(' ') {
						goto l257
					}
					position++
					goto l256
				l257:
					position, tokenIndex = position256, tokenIndex256
					if buffer[position] != rune('U') {
						goto l258
					}
					position++
					goto l256
				l258:
					position, tokenIndex = position256, tokenIndex256
					if buffer[position] != rune('+') {
						goto l259
					}
					position++
					goto l256
				l259:
					position, tokenIndex = position256, tokenIndex256
					if buffer[position] != rune('0') {
						goto l260
					}
					position++
					goto l256
				l260:
					position, tokenIndex = position256, tokenIndex256
					if buffer[position] != rune('0') {
						goto l261
					}
					position++
					goto l256
				l261:
					position, tokenIndex = position256, tokenIndex256
					if buffer[position] != rune('0') {
						goto l262
					}
					position++
					goto l256
				l262:
					position, tokenIndex = position256, tokenIndex256
					if buffer[position] != rune('A') {
						goto l263
					}
					position++
					goto l256
				l263:
					position, tokenIndex = position256, tokenIndex256
					if buffer[position] != rune(' ') {
						goto l246
					}
					position++
				}
			l256:
				add(ruleCRLF, position247)
			}
			return true
		l246:
			position, tokenIndex = position246, tokenIndex246
			return false
		},
		/* 19 S <- <(' ' / 'U' / '+' / '0' / '0' / '2' / '0')+> */
		func() bool {
			position264, tokenIndex264 := position, tokenIndex
			{
				position265 := position
				{
					position268, tokenIndex268 := position, tokenIndex
					if buffer[position] != rune(' ') {
						goto l269
					}
					position++
					goto l268
				l269:
					position, tokenIndex = position268, tokenIndex268
					if buffer[position] != rune('U') {
						goto l270
					}
					position++
					goto l268
				l270:
					position, tokenIndex = position268, tokenIndex268
					if buffer[position] != rune('+') {
						goto l271
					}
					position++
					goto l268
				l271:
					position, tokenIndex = position268, tokenIndex268
					if buffer[position] != rune('0') {
						goto l272
					}
					position++
					goto l268
				l272:
					position, tokenIndex = position268, tokenIndex268
					if buffer[position] != rune('0') {
						goto l273
					}
					position++
					goto l268
				l273:
					position, tokenIndex = position268, tokenIndex268
					if buffer[position] != rune('2') {
						goto l274
					}
					position++
					goto l268
				l274:
					position, tokenIndex = position268, tokenIndex268
					if buffer[position] != rune('0') {
						goto l264
					}
					position++
				}
			l268:
			l266:
				{
					position267, tokenIndex267 := position, tokenIndex
					{
						position275, tokenIndex275 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l276
						}
						position++
						goto l275
					l276:
						position, tokenIndex = position275, tokenIndex275
						if buffer[position] != rune('U') {
							goto l277
						}
						position++
						goto l275
					l277:
						position, tokenIndex = position275, tokenIndex275
						if buffer[position] != rune('+') {
							goto l278
						}
						position++
						goto l275
					l278:
						position, tokenIndex = position275, tokenIndex275
						if buffer[position] != rune('0') {
							goto l279
						}
						position++
						goto l275
					l279:
						position, tokenIndex = position275, tokenIndex275
						if buffer[position] != rune('0') {
							goto l280
						}
						position++
						goto l275
					l280:
						position, tokenIndex = position275, tokenIndex275
						if buffer[position] != rune('2') {
							goto l281
						}
						position++
						goto l275
					l281:
						position, tokenIndex = position275, tokenIndex275
						if buffer[position] != rune('0') {
							goto l267
						}
						position++
					}
				l275:
					goto l266
				l267:
					position, tokenIndex = position267, tokenIndex267
				}
				add(ruleS, position265)
			}
			return true
		l264:
			position, tokenIndex = position264, tokenIndex264
			return false
		},
		/* 21 Action0 <- <{fmt.Println(buffer[begin:end])}> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
	}
	p.rules = _rules
}
