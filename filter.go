package toml

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

var (
	ErrDontAdvance    = fmt.Errorf(`dont-advance`)
	ErrNotImplemented = fmt.Errorf(`not-implemented`)
	EOF               = '\255' // invalid utf 8 character

	bin = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     48,
				Hi:     49,
				Stride: 1,
			},
		},
	}

	binRanges = []*unicode.RangeTable{
		bin,
	}

	octal = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     48,
				Hi:     55,
				Stride: 1,
			},
		},
	}

	octalRanges = []*unicode.RangeTable{
		octal,
	}

	digits = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     48,
				Hi:     57,
				Stride: 1,
			},
		},
	}

	letters = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     65,
				Hi:     90,
				Stride: 1,
			},
			{
				Lo:     97,
				Hi:     122,
				Stride: 1,
			},
		},
	}

	dash = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     45,
				Hi:     45,
				Stride: 1,
			},
		},
	}

	underscore = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     95,
				Hi:     95,
				Stride: 1,
			},
		},
	}

	bareRanges = []*unicode.RangeTable{
		digits,
		letters,
		dash,
		underscore,
	}

	hex = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     65,
				Hi:     70,
				Stride: 1,
			},
			{
				Lo:     97,
				Hi:     102,
				Stride: 1,
			},
		},
	}

	hexRanges = []*unicode.RangeTable{
		digits,
		hex,
	}

	digitRanges = []*unicode.RangeTable{
		{
			R16: []unicode.Range16{
				{
					Lo:     48,
					Hi:     57,
					Stride: 1,
				},
			},
		},
	}

	notAllowedStringRanges = []*unicode.RangeTable{
		{
			R16: []unicode.Range16{
				{ // backspace
					Lo:     8,
					Hi:     8,
					Stride: 1,
				},

				{ // \n
					Lo:     10,
					Hi:     10,
					Stride: 1,
				},

				{ // form feed
					Lo:     12,
					Hi:     12,
					Stride: 1,
				},
			},
		},
	}

	allowedMultiStringRanges = []*unicode.RangeTable{
		{
			R16: []unicode.Range16{
				{ // tab & \n
					Lo:     9,
					Hi:     10,
					Stride: 1,
				},
				{ // form feed
					Lo:     12,
					Hi:     12,
					Stride: 1,
				},
				{ // space
					Lo:     32,
					Hi:     32,
					Stride: 1,
				},
			},
		},
	}

	escapeCharacters = []rune{'\\', 'b', 't', 'n', 'f', 'r', '"'}

	daysInMonth = []int{31 /*jan*/, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
)

func toJSONString(r rune) string {

	if r == '"' {
		return `\"`
	}

	if r == '\\' {
		return `\\`
	}

	if r == '/' {
		return `\/`
	}

	if r == '\b' {
		return `\b`
	}

	if r == '\f' {
		return `\f`
	}

	if r == '\n' {
		return `\n`
	}

	if r == '\r' {
		return `\r`
	}

	if r == '\t' {
		return `\t`
	}

	return string(r)
}

type Token string

var (
	OTHERT Token
	QQQQT  Token = `""""`
	QQQT   Token = `"""`
	QQT    Token = `""`
	QT     Token = `"`
	SQQQQT Token = `''''`
	SQQQT  Token = `'''`
	SQQT   Token = `''`
	SQT    Token = `'`
	ESCT   Token = `escape`

	OBT  Token = `[`  // open bracket
	OBBT Token = `[[` // open bracket
	CBT  Token = `]`
	CBBT Token = `]]`

	DIGITT Token = `digit`
	SIGNT  Token = `sign`
	DOTT   Token = `dot`
	COMT   Token = `comma`

	UNDERT Token = `underscore`
	EXPT   Token = `exp`
	SPACET Token = `space`
)

type ScopeState string

var (
	OtherState ScopeState
	InitState  ScopeState = `init`
	DoneState  ScopeState = `done`

	AfterInitialZeroState ScopeState = `after-initial-zero`
	AfterDotState         ScopeState = `after-dot`
	AfterExpState         ScopeState = `after-exp`
	AfterEOLState         ScopeState = `after-eol`
	AfterEOLReturnState   ScopeState = `after-eol-return`

	AfterTableState ScopeState = `after-table-state`
	AfterArrayState ScopeState = `after-array-state`

	AfterKeyState        ScopeState = `after-key`
	AfterValueState      ScopeState = `after-value`
	AfterFirstValueState ScopeState = `after-first-value`

	AfterQuoteState ScopeState = `after-quote`
	AfterTimeState  ScopeState = `after-time`
	AfterTState     ScopeState = `after-T`
)

type ScopeType string

var (
	OtherType  ScopeType = `other`
	StringType ScopeType = `string`
	KeyType    ScopeType = `key`
)

type Filter struct {
	state State
	buf   *bytes.Buffer
}

func NewFilter() *Filter {

	state := State{
		defs: Defs{m: map[string]Var{},
			arrayKeyStack: &ArrayKeyStack{},
			keyFilter:     &KeyFilter{},
		},
		buf: bytes.NewBufferString(`{`),
	}

	state.PushScope(Top, OtherType, nil)

	return &Filter{buf: &bytes.Buffer{}, state: state}
}

