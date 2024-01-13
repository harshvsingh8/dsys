pragma solidity >=0.8.2 <0.9.0;

contract StableCoin {

    mapping (address => uint256) private balances;

    constructor() {

    }

    function deposit() external payable {
        balances[msg.sender] += msg.value;
    }

    function getBalance() external view returns (uint256) {
        return balances[msg.sender];
    }

    function getBalanceOf(address other) external view returns (uint256) {
        return balances[other];
    }

    function transfer(address recipient, uint256 amount) external {
        require(amount <= balances[msg.sender]);
        balances[recipient] += amount;
        balances[msg.sender] -= amount;
    }

    function withdraw() external {
        require(balances[msg.sender] > 0);
        (bool success, ) = msg.sender.call{value: balances[msg.sender]}("");
        require(success, "Transfer failed.");
        balances[msg.sender] = 0;
    }
}
