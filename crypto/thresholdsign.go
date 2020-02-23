// +build relic

package crypto

// #cgo CFLAGS: -g -Wall -std=c99 -I./ -I./relic/include -I./relic/include/low
// #cgo LDFLAGS: -Lrelic/build/lib -l relic_s
// #include "thresholdsign_include.h"
import "C"

import (
	"fmt"
)

// ThresholdSigner holds the data needed for threshold signaures
type ThresholdSigner struct {
	// size of the group
	size int
	// the thresold t of the scheme where (t+1) shares are
	// required to reconstruct a signature
	threshold int
	// the current node private key (a DKG output)
	currentPrivateKey PrivateKey
	// the group public key (a DKG output)
	groupPublicKey PublicKey
	// the group public key shares (a DKG output)
	publicKeyShares []PublicKey
	// the hasher to be used for all signatures
	hashAlgo Hasher
	// the message to be signed. Siganture shares and the threshold signature
	// are verified using this message
	messageToSign []byte
	// the valid signature shares received from other nodes
	shares []byte // simulates an array of Signatures
	// (or a matrix of by bytes) to accommodate a cgo constraint
	// the list of signers corresponding to the list of shares
	signers []index
	// the threshold signature. It is equal to nil if less than (t+1) shares are
	// received
	thresholdSignature Signature
}

const ThresholdSignaureTag = "Threshold Signatures"

// NewThresholdSigner creates a new instance of Threshold signer using BLS
// hash is the hashing algorithm to be used
// size is the number of participants
func NewThresholdSigner(size int, hashingAlgo Hasher) (*ThresholdSigner, error) {
	if size < ThresholdMinSize || size > ThresholdMaxSize {
		return nil, cryptoError{fmt.Sprintf("size should be between %d and %d", ThresholdMinSize, ThresholdMaxSize)}
	}

	// optimal threshold (t) to allow the largest number of malicious nodes (m)
	threshold := optimalThreshold(size)
	// Hahser to be used
	hasher := hashingAlgo
	shares := make([]byte, 0, (threshold+1)*SignatureLenBLS_BLS12381)
	signers := make([]index, 0, threshold+1)

	return &ThresholdSigner{
		size:               size,
		threshold:          threshold,
		hashAlgo:           hasher,
		shares:             shares,
		signers:            signers,
		thresholdSignature: nil,
	}, nil
}

// SetKeys sets the private and public keys needed by the threshold signature
// the input keys can be the output keys of a Distributed Key Generator
func (s *ThresholdSigner) SetKeys(currentPrivateKey PrivateKey,
	groupPublicKey PublicKey,
	sharePublicKeys []PublicKey) {

	s.currentPrivateKey = currentPrivateKey
	s.groupPublicKey = groupPublicKey
	s.publicKeyShares = sharePublicKeys
}

// SetMessageToSign sets the next message to be signed
// all received signatures of a different message are ignored
func (s *ThresholdSigner) SetMessageToSign(message []byte) {
	s.ClearShares()
	s.messageToSign = message
}

// SignShare generates a signature share using the current private key share
func (s *ThresholdSigner) SignShare() (Signature, error) {
	if s.currentPrivateKey == nil {
		return nil, cryptoError{"The private key of the current node is not set"}
	}
	// TOD0: should ReceiveThresholdSignatureMsg be called ?
	return s.currentPrivateKey.Sign(s.messageToSign, s.hashAlgo)
}

// VerifyShare verifies a signature share using the signer's public key
func (s *ThresholdSigner) verifyShare(share Signature, signerIndex index) (bool, error) {
	if s.size-1 < int(signerIndex) {
		return false, cryptoError{"The signer index is larger than the group size"}
	}
	if len(s.publicKeyShares)-1 < int(signerIndex) {
		return false, cryptoError{"The node public keys are not set"}
	}

	return s.publicKeyShares[signerIndex].Verify(share, s.messageToSign, s.hashAlgo)
}

// VerifyThresholdSignature verifies a threshold signature using the group public key
func (s *ThresholdSigner) VerifyThresholdSignature(thresholdSignature Signature) (bool, error) {
	if s.groupPublicKey == nil {
		return false, cryptoError{"The group public key is not set"}
	}
	return s.groupPublicKey.Verify(thresholdSignature, s.messageToSign, s.hashAlgo)
}

// ClearShares clears the shares and signers lists
func (s *ThresholdSigner) ClearShares() {
	s.thresholdSignature = nil
	s.signers = s.signers[:0]
	s.shares = s.shares[:0]
}

