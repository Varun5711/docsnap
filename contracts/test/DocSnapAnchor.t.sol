// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import {Test} from "forge-std/Test.sol";
import {DocSnapAnchor} from "../src/DocSnapAnchor.sol";

contract DocSnapAnchorTest is Test {
    DocSnapAnchor anchor;
    address teeReporter = address(0xBEEF);

    function setUp() public {
        anchor = new DocSnapAnchor(teeReporter);
    }

    function testSubmitCertifyAndVerify() public {
        bytes32 evidenceId = keccak256("evidence-1");
        bytes32 commitment = keccak256("commitment");

        anchor.submitEvidence(
            evidenceId,
            commitment,
            keccak256("screenshot"),
            keccak256("text"),
            keccak256("metadata"),
            keccak256("claims")
        );

        vm.prank(teeReporter);
        anchor.recordTEECertificate(evidenceId, keccak256("cert"), true);

        assertTrue(anchor.verifyEvidence(evidenceId, commitment));
        assertFalse(anchor.verifyEvidence(evidenceId, keccak256("tampered")));
    }
}

