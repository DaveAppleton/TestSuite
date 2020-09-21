# first cut test environment

## Background

This was developed to evaluate the results of a programming test in a solidity programming class.

The specifications were as follows

Having had a class that showed the structure of an ERC721, they were asked to develop a single contract with the following functions

### AddCarToRegistry

``` solidity
    function addCarToRegistry(
        string memory _manufacturer,
        string memory _model,
        uint   _year_made,
        string memory _chassis_number,
        address _newOwner
        ) public returns (uint256) ;
```

### Ownership / DataAccess Functions

``` solidity
    function getOwner(uint256 index) public view returns(address);
    function howManyCarsDoTheyOwn(address them) public view returns (uint256);
    function getCarByOwnerAndIndex(address them, uint256 index) public view returns (uint256);

    function transfer(uint256 index, address newOwner) public returns (bool)

    function getManufacturer(uint256 index) public view returns (string memory);
    function getModel(uint256 index) public view returns (string memory)
    function getYear(uint256 index) public view returns (uint256);


```

Events were not part of the assignment.

## Operation

Solidity source files are uploaded to a web server.

The received file is compiled and deployed on a GETH simulated backend.

The test script is executed, parameter encoding is based on the data types extracted from the ABI obtained from the compilation.

If the script makes it to the end, the test has passed.

## Notes

The code has been working - however since it was developed, I changed my dev machine to use solc-select which means that some of the compilation process needs to be re-worked (it also plays hell with my use of ABIGEN)

## Plans

Immediate work would be

1. Handling of different compiler versions
2. Monitoring of events
3. Building a decent test script system instead of current hack

