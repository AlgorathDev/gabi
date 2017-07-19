// Copyright 2016 Maarten Everts. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gabi

import (
	"crypto/rand"
	"math/big"
)

// CredentialIssuer holds the key material for a credential issuer.
type CredentialIssuer struct {
	Sk      *PrivateKey
	Pk      *PublicKey
	Context *big.Int
}

// NewCredentialIssuer creates a new credential issuer.
func NewCredentialIssuer(sk *PrivateKey, pk *PublicKey, context *big.Int) *CredentialIssuer {
	return &CredentialIssuer{Sk: sk, Pk: pk, Context: context}
}

// IssueSignature produces an IssueSignatureMessage for the attributes based on
// the IssueCommitmentMessage provided. Note that this function DOES NOT check
// the proofs containted in the IssueCommitmentMessage! That needs to be done at
// a higher level!
func (i *CredentialIssuer) IssueSignature(msg *IssueCommitmentMessage, attributes []*big.Int, nonce1 *big.Int) (*IssueSignatureMessage, error) {

	signature, err := i.signCommitmentAndAttributes(msg.U, attributes)
	if err != nil {
		return nil, err
	}
	proof := i.proveSignature(signature, msg.Nonce2)

	return &IssueSignatureMessage{Signature: signature, Proof: proof}, nil
}

// signCommitmentAndAttributes produces a (partial) signature on the commitment
// and the attributes. The signature by itself does not verify because the
// commitment contains a blinding factor that needs to be taken into account
// when verifying the signature.
func (i *CredentialIssuer) signCommitmentAndAttributes(U *big.Int, attributes []*big.Int) (*CLSignature, error) {
	// Skip the first generator
	return signMessageBlockAndCommitment(i.Sk, i.Pk, U, attributes, i.Pk.R[1:])
}

// randomElementMultiplicativeGroup returns a random element in the
// multiplicative group Z_{modulus}^*.
func randomElementMultiplicativeGroup(modulus *big.Int) *big.Int {
	r := big.NewInt(0)
	t := new(big.Int)
	for r.Sign() <= 0 || t.GCD(nil, nil, r, modulus).Cmp(bigONE) != 0 {
		// TODO: for memory/cpu efficiency re-use r's memory. See Go's
		// implementation for finding a random prime.
		r, _ = rand.Int(rand.Reader, modulus)
	}
	return r
}

// proveSignature returns a proof of knowledge of $e^{-1}$ in the signature.
func (i *CredentialIssuer) proveSignature(signature *CLSignature, nonce2 *big.Int) *ProofS {
	Q := new(big.Int).Exp(signature.A, signature.E, i.Pk.N)
	groupModulus := new(big.Int).Mul(i.Sk.PPrime, i.Sk.QPrime)
	d := new(big.Int).ModInverse(signature.E, groupModulus)

	eCommit := randomElementMultiplicativeGroup(groupModulus)
	ACommit := new(big.Int).Exp(Q, eCommit, i.Pk.N)

	c := hashCommit([]*big.Int{i.Context, Q, signature.A, nonce2, ACommit})
	eResponse := new(big.Int).Mul(c, d)
	eResponse.Sub(eCommit, eResponse).Mod(eResponse, groupModulus)

	return &ProofS{c, eResponse}
}
