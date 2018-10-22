package proio_test

import (
	"fmt"

	"github.com/proio-org/go-proio"
	model "github.com/proio-org/go-proio-pb/model/example"
)

func Example_print() {
	event := proio.NewEvent()

	parent := &model.Particle{Pdg: 443}
	parentID := event.AddEntry("Particle", parent)
	event.TagEntry(parentID, "Truth", "Primary")

	child1 := &model.Particle{Pdg: 11}
	child2 := &model.Particle{Pdg: -11}
	childIDs := event.AddEntries("Particle", child1, child2)
	for _, id := range childIDs {
		event.TagEntry(id, "Truth", "GenStable")
	}

	parent.Child = append(parent.Child, childIDs...)
	child1.Parent = append(child1.Parent, parentID)
	child2.Parent = append(child2.Parent, parentID)

	fmt.Print(event)

	// Output:
	// ---------- TAG: GenStable ----------
	// ID: 2
	// Entry type: proio.model.example.Particle
	// parent: 1
	// pdg: 11
	//
	// ID: 3
	// Entry type: proio.model.example.Particle
	// parent: 1
	// pdg: -11
	//
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// child: 2
	// child: 3
	// pdg: 443
	//
	// ID: 2
	// Entry type: proio.model.example.Particle
	// parent: 1
	// pdg: 11
	//
	// ID: 3
	// Entry type: proio.model.example.Particle
	// parent: 1
	// pdg: -11
	//
	// ---------- TAG: Primary ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// child: 2
	// child: 3
	// pdg: 443
	//
	// ---------- TAG: Truth ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// child: 2
	// child: 3
	// pdg: 443
	//
	// ID: 2
	// Entry type: proio.model.example.Particle
	// parent: 1
	// pdg: 11
	//
	// ID: 3
	// Entry type: proio.model.example.Particle
	// parent: 1
	// pdg: -11
}
