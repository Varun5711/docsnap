// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import {Script} from "forge-std/Script.sol";
import {DocSnapAnchor} from "../src/DocSnapAnchor.sol";

contract DeployDocSnapAnchor is Script {
    function run() external returns (DocSnapAnchor anchor) {
        address teeReporter = vm.envAddress("DOCSNAP_TEE_REPORTER");
        vm.startBroadcast();
        anchor = new DocSnapAnchor(teeReporter);
        vm.stopBroadcast();
    }
}