func (f *Filter) Write(p []byte) (int, error) {

	f.buf.Write(p)

	for {
		r, _, err := f.buf.ReadRune()
		if errors.Is(err, io.EOF) {
			break
		}

		if r == '\r' {
			continue
		}

		if r == '\n' {
			f.state.line++
			f.state.position = 0
		} else {
			f.state.position++
		}

		if len(f.state.scopes) == 0 {
			break
		}

		if f.state.inComment {
			if r == '\n' {
				f.state.inComment = false
			}

			if f.state.inComment {
				continue
			}
		}

		if f.state.PeekScope().scopeType == OtherType && r == '#' {
			f.state.inComment = true
			continue
		}
		err = f.WriteRune(r)
		if err != nil {
			return 0, err
		}
	}
	f.buf.Truncate(f.buf.Len())
	return len(p), nil
}

func (f *Filter) WriteRune(r rune) error {

	for {
		sc := f.state.PeekScope()
		if sc != nil {
			err := sc.Parse(r, &f.state)
			if errors.Is(err, ErrDontAdvance) {
				continue
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func (f *Filter) Close() {
	f.state.defs.keyFilter.Close(f.state.buf)
	f.state.buf.WriteRune('}')
}

type Scope struct {
	state     ScopeState
	key       []string
	scopeType ScopeType
	counter   int
	val       []rune
	lastToken Token
	parseFunc ParseFunc
}

func (s *Scope) Parse(r rune, state *State) error {
	return s.parseFunc(r, state, s)
}

type ParseFunc func(r rune, state *State, scope *Scope) error

type State struct {
	scopes    []Scope
	defs      Defs
	line      int
	position  int
	inComment bool
	data      []rune
	buf       *bytes.Buffer
}

func (s *State) PushScope(parse ParseFunc, scopeType ScopeType, thisScope *Scope) {

	if thisScope != nil {
		s.scopes[len(s.scopes)-1] = *thisScope
	}
	s.scopes = append(s.scopes, Scope{parseFunc: parse, scopeType: scopeType})
}

func (s *State) PopScope() *Scope {
	if len(s.scopes) > 0 {
		sc := s.scopes[len(s.scopes)-1]
		s.scopes = s.scopes[:len(s.scopes)-1]
		return &sc
	}
	return nil
}

func (s *State) PeekScope() *Scope {

	if len(s.scopes) > 0 {
		return &s.scopes[len(s.scopes)-1]
	}
	return nil
}

func (s *State) ResetData() {
	if s.data != nil {
		s.data = s.data[0:0]
	}
}

func (s *State) ExtractKeys() []string {
	rawKey := string(s.data)
	s.ResetData()
	return strings.Split(rawKey, "\n")
}

func fullKey(key []string) string {
	return strings.Join(key, "\n")
}

type Var string

var (
	BasicVar Var = `basic-var`
	TableVar Var = `table-var`
	ArrayVar Var = `array-var`
)

type DefineFunc func(key []string, v Var) bool

type Defs struct {
	m             map[string]Var
	arrayKeyStack *ArrayKeyStack
	keyFilter     *KeyFilter
}

func (d Defs) Define(key []string, v Var) bool {

	if len(key) == 0 {
		return false
	}

	buf := bytes.NewBufferString(key[0])
	for i := 1; i < len(key); i++ {

		subKey := buf.String()

		if sv, ok := d.m[subKey]; ok {
			if sv == BasicVar {
				return false
			}
		} else {
			d.m[subKey] = TableVar
		}

		buf.WriteString("\n")
		buf.WriteString(key[i])
	}

	fullKey := buf.String()

	fv, ok := d.m[fullKey]

	if ok && fv != ArrayVar {
		return false
	}

	if !ok {
		d.m[fullKey] = v
	}

	toClose := d.arrayKeyStack.Push(fullKey, v)
	for _, k := range toClose {

		if fullKey != k || v != ArrayVar {
			d.m[k] = BasicVar
		}

		for key := range d.m {
			if k != key && strings.HasPrefix(key, k) {
				delete(d.m, key)
			}
		}
	}

	return true
}

func BaseKeyDefs(baseKey []string, defs Defs) DefineFunc {

	return func(key []string, v Var) bool {
		key = append(baseKey, key...)
		return defs.Define(key, v)
	}
}

func validUnicode(code int64) bool {

	if code >= 0 &&
		code <= 0xD7FF16 {
		return true
	}
	if code >= 0xE00016 &&
		code <= 0x10FFFF {
		return true
	}
	return false
}

func ShortUnicode(r rune, state *State, scope *Scope) error {

	if !unicode.IsOneOf(hexRanges, r) {
		return parseError(state, `invalid digit`)
	}

	scope.counter++

	scope.val = append(scope.val, r)

	r = unicode.ToUpper(r)
	if scope.scopeType == KeyType {
		state.data = append(state.data, r)
	} else {
		state.buf.WriteRune(r)
	}

	if scope.counter == 4 {

		v, err := strconv.ParseInt(string(scope.val), 16, 64)
		if err != nil {
			return parseError(state, `invalid number`)
		}

		if ok := validUnicode(v); !ok {
			return parseError(state, `invalid code`)
		}

		state.PopScope()
		return nil
	}
	return nil
}

func Unicode(r rune, state *State, scope *Scope) error {

	if !unicode.IsOneOf(hexRanges, r) {
		return parseError(state, `invalid digit`)
	}

	scope.counter++

	scope.val = append(scope.val, r)

	r = unicode.ToUpper(r)
	if scope.scopeType == KeyType {
		state.data = append(state.data, r)
	} else {
		state.buf.WriteRune(r)
	}

	if scope.counter == 6 {

		v, err := strconv.ParseInt(string(scope.val), 16, 64)
		if err != nil {
			return parseError(state, `invalid number`)
		}

		if ok := validUnicode(v); !ok {
			return parseError(state, `invalid code`)
		}

		state.PopScope()
		return nil
	}
	return nil
}

func QuotedString(r rune, state *State, scope *Scope) error {

	if unicode.IsOneOf(notAllowedStringRanges, r) {
		return parseError(state, `character not allowed in quoted string`)
	}

	if unicode.IsSpace(r) && r != '\t' && r != ' ' {
		return parseError(state, `character not allowed in quoted string`)
	}

	if scope.lastToken != ESCT && r == '\\' {
		scope.lastToken = ESCT
		return nil
	}

	if r == '"' && scope.lastToken != ESCT {
		state.PopScope()

		if scope.scopeType != KeyType {
			state.buf.WriteRune('"')
		}
		return nil
	}

	if scope.lastToken == ESCT && r == 'U' {
		if scope.scopeType == KeyType {
			state.data = append(state.data, '\\')
			state.data = append(state.data, '\\')
			state.data = append(state.data, 'U')
		} else {
			state.buf.WriteString("\\U")
		}

		scope.lastToken = OTHERT
		state.PushScope(Unicode, scope.scopeType, scope)
		return nil
	}

	if scope.lastToken == ESCT && r == 'u' {
		if scope.scopeType == KeyType {
			state.data = append(state.data, '\\')
			state.data = append(state.data, 'u')
		} else {
			state.buf.WriteString("\\u")
		}

		scope.lastToken = OTHERT
		state.PushScope(ShortUnicode, scope.scopeType, scope)
		return nil
	}

	if scope.lastToken == ESCT && !isOneOf(r, escapeCharacters) {
		return parseError(state, `quoted string has unescaped backslash`)
	}
	scope.lastToken = OTHERT

	js := toJSONString(r)
	if scope.scopeType == KeyType {
		state.data = append(state.data, []rune(js)...)
	} else {
		state.buf.WriteString(js)
	}
	return nil
}

func TrippleQuotedString(r rune, state *State, scope *Scope) error {

	if unicode.IsSpace(r) && !unicode.IsOneOf(allowedMultiStringRanges, r) {
		return parseError(state, `disallowed character in """`)
	}

	if scope.state == DoneState {
		if r != '"' {
			if scope.lastToken == QQQQT {
				state.buf.WriteString(`\"`)
			}

			state.buf.WriteRune('"')
			state.PopScope()
			return ErrDontAdvance
		}

		if scope.lastToken == QQQT {
			scope.lastToken = QQQQT
			return nil
		}

		state.buf.WriteString(`\"\""`)
		state.PopScope()
		return nil
	}

	if scope.state == AfterEOLReturnState {
		if !unicode.IsSpace(r) {
			scope.state = InitState
			return ErrDontAdvance
		}

		return nil
	}

	if scope.state == AfterEOLState {

		if r == '\n' {
			scope.state = AfterEOLReturnState
			return nil
		}

		if !unicode.IsSpace(r) {
			return parseError(state, `invalid character in EOL`)
		}
		return nil
	}

	st := scope.state
	if st == OtherState {
		scope.state = InitState
	}

	if st == OtherState && r == '\n' {
		return nil
	}

	if scope.lastToken != ESCT && r == '\\' {
		scope.lastToken = ESCT
		return nil
	}

	if scope.lastToken == ESCT && r == 'U' {
		scope.lastToken = OTHERT
		state.buf.WriteString(`\\U`)
		state.PushScope(Unicode, StringType, scope)
		return nil
	}

	if scope.lastToken == ESCT && r == 'u' {
		scope.lastToken = OTHERT
		state.buf.WriteString(`\u`)
		state.PushScope(ShortUnicode, StringType, scope)
		return nil
	}

	if scope.lastToken == ESCT && unicode.IsSpace(r) {
		scope.lastToken = OTHERT
		scope.state = AfterEOLState
		return ErrDontAdvance
	}

	if scope.lastToken == ESCT && r == '"' {
		scope.lastToken = OTHERT
		state.buf.WriteString(`\"`)
		return nil
	}

	if scope.lastToken == QQT && r == '"' {
		scope.lastToken = QQQT
		scope.state = DoneState
		return nil
	}

	if scope.lastToken == QT && r == '"' {
		scope.lastToken = QQT
		return nil
	}

	if scope.lastToken != ESCT && r == '"' {
		scope.lastToken = QT
		return nil
	}

	if scope.lastToken == ESCT && !isOneOf(r, escapeCharacters) {
		return parseError(state, `backslash \ not allowed`)
	}

	if scope.lastToken == ESCT {
		state.buf.WriteRune('/')
	}

	if scope.lastToken == QT {
		state.buf.WriteString(`\"`)
	}

	if scope.lastToken == QQT {
		state.buf.WriteString(`\"\"`)
	}

	scope.lastToken = OTHERT
	state.buf.WriteString(toJSONString(r))
	return nil
}

func LiteralString(r rune, state *State, scope *Scope) error {

	if r == '\n' {
		return parseError(state, `\n found in literal string`)
	}

	if r == '\'' {
		state.PopScope()

		if scope.scopeType != KeyType {
			state.buf.WriteRune('"')
		}
		return nil
	}

	js := toJSONString(r)
	if scope.scopeType == KeyType {
		state.data = append(state.data, []rune(js)...)
	} else {
		state.buf.WriteString(js)
	}
	return nil
}

func MultiLineLiteralString(r rune, state *State, scope *Scope) error {

	if scope.state == DoneState {
		if r != '\'' {
			if scope.lastToken == SQQQQT {
				state.buf.WriteRune('\'')
			}

			state.buf.WriteRune('"')
			state.PopScope()
			return ErrDontAdvance
		}

		if scope.lastToken == SQQQT {
			scope.lastToken = SQQQQT
			return nil
		}

		state.buf.WriteString(`''"`)
		state.PopScope()
		return nil
	}

	st := scope.state
	if st == OtherState {
		scope.state = InitState
	}

	if st == OtherState && r == '\n' {
		return nil
	}

	if scope.lastToken == SQQT && r == '\'' {
		scope.lastToken = SQQQT
		scope.state = DoneState
		return nil
	}

	if scope.lastToken == SQT && r == '\'' {
		scope.lastToken = SQQT
		return nil
	}

	if r == '\'' {
		scope.lastToken = SQT
		return nil
	}

	if scope.lastToken == SQT {
		state.buf.WriteRune('\'')
	}

	scope.lastToken = OTHERT
	state.buf.WriteString(toJSONString(r))
	return nil
}

func PrefixNumber(ranges []*unicode.RangeTable) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.state != InitState && unicode.IsSpace(r) {
			return parseError(state, `empty number`)
		}
		scope.state = InitState

		if unicode.IsSpace(r) || r == ']' || r == '}' || r == ',' {
			if scope.lastToken != DIGITT {
				return parseError(state, `invalid character at number end`)
			}
			state.PopScope()
			state.buf.WriteRune('"')
			return ErrDontAdvance
		}

		if r == '_' {
			if scope.lastToken != DIGITT {
				return parseError(state, `invalid character after underscore in number`)
			}
			scope.lastToken = UNDERT
			return nil
		}

		if !unicode.IsOneOf(ranges, r) {
			return parseError(state, `invalid character in number`)
		}

		scope.lastToken = DIGITT
		state.buf.WriteRune(unicode.ToUpper(r))
		return nil
	}
}

func floatDispatchSign(r rune, state *State, scope *Scope) (bool, error) {
	if r == '-' || r == '+' {
		if scope.state != OtherState &&
			scope.lastToken != EXPT {
			return false, parseError(state, `invalid sign in float`)
		}
		scope.lastToken = SIGNT
		return true, nil
	}
	return false, nil
}

func floatDispatchUnderscore(r rune, state *State, scope *Scope) (bool, error) {
	if r == '_' {
		if scope.lastToken != DIGITT {
			return false, parseError(state, `invalid '_' in float`)
		}

		scope.lastToken = UNDERT
		return true, nil
	}
	return false, nil
}

func floatDispatchExp(r rune, state *State, scope *Scope) (bool, error) {
	if r == 'e' || r == 'E' {
		if scope.lastToken != DIGITT {
			return false, parseError(state, `invalid 'e' in float`)
		}
		scope.lastToken = EXPT
		scope.val = append(scope.val, r)
		scope.state = AfterExpState
		return true, nil
	}
	return false, nil
}

func floatDispatchDigit(r rune, state *State, scope *Scope) error {
	if !unicode.IsOneOf(digitRanges, r) {
		return parseError(state, `invalid digit in float`)
	}
	scope.lastToken = DIGITT
	scope.counter++
	scope.val = append(scope.val, r)
	return nil
}

func Float(firstState ScopeState, vals []rune, counter int) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.lastToken == SIGNT {
			if r == 'n' {
				state.buf.WriteRune('"')
				if len(scope.val) > 0 {
					state.buf.WriteRune(scope.val[len(scope.val)-1])
				}
				state.buf.WriteString(`nan"`)

				state.PopScope()
				state.PushScope(LiteralValue([]rune(`nan`)), OtherType, nil)
				return ErrDontAdvance
			}
			if r == 'i' {
				state.buf.WriteRune('"')
				if len(scope.val) > 0 {
					state.buf.WriteRune(scope.val[len(scope.val)-1])
				}
				state.buf.WriteString(`inf"`)

				state.PopScope()
				state.PushScope(LiteralValue([]rune(`inf`)), OtherType, nil)
				return ErrDontAdvance
			}
		}

		if scope.counter == 0 && firstState != OtherState {
			scope.val = append(scope.val, vals...)
			scope.state = firstState
			scope.counter = counter
		}

		if len(scope.val) == 2 && scope.counter == 2 && r == ':' {
			state.PopScope()
			state.PushScope(Time(2, scope.val, true, false), OtherType, nil)
			return ErrDontAdvance
		}

		if len(scope.val) == 4 && scope.counter == 4 && r == '-' {
			state.PopScope()
			state.PushScope(Date(4, scope.val), OtherType, nil)
			return ErrDontAdvance
		}

		if unicode.IsSpace(r) && scope.lastToken == DIGITT {
			state.PopScope()
			state.buf.WriteString(string(scope.val))
			return ErrDontAdvance
		}

		if (r == ']' || r == '}' || r == ',') && scope.lastToken == DIGITT {
			state.PopScope()
			state.buf.WriteString(string(scope.val))
			return ErrDontAdvance
		}

		if unicode.IsSpace(r) {
			return parseError(state, `invalid float ending`)
		}

		switch scope.state {
		case OtherState:

			ok, err := floatDispatchSign(r, state, scope)
			if ok || err != nil {
				if r == '-' {
					scope.val = append(scope.val, r)
				}
				return err
			}

			if r == '0' && (scope.lastToken == OTHERT || scope.lastToken == SIGNT) {
				scope.val = append(scope.val, '0')
				scope.counter++
				scope.lastToken = DIGITT
				scope.state = AfterInitialZeroState
				return nil
			}

			if r == '.' {
				if scope.lastToken != DIGITT {
					return parseError(state, `invalid '.' in float`)
				}
				scope.lastToken = DOTT
				scope.val = append(scope.val, '.')
				scope.state = AfterDotState
				return nil
			}

			ok, err = floatDispatchUnderscore(r, state, scope)
			if ok || err != nil {
				return err
			}

			ok, err = floatDispatchExp(r, state, scope)
			if ok || err != nil {
				return err
			}

			return floatDispatchDigit(r, state, scope)

		case AfterInitialZeroState:

			if r == '.' {
				scope.lastToken = DOTT
				scope.val = append(scope.val, '.')
				scope.state = AfterDotState
				return nil
			}

			ok, err := floatDispatchExp(r, state, scope)
			if ok || err != nil {
				return err
			}

			return parseError(state, `invalid float character after zero`)

		case AfterDotState:

			ok, err := floatDispatchUnderscore(r, state, scope)
			if ok || err != nil {
				return err
			}

			ok, err = floatDispatchExp(r, state, scope)
			if ok || err != nil {
				return err
			}

			return floatDispatchDigit(r, state, scope)

		case AfterExpState:

			ok, err := floatDispatchSign(r, state, scope)
			if ok || err != nil {
				scope.val = append(scope.val, r)
				return err
			}

			ok, err = floatDispatchUnderscore(r, state, scope)
			if ok || err != nil {
				return err
			}

			return floatDispatchDigit(r, state, scope)
		}

		return nil
	}
}

