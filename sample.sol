pragma solidity ^0.6.0;

contract Vijeshmycar {

    // either of these addresses can add new cars to the registry
    address public mainOwner = msg.sender;
    address public alternateOwner = 0x20a540aE4629483dc4B9a1B81D9722B7cdD960A2;
    
    struct car {
        string  manufacturer;
        string  model;
        uint    year_made;
        string  chassis_number;
        address newOwner;
    }
    
    uint256 nextcar;
    mapping(uint256=>car) car_registry;
    mapping(address=>uint256[]) collections;
    mapping(uint256=>address) owners;
    mapping(uint256=>uint256) ownedPosition;
    uint carcount;
    
    function addCarToRegistry(
        string memory _manufacturer,
        string memory _model,
        uint   _year_made,
        string memory _chassis_number,
        address _newOwner
        ) public returns (uint256) {
            
            // Insert your code here
            
            car_registry[nextcar] = car( {
                manufacturer  :_manufacturer,
                model         :_model,
                year_made     : _year_made,
                chassis_number:_chassis_number,
                newOwner      :_newOwner
            });
            nextcar++;
            
            owners[nextcar-1] = getOwner(nextcar-1);    
            collections[_newOwner].push(nextcar-1);
            ownedPosition[nextcar-1] = collections[_newOwner].length;
            return nextcar-1;

        
        
    }
    
    function getManufacturer(uint256 index) public view returns (string memory) {
        return car_registry[index].manufacturer;
    }
    function getModel(uint256 index) public view returns (string memory) {
        
        return car_registry[index].model;
    }
    function getChassis(uint256 index) public view returns (string memory) {
        
        return car_registry[index].chassis_number;
    }

    function getYear(uint256 index) public view returns (uint256) {
        
        return car_registry[index].year_made;
    }
    
    function getOwner(uint256 index) public view returns(address) {
        
        return car_registry[index].newOwner;
            }
    
   function howManyCarsDoTheyOwn(address them) public view returns (uint256) {
       
        
        return collections[them].length;
    }
    
    function getCarByOwnerAndIndex(address them, uint256 index) public view returns (uint256) {
        
        return collections[them][index];
    }
    
    function transfer(uint256 index, address newOwner) public returns (bool) {

        require(getOwner(index)== mainOwner , "This car not owned by  You so cannot transfer");
        car_registry[index].newOwner = newOwner;
        owners[index]= newOwner;
        uint pos = ownedPosition[index];
        uint len = collections[mainOwner].length-1;
        collections[mainOwner][pos] = collections[mainOwner][len];
        collections[mainOwner].pop();
        collections[newOwner].push(index);
        ownedPosition[index] = collections[newOwner].length-1;
        return true;
        
    }

}