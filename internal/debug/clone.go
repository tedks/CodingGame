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