func timeVerfiyHours(val []rune) error {

	if len(val) != 2 {
		return fmt.Errorf(`hours wrong lenth`)
	}

	v, err := strconv.ParseInt(string(val), 10, 8)
	if err != nil {
		return errors.Wrap(err, `verifyHours strconv.ParseInt failed`)
	}

	if v > 23 {
		return fmt.Errorf(`hours invalid`)
	}
	return nil
}

func timeVerfiy60(val []rune) error {

	if len(val) != 2 {
		return fmt.Errorf(`hours wrong lenth`)
	}

	v, err := strconv.ParseInt(string(val), 10, 8)
	if err != nil {
		return errors.Wrap(err, `verifyHours strconv.ParseInt failed`)
	}

	if v > 59 {
		return fmt.Errorf(`hours invalid`)
	}
	return nil
}

func Time(offset int, val []rune, flush bool, uptoMinutes bool) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.counter == 0 {
			if flush {
				state.buf.WriteRune('"')
				for _, r := range val {
					state.buf.WriteRune(r)
				}
			}

			scope.counter += offset
			scope.val = val
		}

		if scope.counter > 8 {
			if !unicode.IsOneOf(digitRanges, r) {
				state.PopScope()
				if flush {
					state.buf.WriteRune('"')
				}
				return ErrDontAdvance
			}
		}

		if scope.counter == 8 {
			if r != '.' {
				state.PopScope()
				if flush {
					state.buf.WriteRune('"')
				}
				return ErrDontAdvance
			}
		}

		if scope.counter < 2 ||
			(scope.counter > 2 && scope.counter < 5) ||
			scope.counter > 5 && scope.counter < 8 {
			if !unicode.IsOneOf(digitRanges, r) {
				return parseError(state, `invalid digit in time`)
			}

			scope.val = append(scope.val, r)
		}

		if scope.counter == 5 {

			if len(scope.val) != 4 {
				return parseError(state, `invalid time, hour and minutes`)
			}

			err := timeVerfiyHours(scope.val[:2])
			if err != nil {
				return parseError(state, `invalid time, hours invalid`)
			}
			err = timeVerfiy60(scope.val[2:])
			if err != nil {
				return parseError(state, `invalid time, minutes invalid`)
			}
			scope.val = scope.val[0:0]
		}

		if uptoMinutes && scope.counter == 5 {
			state.PopScope()
			if flush {
				state.buf.WriteRune('"')
			}
			return ErrDontAdvance
		}

		state.buf.WriteRune(r)

		if scope.counter == 2 || scope.counter == 5 {
			if r != ':' {
				return parseError(state, `invalid character in time`)
			}
		}

		if scope.counter == 8 {
			if len(scope.val) != 2 {
				return parseError(state, `invalid time, seconds invalid`)
			}

		}

		scope.counter++
		return nil
	}
}

