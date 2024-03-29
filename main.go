package main

import (
	"fmt"
	"log"
	"os"

	"gnark/cook-gnark/circuit"

	"github.com/consensys/gnark/backend/plonk"
	plonk_bn254 "github.com/consensys/gnark/backend/plonk/bn254"
	"github.com/consensys/gnark/frontend"

	cs "github.com/consensys/gnark/constraint/bn254"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend/cs/scs"

	"github.com/consensys/gnark/test/unsafekzg"
)

func main() {
	var x_vec []frontend.Variable
	x_len := 10
	// x_vec = make([]frontend.Variable, x_len)
	res := 1
	for i := 0; i < x_len; i++ {
		// x_vec[i] = frontend.Variable(i + 1)
		x_vec = append(x_vec, frontend.Variable(i+1))
		res *= (i + 1)
	}

	s := 5

	res += s

	// for public inputs
	t := &circuit.TCircuit{
		X: make([]frontend.Variable, x_len), //SHOULD ALLOCATE HERE
	}

	// for secret inputs
	// t := &circuit.TCircuit{
	// 	X: x_vec,
	// 	S: frontend.Variable(s),
	// 	Y: frontend.Variable(res),
	// }

	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), scs.NewBuilder, t)
	if err != nil {
		fmt.Println("circuit compilation error")
	}

	scs := ccs.(*cs.SparseR1CS)
	// NB! Unsafe, use MPC!!!
	srs, srsLagrange, err := unsafekzg.NewSRS(scs, unsafekzg.WithFSCache())
	if err != nil {
		panic(err)
	}

	// Correct data: the proof passes
	{
		// Witnesses instantiation. Witness is known only by the prover,
		// while public w is a public data known by the verifier.

		w := circuit.TCircuit{
			X: x_vec,
			S: frontend.Variable(s),
			Y: frontend.Variable(res),
		}

		witnessFull, err := frontend.NewWitness(&w, ecc.BN254.ScalarField())
		if err != nil {
			// error happens here
			log.Fatal(err)
		}

		witnessPublic, err := frontend.NewWitness(&w, ecc.BN254.ScalarField(), frontend.PublicOnly())
		if err != nil {
			log.Fatal(err)
		}
		fWitness, _ := os.Create("./gnark-verifier/data/add_public.wit")
		// witnessPublic.WriteTo(fWitness)
		schema, _ := frontend.NewSchema(&w)
		wpis_json, _ := witnessPublic.ToJSON(schema)
		fWitness.Write(wpis_json)
		fWitness.Close()

		// public data consists of the polynomials describing the constants involved
		// in the constraints, the polynomial describing the permutation ("grand
		// product argument"), and the FFT domains.
		pk, vk, err := plonk.Setup(ccs, srs, srsLagrange)
		//_, err := plonk.Setup(r1cs, kate, &publicWitness)
		if err != nil {
			log.Fatal(err)
		}

		proof, err := plonk.Prove(ccs, pk, witnessFull)
		if err != nil {
			log.Fatal(err)
		}

		err = plonk.Verify(proof, vk, witnessPublic)
		if err != nil {
			log.Fatal(err)
		}

		fSolidity, _ := os.Create("./gnark-verifier/contracts/PlonkVerifier.sol")
		_ = vk.ExportSolidity(fSolidity)

		fProof, _ := os.Create("./gnark-verifier/data/add_proof.proof")
		_proof := proof.(*plonk_bn254.Proof)
		proof_marshal := _proof.MarshalSolidity()
		fProof.Write(proof_marshal)
		fProof.Close()
	}
}
