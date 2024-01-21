package circuit

import (
	"errors"

	"github.com/consensys/gnark/frontend"
)

type TCircuit struct {
	X []frontend.Variable `gnark:",public"`
	S frontend.Variable   `gnark:"S,secret"`
	Y frontend.Variable   `gnark:",public"`
}

// Define declares the circuit logic. The compiler then produces a list of constraints
// which must be satisfied (valid witness) in order to create a valid zk-SNARK
func (circuit *TCircuit) Define(api frontend.API) error {
	if len(circuit.X) > 0 {
		res := circuit.X[0]
		for i := 1; i < len(circuit.X); i++ {
			res = api.Mul(res, circuit.X[i])
		}

		res = api.Add(res, circuit.S)

		api.AssertIsEqual(circuit.Y, res)
		return nil
	} else {
		return errors.New("invalid length")
	}
}