func Date(offset int, val []rune) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.state == AfterTimeState {
			state.buf.WriteRune('"')
			state.PopScope()
			return ErrDontAdvance
		}

		if scope.state == AfterTState {

			if r == 'Z' {
				state.PopScope()
				state.buf.WriteString(`Z"`)
				return nil
			}

			if r == '-' || r == '+' {
				state.buf.WriteRune(r)
				scope.state = AfterTimeState
				state.PushScope(Time(0, nil, false, true), OtherType, scope)
				return nil
			}

			state.buf.WriteRune('"')
			state.PopScope()
			return ErrDontAdvance
		}

		if scope.state == InitState {

			if r == ' ' || r == 'T' {
				state.buf.WriteRune(r)
				scope.state = AfterTState
				state.PushScope(Time(0, nil, false, false), OtherType, scope)
				return nil
			}

			state.buf.WriteRune('"')
			state.PopScope()
			return ErrDontAdvance
		}

		if scope.counter == 0 {
			state.buf.WriteRune('"')
			for _, r := range val {
				state.buf.WriteRune(r)
			}

			scope.counter += offset
			scope.val = val
		}

		if scope.counter == 2 && r == ':' {
			scope.state = AfterTimeState
			state.PushScope(Time(2, scope.val, false, false), OtherType, scope)
			return ErrDontAdvance
		}

		if scope.counter < 4 {
			if !unicode.IsOneOf(digitRanges, r) {
				return parseError(state, `invalid digit in date year`)
			}
			scope.val = append(scope.val, r)
		}

		if (scope.counter > 4 && scope.counter < 7) ||
			(scope.counter > 7 && scope.counter < 10) {
			if !unicode.IsOneOf(digitRanges, r) {
				return parseError(state, `invalid digit in date`)
			}
			scope.val = append(scope.val, r)
		}

		state.buf.WriteRune(r)

		if scope.counter == 4 {
			if r != '-' {
				return parseError(state, `invalid character in year`)
			}

		}

		if scope.counter == 7 {
			if r != '-' {
				return parseError(state, `invalid character in date day`)
			}
		}

		scope.counter++

		if scope.counter == 10 {

			if len(scope.val) != 8 {
				return parseError(state, `invalid date, month or day`)
			}

			month, err := strconv.ParseInt(string(scope.val[4:6]), 10, 8)
			if err != nil {
				return parseError(state, `invalid digit in dates month`)
			}

			if month < 1 || month > 12 {
				return parseError(state, `invalid month in date`)
			}

			day, err := strconv.ParseInt(string(scope.val[6:8]), 10, 8)
			if err != nil {
				return parseError(state, `invalid month in date`)
			}

			if day < 1 || int(day) > daysInMonth[month-1] {
				return parseError(state, `invalid number of days in month`)
			}

			scope.state = InitState
			return nil
		}
		return nil
	}
}

