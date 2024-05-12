pragma solidity >=0.7.0 <0.9.0;

contract Billboard {

    address payable private owner;
    uint256 private current_cost;
    string private message;

    constructor() {
        owner = payable(msg.sender);
        current_cost = 0;
    }

    function set(string calldata new_message) external payable {
        require(msg.value > current_cost);
        current_cost = msg.value;
        message = new_message;
    }

    function currentPrice() external view returns (uint256) {
        return current_cost;
    }

    function display() external view returns (string memory) {
        return message;
    }

    function withdraw() external {
        if (msg.sender == owner) {
            owner.transfer(address(this).balance);
        }
    }

}
