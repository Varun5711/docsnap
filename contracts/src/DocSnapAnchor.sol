// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

contract DocSnapAnchor {
    enum EvidenceStatus {
        Pending,
        Certified,
        Rejected
    }

    struct EvidenceRecord {
        bytes32 evidenceCommitment;
        bytes32 screenshotHash;
        bytes32 scrapedTextHash;
        bytes32 metadataCommitment;
        bytes32 claimsRoot;
        bytes32 teeCertificateHash;
        address submitter;
        uint64 submittedAt;
        uint64 certifiedAt;
        EvidenceStatus status;
    }

    address public owner;
    address public teeReporter;

    mapping(bytes32 => EvidenceRecord) public records;

    event EvidenceSubmitted(
        bytes32 indexed evidenceId,
        address indexed submitter,
        bytes32 evidenceCommitment,
        bytes32 claimsRoot
    );

    event TeeInstructionRequested(
        bytes32 indexed evidenceId,
        bytes32 evidenceCommitment,
        bytes32 metadataCommitment,
        bytes32 claimsRoot
    );

    event EvidenceCertified(
        bytes32 indexed evidenceId,
        bytes32 teeCertificateHash,
        EvidenceStatus status
    );

    error NotOwner();
    error NotTEEReporter();
    error EvidenceExists();
    error EvidenceMissing();

    constructor(address initialTEEReporter) {
        owner = msg.sender;
        teeReporter = initialTEEReporter;
    }

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier onlyTEEReporter() {
        if (msg.sender != teeReporter) revert NotTEEReporter();
        _;
    }

    function setTEEReporter(address nextTEEReporter) external onlyOwner {
        teeReporter = nextTEEReporter;
    }

    function transferOwnership(address nextOwner) external onlyOwner {
        owner = nextOwner;
    }

    function submitEvidence(
        bytes32 evidenceId,
        bytes32 evidenceCommitment,
        bytes32 screenshotHash,
        bytes32 scrapedTextHash,
        bytes32 metadataCommitment,
        bytes32 claimsRoot
    ) external {
        if (records[evidenceId].submitter != address(0)) revert EvidenceExists();

        records[evidenceId] = EvidenceRecord({
            evidenceCommitment: evidenceCommitment,
            screenshotHash: screenshotHash,
            scrapedTextHash: scrapedTextHash,
            metadataCommitment: metadataCommitment,
            claimsRoot: claimsRoot,
            teeCertificateHash: bytes32(0),
            submitter: msg.sender,
            submittedAt: uint64(block.timestamp),
            certifiedAt: 0,
            status: EvidenceStatus.Pending
        });

        emit EvidenceSubmitted(evidenceId, msg.sender, evidenceCommitment, claimsRoot);
        emit TeeInstructionRequested(evidenceId, evidenceCommitment, metadataCommitment, claimsRoot);
    }

    function recordTEECertificate(
        bytes32 evidenceId,
        bytes32 teeCertificateHash,
        bool accepted
    ) external onlyTEEReporter {
        EvidenceRecord storage record = records[evidenceId];
        if (record.submitter == address(0)) revert EvidenceMissing();

        record.teeCertificateHash = teeCertificateHash;
        record.certifiedAt = uint64(block.timestamp);
        record.status = accepted ? EvidenceStatus.Certified : EvidenceStatus.Rejected;

        emit EvidenceCertified(evidenceId, teeCertificateHash, record.status);
    }

    function verifyEvidence(bytes32 evidenceId, bytes32 evidenceCommitment) external view returns (bool) {
        EvidenceRecord storage record = records[evidenceId];
        return record.submitter != address(0)
            && record.status == EvidenceStatus.Certified
            && record.evidenceCommitment == evidenceCommitment;
    }
}

