package estimator

import (
	"context"
	"testing"
)

func TestProvider(t *testing.T) {
	p := NewProvider()

	// Initial state
	_, err := p.Current(context.Background())
	if err != ErrNotReady {
		t.Errorf("Current() error = %v, want ErrNotReady", err)
	}
	if p.Ready() {
		t.Error("Ready() = true, want false")
	}

	// Update
	est := &GasEstimate{BlockNumber: 1}
	p.Update(est)

	// Check state
	got, err := p.Current(context.Background())
	if err != nil {
		t.Errorf("Current() error = %v", err)
	}
	if got != est {
		t.Error("Current() returned different pointer")
	}
	if !p.Ready() {
		t.Error("Ready() = false, want true")
	}

	// Update again
	est2 := &GasEstimate{BlockNumber: 2}
	p.Update(est2)

	got, err = p.Current(context.Background())
	if err != nil {
		t.Errorf("Current() error = %v", err)
	}
	if got != est2 {
		t.Error("Current() returned different pointer")
	}
}
