package debug

// MaxCloneDepth limits recursion depth when cloning nested variables.
const MaxCloneDepth = 100

func cloneFrames(frames []*Frame) []*Frame {
	if frames == nil {
		return nil
	}
	cloned := make([]*Frame, len(frames))
	for i, frame := range frames {
		cloned[i] = cloneFrame(frame)
	}
	return cloned
}

func cloneFrame(frame *Frame) *Frame {
	if frame == nil {
		return nil
	}
	cloned := *frame
	cloned.Arguments = cloneVariables(frame.Arguments)
	cloned.Locals = cloneVariables(frame.Locals)
	cloned.Returns = cloneVariables(frame.Returns)
	return &cloned
}

func cloneVariables(vars []*Variable) []*Variable {
	return cloneVariablesWithDepth(vars, 0)
}

func cloneVariablesWithDepth(vars []*Variable, depth int) []*Variable {
	if vars == nil {
		return nil
	}
	cloned := make([]*Variable, len(vars))
	for i, v := range vars {
		cloned[i] = cloneVariableWithDepth(v, depth)
	}
	return cloned
}

func cloneVariable(v *Variable) *Variable {
	return cloneVariableWithDepth(v, 0)
}

func cloneVariableWithDepth(v *Variable, depth int) *Variable {
	if v == nil {
		return nil
	}
	if depth >= MaxCloneDepth {
		cloned := *v
		cloned.Children = nil
		cloned.HasMore = true
		return &cloned
	}
	cloned := *v
	cloned.Children = cloneVariablesWithDepth(v.Children, depth+1)
	return &cloned
}

func cloneDataFlows(flows []*DataFlow) []*DataFlow {
	if flows == nil {
		return nil
	}
	cloned := make([]*DataFlow, len(flows))
	for i, flow := range flows {
		cloned[i] = cloneDataFlow(flow)
	}
	return cloned
}

func cloneDataFlow(flow *DataFlow) *DataFlow {
	if flow == nil {
		return nil
	}
	cloned := *flow
	cloned.Variables = cloneVariables(flow.Variables)
	return &cloned
}

func cloneBreakpoints(bps map[string]*Breakpoint) []*Breakpoint {
	if bps == nil {
		return nil
	}
	cloned := make([]*Breakpoint, 0, len(bps))
	for _, bp := range bps {
		cloned = append(cloned, cloneBreakpoint(bp))
	}
	return cloned
}

func cloneBreakpoint(bp *Breakpoint) *Breakpoint {
	if bp == nil {
		return nil
	}
	cloned := *bp
	return &cloned
}
