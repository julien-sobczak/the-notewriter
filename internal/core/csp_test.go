package core

import (
	"context"
	"fmt"
	"testing"

	"github.com/gnboorse/centipede"
	"github.com/stretchr/testify/assert"
)

// Supporting file inclusions require to process notes in a given order to process included notes first.
// Basically, we must control in which order files are traversed (for links between files) and in
// which order notes inside a given file are traversed (for links between notes).
//
// A naive approach is to sort files by appearance (natural order). Then:
// - Iterate over files, for every file with already satisfied dependency, add the file.
// - Continue until all files are added.
// - If no files are added during an iteration = conflicts...
// Repeat the same logic for notes.

/*
ParsedFile.Dependencies() []string // medialinks 👍
ParsedNote.Dependencies() []string // medialinks 👍 (NB: external links must have been addressed by ParsedFile.Dependencies())
File.Dependencies() []string // 👎 too restrictive? required to parse the raw content every time (as not stored in DB)
Note.Dependencies() []string
// Or prefer:
File.Relations() // include, reference, inspire <= Query the DB
Note.Relations() // include, reference, inspire <= Query the DB
*/

// Assertions:
// - Do not manage note inclusions from ParsedNote
//   (because require checking the database when the astraction ParsedNote was created
//    to test easily the parsing logic without any external interaction)

// Questions:
// - Manage files AND/OR notes inclusion ([[file]] or only [[file#note]]???) <= The conceptual building block must be notes, not files. Why a note would include a file?
// - How to address included note update as content is denormalized at note creation time?
//   => Check the new table relations and append concerned notes when running nt add.
//   => Add Refresh on StatefulObject?
// - What about medias update => Trigger a refresh a every concerned note to refresh their metadata.

func TestCentipede(t *testing.T) {
	t.Skip()

	// initialize variables
	vars := make(centipede.Variables[int], 0)
	constraints := make(centipede.Constraints[int], 0)
	varNames := make(centipede.VariableNames, 0)

	oids := [9]string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	domain := centipede.IntRange(0, len(oids)*100) // we want to have an order

	Before := func(var1 centipede.VariableName, var2 centipede.VariableName) centipede.Constraint[int] {
		return centipede.Constraint[int]{Vars: centipede.VariableNames{var1, var2}, ConstraintFunction: func(variables *centipede.Variables[int]) bool {
			if variables.Find(var1).Empty || variables.Find(var2).Empty {
				return true
			}
			v1 := variables.Find(var1).Value
			v2 := variables.Find(var2).Value
			return v1 < v2
		}}
	}

	for _, oid := range oids {
		varName := centipede.VariableName(oid)
		vars = append(vars, centipede.NewVariable(varName, domain))
		varNames = append(varNames, varName)
	}

	// F must be before B
	constraints = append(constraints, Before(varNames[5], varNames[1]))
	constraints = append(constraints, Before(varNames[5], varNames[2]))
	constraints = append(constraints, Before(varNames[1], varNames[2]))
	for i := 0; i < len(oids); i++ {
		for j := 0; i < len(oids); i++ {
			if i == j {
				continue
			}
			constraints = append(constraints, centipede.NotEquals[int](varNames[i], varNames[j]))
		}
	}

	// All order index must be unique
	// constraints = append(constraints, centipede.AllUnique[int](varNames...)...)

	// create solver
	solver := centipede.NewBackTrackingCSPSolver(vars, constraints)

	// simplify variable domains following initial assignment
	solver.State.MakeArcConsistent(context.TODO())
	success, err := solver.Solve(context.TODO()) // run the solution
	assert.Nil(t, err)

	assert.True(t, success)

	for _, oid := range oids {
		varName := centipede.VariableName(oid)
		variable := solver.State.Vars.Find(varName)
		fmt.Println(varName, "=", variable.Value)
	}

	t.Fail()
}