func Zero(r rune, state *State, scope *Scope) error {

	if unicode.IsSpace(r) || r == ',' || r == ']' || r == '}' {
		state.buf.WriteString(`0`)
		state.PopScope()
		return ErrDontAdvance
	}

	if r == 'x' {
		state.PopScope()
		state.buf.WriteString(`"0x`)
		state.PushScope(PrefixNumber(hexRanges), OtherType, nil)
		return nil
	}

	if r == 'o' {
		state.PopScope()
		state.buf.WriteString(`"0o`)
		state.PushScope(PrefixNumber(octalRanges), OtherType, nil)
		return nil
	}

	if r == 'b' {
		state.PopScope()
		state.buf.WriteString(`"0b`)
		state.PushScope(PrefixNumber(binRanges), OtherType, nil)
		return nil
	}

	if r == 'e' || r == 'E' {
		state.PopScope()
		state.PushScope(Float(AfterExpState, []rune{'0', r}, 2), OtherType, nil)
		return nil
	}

	if unicode.IsOneOf(digitRanges, r) {
		state.PopScope()
		state.PushScope(Date(2, []rune{'0', r}), OtherType, nil)
		return nil
	}

	if r == '.' {
		state.PopScope()
		state.PushScope(Float(AfterDotState, []rune("0."), 2), OtherType, nil)
		return nil
	}

	return parseError(state, `invalid character after zero`)
}