// ReceiveSignatureShare processes a new TS share
// If the share is valid, not perviously added and the threshold not reached yet,
// it is appended to a local list of valid signatures
func (s *ThresholdSigner) ReceiveSignatureShare(orig int, share Signature) (bool, error) {
	verif, err := s.verifyShare(share, index(orig))
	if err != nil {
		return false, err
	}
	// check if share is valid and threshold is not reached
	if verif && len(s.signers) < (s.threshold+1) {
		// check if the share is new
		isSeen := false
		for _, e := range s.signers {
			if e == index(orig) {
				isSeen = true
				break
			}
		}
		if !isSeen {
			// append the share
			s.shares = append(s.shares, share...)
			s.signers = append(s.signers, index(orig))
		}
	}
	return verif, nil
}

// ThresholdSignaure returns:
// - bool: true if the threshold was reached, false otherwise
// - Signature: the threshold signature if the threshold was reached, nil otherwise
func (s *ThresholdSigner) ThresholdSignaure() (bool, Signature, error) {
	// thresholdSignature is only computed once
	if s.thresholdSignature != nil {
		return true, s.thresholdSignature, nil
	}
	// reconstruct the threshold signature
	if len(s.signers) == (s.threshold + 1) {
		thresholdSignature, err := s.reconstructThresholdSignature()
		if err != nil {
			return false, nil, err
		}
		s.thresholdSignature = thresholdSignature
		return true, thresholdSignature, nil
	}
	return false, nil, nil
}

// ReconstructThresholdSignature reconstructs the threshold signature from at least (t+1) shares.
func (s *ThresholdSigner) reconstructThresholdSignature() (Signature, error) {
	// sanity check
	if len(s.shares) != len(s.signers)*signatureLengthBLS_BLS12381 {
		s.ClearShares()
		return nil, cryptoError{"The number of signature shares is not matching the number of signers"}
	}
	thresholdSignature := make([]byte, signatureLengthBLS_BLS12381)
	// Lagrange Interpolate at point 0
	C.G1_lagrangeInterpolateAtZero(
		(*C.uchar)(&thresholdSignature[0]),
		(*C.uchar)(&s.shares[0]),
		(*C.uint8_t)(&s.signers[0]), (C.int)(len(s.signers)),
	)

	// Verify the computed signature
	verif, err := s.VerifyThresholdSignature(thresholdSignature)
	if err != nil {
		return nil, err
	}
	if !verif {
		return nil, cryptoError{
			"The constructed threshold signature in incorrect. There might be an issue with the set keys"}
	}

	return thresholdSignature, nil
}

// ReconstructThresholdSignature is a stateless function that takes a list of
// signatures and their signers's indices and returns the threshold signature
// size is the size of the threshold signature group
// The function does not check the validity of the shares, and does not check
// the validity of the resulting signature.
// The function assumes the threshold value is equal to floor((n-1)/2)
// ReconstructThresholdSignature returns:
// - Signature: the threshold signature if the threshold was reached, nil otherwise
func ReconstructThresholdSignature(size int, shares []Signature, signers []int) (Signature, error) {
	if len(shares) != len(signers) {
		return nil, cryptoError{"The number of signature shares is not matching the number of signers"}
	}
	// check if the threshold was not reached
	threshold := optimalThreshold(size)
	if len(shares) < threshold+1 {
		return nil, nil
	}

	// flatten the shares (required by the C layer)
	flatShares := make([]byte, 0, signatureLengthBLS_BLS12381*(threshold+1))
	indexSigners := make([]index, 0, threshold+1)
	for i, share := range shares {
		flatShares = append(flatShares, share...)
		indexSigners = append(indexSigners, index(signers[i]))
	}

	thresholdSignature := make([]byte, signatureLengthBLS_BLS12381)
	// Lagrange Interpolate at point 0
	C.G1_lagrangeInterpolateAtZero(
		(*C.uchar)(&thresholdSignature[0]),
		(*C.uchar)(&flatShares[0]),
		(*C.uint8_t)(&indexSigners[0]), (C.int)(threshold+1),
	)
	return thresholdSignature, nil
}

// EnoughShares is a stateless function that takes the size of the threshold
// signature group and a shares number and returns true if the shares number
// is enough to reconstruct a threshold signature
// The function assumes the threshold value is equal to floor((n-1)/2)
func EnoughShares(size int, sharesNumber int) bool {
	return sharesNumber >= (optimalThreshold(size) + 1)
}
