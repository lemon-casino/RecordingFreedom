package audio

import "errors"

type MonoResampler struct {
	sourceRate int
	targetRate int
	pending    []float32
	position   float64
}

func NewMonoResampler(sourceRate int, targetRate int) (*MonoResampler, error) {
	if sourceRate <= 0 {
		return nil, errors.New("source sample rate is required")
	}
	if targetRate <= 0 {
		return nil, errors.New("target sample rate is required")
	}
	return &MonoResampler{sourceRate: sourceRate, targetRate: targetRate}, nil
}

func (r *MonoResampler) Convert(samples []float32) []float32 {
	if len(samples) == 0 {
		return nil
	}
	if r.sourceRate == r.targetRate {
		output := make([]float32, len(samples))
		copy(output, samples)
		return output
	}

	r.pending = append(r.pending, samples...)
	step := float64(r.sourceRate) / float64(r.targetRate)
	output := make([]float32, 0, int(float64(len(samples))*float64(r.targetRate)/float64(r.sourceRate))+2)
	for r.position+1 < float64(len(r.pending)) {
		index := int(r.position)
		fraction := float32(r.position - float64(index))
		current := r.pending[index]
		next := r.pending[index+1]
		output = append(output, current+(next-current)*fraction)
		r.position += step
	}

	consumed := int(r.position)
	if consumed > 0 {
		copy(r.pending, r.pending[consumed:])
		r.pending = r.pending[:len(r.pending)-consumed]
		r.position -= float64(consumed)
	}
	return output
}

func (r *MonoResampler) Reset() {
	r.pending = nil
	r.position = 0
}