func inlineTableDispatchKeyValue(r rune, defs Defs, state *State, scope *Scope) error {

	if unicode.IsOneOf(bareRanges, r) || r == '"' || r == '\'' {
		scope.lastToken = OTHERT
		scope.state = AfterValueState
		state.PushScope(KeyValue(defs.Define, defs.keyFilter.Push), OtherType, scope)
		return ErrDontAdvance
	}
	return parseError(state, `inline table could not dispatch key`)
}

func InlineTable() ParseFunc {

	defs := Defs{
		m:             map[string]Var{},
		arrayKeyStack: &ArrayKeyStack{},
		keyFilter:     &KeyFilter{}}
	return func(r rune, state *State, scope *Scope) error {

		if r == '\n' {
			return parseError(state, `inline table contains \n`)
		}

		if unicode.IsSpace(r) {
			return nil
		}

		if (scope.state == OtherState || scope.state == AfterValueState) &&
			r == '}' {
			state.PopScope()

			if scope.lastToken == COMT {
				return parseError(state, `inline table invalid comma at end`)
			}

			defs.keyFilter.Close(state.buf)
			state.buf.WriteRune('}')
			return nil
		}

		if scope.state == AfterValueState {

			if r == ',' && scope.lastToken != COMT {
				scope.lastToken = COMT
				return nil
			}

			if scope.lastToken != COMT {
				return parseError(state, `inline table comma not found`)
			}
		}

		return inlineTableDispatchKeyValue(r, defs, state, scope)
	}
}

func InlineArray(r rune, state *State, scope *Scope) error {

	if unicode.IsSpace(r) {
		return nil
	}

	if r == ']' {
		state.PopScope()
		state.buf.WriteRune(']')
		return nil
	}

	if scope.state == AfterValueState {

		if r == ',' && scope.lastToken != COMT {
			scope.lastToken = COMT
			return nil
		}

		if scope.lastToken != COMT {
			return parseError(state, `inline table comma not found`)
		}
		state.buf.WriteRune(',')
	}
	scope.lastToken = OTHERT
	scope.state = AfterValueState
	state.PushScope(Value, OtherType, scope)
	return ErrDontAdvance
}

func LiteralValue(value []rune) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.counter < len(value) {
			if value[scope.counter] != r {
				return parseError(state, `invalid boolean`)
			}
		}

		scope.counter++
		if scope.counter >= len(value) {
			state.PopScope()
			return nil
		}
		return nil
	}
}

