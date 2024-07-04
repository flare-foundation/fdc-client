package attestation

// exports of private functions for tests

var AndBitwise = andBitwise

var BitVoteForSet = bitVoteForSet

var EarlierLog = earlierLog

var AttestationFromDatabaseLog = attestationFromDatabaseLog

var ValidateResponse = (*Attestation).validateResponse

var PrepareRequest = (*Attestation).prepareRequest

var SkipDuplicates = (checkList).skipDuplicates

var AddAttestation = (*Round).addAttestation

type CheckList = checkList
