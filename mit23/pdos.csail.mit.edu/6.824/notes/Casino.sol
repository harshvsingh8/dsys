pragma solidity >=0.7.0 <0.9.0;

contract Casino {

    address payable private owner;
    mapping (address => uint256) private balances;

    constructor() {
        owner = payable(msg.sender);
    }

    function deposit() external payable {
        balances[msg.sender] += msg.value;
    }

    function withdraw(uint256 amount) external {
        require(amount <= balances[msg.sender]);
        balances[msg.sender] -= amount;
        payable(msg.sender).transfer(amount);
    }

    function getBalance() external view returns (uint256) {
        return balances[msg.sender];
    }

    function gamble(uint256 wager) external returns (bool) {
        // ensure that player has enough to gamble
        require(balances[msg.sender] >= wager);
        // ensure that house has enough to pay out
        require(balances[owner] >= wager);
        // ~ 5% house edge
        if (uint256(blockhash(block.number-1)) % 100 >= 55) {
            balances[msg.sender] += wager;
            balances[owner] -= wager;
            return true;
        } else {
            balances[msg.sender] -= wager;
            balances[owner] += wager;
            return false;
        }
    }

}