func Value(r rune, state *State, scope *Scope) error {

	if scope.lastToken == OTHERT && r == '"' {
		scope.lastToken = QT
		scope.scopeType = StringType
		return nil
	}

	if scope.lastToken == QT && r != '"' {
		state.PopScope()
		state.PushScope(QuotedString, StringType, nil)
		state.buf.WriteRune('"')
		return ErrDontAdvance
	}

	if scope.lastToken == QT && r == '"' {
		scope.lastToken = QQT
		return nil
	}

	if scope.lastToken == QQT && r != '"' {
		state.PopScope()
		state.buf.WriteString(`""`)
		return ErrDontAdvance
	}

	if scope.lastToken == QQT && r == '"' {
		state.PopScope()
		state.PushScope(TrippleQuotedString, StringType, nil)
		state.buf.WriteRune('"')
		return nil
	}

	if scope.lastToken == OTHERT && r == '\'' {
		scope.lastToken = SQT
		scope.scopeType = StringType
		return nil
	}

	if scope.lastToken == SQT && r != '\'' {
		state.PopScope()
		state.PushScope(LiteralString, StringType, nil)
		state.buf.WriteRune('"')
		return ErrDontAdvance
	}

	if scope.lastToken == SQT && r == '\'' {
		scope.lastToken = SQQT
		return nil
	}

	if scope.lastToken == SQQT && r != '\'' {
		state.PopScope()
		state.buf.WriteString(`""`)
		return ErrDontAdvance
	}

	if scope.lastToken == SQQT && r == '\'' {
		state.PopScope()
		state.PushScope(MultiLineLiteralString, StringType, nil)
		state.buf.WriteRune('"')
		return nil
	}

	if unicode.IsSpace(r) {
		return nil
	}

	if r == 't' {
		state.PopScope()
		state.PushScope(LiteralValue([]rune(`true`)), OtherType, nil)
		state.buf.WriteString(`true`)
		return ErrDontAdvance
	}

	if r == 'f' {
		state.PopScope()
		state.PushScope(LiteralValue([]rune(`false`)), OtherType, nil)
		state.buf.WriteString(`false`)
		return ErrDontAdvance
	}

	if r == '{' {
		state.PopScope()
		state.PushScope(InlineTable(), OtherType, nil)
		state.buf.WriteRune('{')
		return nil
	}

	if r == '[' {
		state.PopScope()
		state.PushScope(InlineArray, OtherType, nil)
		state.buf.WriteRune('[')
		return nil
	}

	if r == 'i' {
		state.PopScope()
		state.PushScope(LiteralValue([]rune(`inf`)), OtherType, nil)
		state.buf.WriteString(`"inf"`)
		return ErrDontAdvance
	}

	if r == 'n' {
		state.PopScope()
		state.PushScope(LiteralValue([]rune(`nan`)), OtherType, nil)
		state.buf.WriteString(`"nan"`)
		return ErrDontAdvance
	}

	if r == '0' {
		state.PopScope()
		state.PushScope(Zero, OtherType, nil)
		return nil
	}

	if r == '-' || r == '+' || unicode.IsOneOf(digitRanges, r) {
		state.PopScope()
		state.PushScope(Float(OtherState, nil, 0), OtherType, nil)
		return ErrDontAdvance
	}

	return parseError(state, `invalid value`)
}

func Key(r rune, state *State, scope *Scope) error {

	if unicode.IsSpace(r) && r != '\n' {
		if scope.lastToken != DOTT {
			scope.lastToken = SPACET
		}
		return nil
	}

	if scope.state != OtherState && (unicode.IsSpace(r) || r == '=' || r == ']') {

		if scope.lastToken == DOTT {
			return parseError(state, `invalid '.' at the end of the key`)
		}
		state.PopScope()
		return ErrDontAdvance
	}

	if (scope.state == OtherState || scope.lastToken == DOTT) && r == '.' {
		return parseError(state, `invalid '.' in key`)
	}

	st := scope.state
	if scope.state == OtherState {
		scope.state = InitState
	}

	if scope.state == AfterQuoteState {
		if r != '.' {
			return parseError(state, `invalid character in key after quote`)
		}
		scope.lastToken = DOTT
		scope.state = InitState
	}

	if r == '.' {
		scope.lastToken = DOTT
		state.data = append(state.data, '\n')
		return nil
	}

	if r == '"' {
		if scope.lastToken != DOTT && st != OtherState {
			return parseError(state, `invalid character before '"' in key`)
		}
		scope.lastToken = OTHERT
		scope.state = AfterQuoteState
		state.PushScope(QuotedString, KeyType, scope)
		return nil
	}

	if r == '\'' {
		if scope.lastToken != DOTT && st != OtherState {
			return parseError(state, `invalid character before '\'' in key`)
		}
		scope.lastToken = OTHERT
		scope.state = AfterQuoteState
		state.PushScope(LiteralString, KeyType, scope)
		return nil
	}

	if !unicode.IsOneOf(bareRanges, r) {
		return parseError(state, `invalid character in key`)
	}

	if scope.lastToken == SPACET {
		return parseError(state, `invalid space in key`)
	}

	scope.lastToken = OTHERT
	state.data = append(state.data, r)
	return nil
}

func KeyValue(defineFunc DefineFunc, pushFilter KeyFilterPushFunc) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.state == AfterValueState {

			if r == EOF || r == '\n' || r == ',' || r == '}' || r == ']' {

				state.PopScope()
				return ErrDontAdvance
			}

			if !unicode.IsSpace(r) {
				return parseError(state, `invalid character after value`)
			}
			return nil
		}
		if scope.state == AfterKeyState {

			if scope.lastToken == OTHERT && unicode.IsSpace(r) {
				return nil
			}

			if r == '=' {

				scope.key = state.ExtractKeys()
				ok := defineFunc(scope.key, BasicVar)
				if !ok {
					return parseError(state, `attempt to redefine a key`)
				}

				scope.state = AfterValueState

				pushFilter(scope.key, BasicVar, state.buf)

				state.buf.WriteString(`"` + scope.key[len(scope.key)-1] + `":`)

				state.PushScope(Value, OtherType, scope)
				return nil
			}

			return parseError(state, `invalid key`)
		}

		if !unicode.IsSpace(r) {
			scope.state = AfterKeyState
			state.PushScope(Key, KeyType, scope)
			return ErrDontAdvance
		}
		return nil
	}
}

