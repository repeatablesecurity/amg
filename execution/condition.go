package execution

import (
	"fmt"
	"strconv"
	"strings"
)

// Condition represents a conditional expression used in 'IF'.
type Condition struct {
	actualExpression   string
	concreteExpression string
	varResolutionList  []string
	valueMap           map[string]*ValueWrapper
	lhs                string
	rhs                string
	operator           string
}

// NewCondition creates a new *Condition
func NewCondition(expr string) *Condition {
	// TODO: Compile the varResolutionMap
	c := &Condition{
		expr, "", make([]string, 0, 2), make(map[string]*ValueWrapper), "", "", "",
	}
	c.parse()
	return c
}

func (c *Condition) parse() {
	// Currently only few forms are supported
	// Plan is to use grammar and add more sophisticated evaluation
	expr := c.actualExpression
	binaryOperators := []string{"==", ">", ">=", "<", "<=", "!="}
	for _, op := range binaryOperators {
		if splitStr := strings.Split(expr, op); len(splitStr) == 2 {
			c.operator = op
			if c.lhs = strings.Trim(splitStr[0], " "); isResolutionNeeded(c.lhs) {
				c.varResolutionList = append(c.varResolutionList, c.lhs)
			}
			if c.rhs = strings.Trim(splitStr[1], " "); isResolutionNeeded(c.rhs) {
				c.varResolutionList = append(c.varResolutionList, c.rhs)
			}
			break
		}
	}
}

func isResolutionNeeded(token string) bool {
	if strings.HasPrefix(token, "@") || strings.HasPrefix(token, "$") {
		return true
	}
	return false
}

func evaluateBinaryExpression(lhsVal string, rhsVal string, operator string) (bool, bool) {
	lhsNum, err1 := strconv.Atoi(lhsVal)
	rhsNum, err2 := strconv.Atoi(rhsVal)
	var bothAreNumbers bool = (err1 == nil) && (err2 == nil)

	if bothAreNumbers {
		return evaluateBinaryExprNumbers(lhsNum, rhsNum, operator), true
	}

	switch operator {
	case "==":
		return strings.Compare(lhsVal, rhsVal) == 0, true
	case "!=":
		return strings.Compare(lhsVal, rhsVal) != 0, true
	default:
		return false, false
	}
}

func evaluateBinaryExprNumbers(lhs int, rhs int, op string) bool {
	switch op {
	case "==":
		return lhs == rhs
	case "!=":
		return lhs != rhs
	case ">":
		return lhs > rhs
	case ">=":
		return lhs >= rhs
	case "<":
		return lhs < rhs
	case "<=":
		return lhs <= rhs
	}
	return false
}

// UnknownVarsList returns the list of variables whose values are not known
func (c *Condition) UnknownVarsList() []string {
	return c.varResolutionList
}

// SetVarValue sets value for an unresolved var
func (c *Condition) SetVarValue(varName string, val *ValueWrapper) {
	c.valueMap[varName] = val
}

// Evaluate the condition and return {result, possible}
func (c *Condition) Evaluate() (bool, bool) {
	var lhsVal string = c.lhs
	if isResolutionNeeded(c.lhs) {
		lhsV, present := c.valueMap[c.lhs]
		if !present {
			return false, false
		}
		lhsVal = lhsV.AsString()
	}
	var rhsVal string = c.rhs
	if isResolutionNeeded(c.rhs) {
		rhsV, present := c.valueMap[c.rhs]
		if !present {
			return false, false
		}
		rhsVal = rhsV.AsString()
	}

	fmt.Printf("Evaluating expression: %s %s %s\n", lhsVal, c.operator, rhsVal)
	return evaluateBinaryExpression(lhsVal, rhsVal, c.operator)
}
