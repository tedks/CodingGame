package debug

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
	if vars == nil {
		return nil
	}
	cloned := make([]*Variable, len(vars))
	for i, v := range vars {
		cloned[i] = cloneVariable(v)
	}
	return cloned
}

func cloneVariable(v *Variable) *Variable {
	if v == nil {
		return nil
	}
	cloned := *v
	cloned.Children = cloneVariables(v.Children)
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