func Table(r rune, state *State, scope *Scope) error {

	if r == '[' || r == EOF {
		state.PopScope()
		return ErrDontAdvance
	}

	if scope.state == AfterValueState {

		if r == '\n' {
			scope.state = AfterFirstValueState
			return nil
		}

		if !unicode.IsSpace(r) {
			return parseError(state, `invalid value in table`)
		}
		return nil
	}

	if scope.state == AfterTableState || scope.state == AfterFirstValueState {

		if !unicode.IsSpace(r) {
			scope.state = AfterValueState
			state.PushScope(
				KeyValue(
					BaseKeyDefs(scope.key, state.defs),
					BaseKeyFilterPushFunc(scope.key, state.defs.keyFilter.Push),
				),
				OtherType, scope)

			return ErrDontAdvance
		}
		return nil
	}

	if r == '\n' {
		if scope.lastToken != CBT {
			return parseError(state, `table end invalid`)
		}
		scope.key = state.ExtractKeys()

		ok := state.defs.Define(scope.key, TableVar)
		if !ok {
			return parseError(state, `table attempt to redefine a key`)
		}
		state.defs.keyFilter.Push(scope.key, TableVar, state.buf)

		scope.state = AfterTableState
		return nil
	}

	if unicode.IsSpace(r) {
		return nil
	}

	if scope.state == AfterKeyState {

		if scope.lastToken != CBT && r == ']' {
			scope.lastToken = CBT
			return nil
		}

		return parseError(state, `invalid character at table end`)
	}

	if unicode.IsOneOf(bareRanges, r) || r == '"' || r == '\'' {
		scope.state = AfterKeyState
		state.PushScope(Key, KeyType, scope)
		return ErrDontAdvance
	}
	return parseError(state, `invalid character at table start`)
}

func Array(r rune, state *State, scope *Scope) error {

	if r == '[' || r == EOF {
		state.PopScope()
		return ErrDontAdvance
	}

	if scope.state == AfterValueState {

		if r == '\n' {
			scope.state = AfterFirstValueState
			return nil
		}

		if !unicode.IsSpace(r) {
			return parseError(state, `invalid value in array`)
		}
		return nil
	}

	if scope.state == AfterArrayState || scope.state == AfterFirstValueState {
		if !unicode.IsSpace(r) {
			scope.state = AfterValueState
			state.PushScope(
				KeyValue(
					BaseKeyDefs(scope.key, state.defs),
					BaseKeyFilterPushFunc(scope.key, state.defs.keyFilter.Push),
				),
				OtherType,
				scope,
			)
			return ErrDontAdvance
		}
		return nil
	}

	if r == '\n' {
		if scope.lastToken != CBBT {
			return parseError(state, `array end invalid`)
		}
		scope.key = state.ExtractKeys()
		ok := state.defs.Define(scope.key, ArrayVar)
		if !ok {
			return parseError(state, `array attempt to redefine a key`)
		}
		state.defs.keyFilter.Push(scope.key, ArrayVar, state.buf)

		scope.state = AfterArrayState
		return nil
	}

	if unicode.IsSpace(r) {
		return nil
	}

	if scope.state == AfterKeyState {

		if scope.lastToken == CBT && r == ']' {
			scope.lastToken = CBBT
			return nil
		}

		if r == ']' {
			scope.lastToken = CBT
			return nil
		}

		return parseError(state, `invalid character at array end`)
	}

	if unicode.IsOneOf(bareRanges, r) || r == '"' || r == '\'' {
		scope.state = AfterKeyState
		state.PushScope(Key, KeyType, scope)
		return ErrDontAdvance
	}
	return parseError(state, `invalid character at array start`)
}

func Top(r rune, state *State, scope *Scope) error {

	if r == EOF {
		state.PopScope()
		return nil
	}

	if scope.state == OtherState && unicode.IsSpace(r) {
		return nil
	}

	if scope.state == OtherState {
		scope.state = InitState
	}

	if scope.state == AfterValueState {

		if r == '[' {
			scope.state = AfterFirstValueState
			return ErrDontAdvance
		}

		if r == '\n' {
			scope.state = AfterFirstValueState
			return nil
		}

		if !unicode.IsSpace(r) {
			return parseError(state, `invalid character after value`)
		}

		return nil
	}

	if unicode.IsSpace(r) {
		return nil
	}

	if scope.lastToken == CBT && r == '[' {
		scope.lastToken = OTHERT
		state.PushScope(Array, OtherType, scope)
		return nil
	}

	if scope.lastToken == CBT {
		scope.lastToken = OTHERT
		state.PushScope(Table, OtherType, scope)
		return ErrDontAdvance
	}

	if r == '[' {
		scope.lastToken = CBT
		return nil
	}

	if unicode.IsOneOf(bareRanges, r) || r == '"' || r == '\'' {
		scope.state = AfterValueState
		state.PushScope(KeyValue(state.defs.Define, state.defs.keyFilter.Push), OtherType, scope)
		return ErrDontAdvance
	}

	return parseError(state, fmt.Sprintf(`invalid character (%v)`, r))
}

func parseError(s *State, msg string) error {
	return fmt.Errorf("position (%v:%v) msg: %v", s.line, s.position, msg)
}

func isOneOf(r rune, runes []rune) bool {

	for _, rn := range runes {
		if r == rn {
			return true
		}
	}
	return false
}
