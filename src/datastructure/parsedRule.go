// TODO redo this ugly makeshift (maybe putting together the rule antlr4 parser with the one from grule) and add support for other modifiers

package datastructure

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/hyperjumptech/grule-rule-engine/ast"
	"github.com/hyperjumptech/grule-rule-engine/builder"
	"github.com/hyperjumptech/grule-rule-engine/pkg"
)

type ParsedRule struct {
	Name           string
	Events         []string
	DefaultActions []ParsedAction
	Task           ParsedTask
}

type ParsedAction struct {
	Resource   string
	Expression *ast.Assignment
}

type ParsedTask struct {
	Mode      string
	Condition *ast.Expression
	Actions   []ParsedAction
}

func (t *ParsedTask) AcceptExpression(exp *ast.Expression) error {
	if t.Condition != nil {
		return errors.New("task condition already assigned")
	}
	t.Condition = exp
	return nil
}

func (a ParsedAction) String() string {
	return a.Expression.GetGrlText()
}

func ActionsToStr(actions []ParsedAction) string {
	res := ""
	for _, action := range actions {
		res += action.String() + "; "
	}
	return res
}

func NewParsedRule(rule Rule, kl *ast.KnowledgeLibrary, types map[string]string) *ParsedRule {
	res := &ParsedRule{
		Name:           rule.Name,
		Events:         make([]string, len(rule.Events)),
		DefaultActions: NewParsedActionList(rule.DefaultActions, rule.Name+"default", kl, types, false),
		Task:           NewParsedTask(rule.Task, rule.Name+"task", kl, types),
	}
	copy(res.Events, rule.Events)
	return res
}

func NewParsedTask(t Task, name string, kl *ast.KnowledgeLibrary, types map[string]string) ParsedTask {
	external := t.Mode == "for all"
	return ParsedTask{
		Mode:      t.Mode,
		Condition: NewParsedExpression(t.Condition, name+"cnd", kl),
		Actions:   NewParsedActionList(t.Actions, name+"actions", kl, types, external),
	}
}

func NewParsedActionList(acts []Action, name string, kl *ast.KnowledgeLibrary, types map[string]string, external bool) []ParsedAction {
	var res []ParsedAction
	for i, a := range acts {
		res = append(res, NewParsedAction(a, name+strconv.Itoa(i), kl, types, external))
	}
	return res
}

func NewParsedAction(a Action, name string, kl *ast.KnowledgeLibrary, types map[string]string, external bool) ParsedAction {
	rb := builder.NewRuleBuilder(kl)
	resource := ""
	if external {
		resource = fmt.Sprintf(`ext.Void["%s"]`, a.Resource)
	} else {
		resource = fmt.Sprintf(`this.%s["%s"]`, types[a.Resource], a.Resource)
	}
	rule := fmt.Sprintf("rule %s { when true then %s = %s; }", name, resource, a.Expression)
	bs := pkg.NewBytesResource([]byte(rule))
	err := rb.BuildRuleFromResource("dummy", "0.0.0", bs)
	if err != nil {
		panic(err)
	}
	kb := kl.NewKnowledgeBaseInstance("dummy", "0.0.0")
	ruleEntry := kb.RuleEntries[name]
	return ParsedAction{
		Resource:   a.Resource,
		Expression: ruleEntry.ThenScope.ThenExpressionList.ThenExpressions[0].Assignment,
	}
}

func NewParsedExpression(str, name string, kl *ast.KnowledgeLibrary) *ast.Expression {
	rb := builder.NewRuleBuilder(kl)
	rule := "rule " + name + " { when " + str + " then Ok(); }"
	bs := pkg.NewBytesResource([]byte(rule))
	err := rb.BuildRuleFromResource("dummy", "0.0.0", bs)
	if err != nil {
		panic(err)
	}
	kb := kl.NewKnowledgeBaseInstance("dummy", "0.0.0")
	ruleEntry := kb.RuleEntries[name]
	return ruleEntry.WhenScope.Expression
}
