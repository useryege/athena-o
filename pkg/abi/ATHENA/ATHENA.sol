// SPDX-License-Identifier: GPL-3.0

pragma solidity >=0.8.2 <0.9.0;

interface IERC20View {
    function name() external view returns (string memory);
    function symbol() external view returns (string memory);
    function decimals() external view returns (uint8);
    function totalSupply() external view returns (uint256);
    function balanceOf(address account) external view returns (uint256);
    function allowance(address owner, address spender) external view returns (uint256);
}

interface IUniswapV2PairView {
    function token0() external view returns (address);
    function token1() external view returns (address);
    function totalSupply() external view returns (uint256);
    function getReserves() external view returns (uint112 reserve0, uint112 reserve1, uint32 blockTimestampLast);
}

interface IPancakeFactoryView {
    function INIT_CODE_PAIR_HASH() external view returns (bytes32);
}

/**
 * @title Athena
 * @author yege
 */
contract Athena {
    struct Token {
        bool isValidERC20;
        string name;
        string symbol;
        uint8 decimals;
        uint256 totalSupply;
    }

    struct Pair {
        // Address of the pair contract
        address contractAddress;
        // Address of the first token
        address token0;
        // Address of the second token
        address token1;
        // Total supply of the pair
        uint256 totalSupply;
        // Locked liquidity amount
        uint256 lockedLiquidity;
        // BalanceOf(tokenContract, pair) baseTokenContract can only be tokenContract
        uint256 baseBalance;
        // BalanceOf(quoteTokenContract, pair)  quoteTokenContract can only be weth or usdt
        uint256 quoteBalance;
        // QuoteBalance transfer to usdt value
        uint256 quoteUsdtValue;
        // Whether the pair is created
        bool isCreated;
        // Reserves of the pair
        uint112 reserve0;
        uint112 reserve1;
        uint32 blockTimestampLast;
        // V2FeeToAddress hold balance of the pair _safeBalanceOf(pair.contractAddress, v2pairFeeToAddress)
        uint256 feeAddressHoldLiquidityBalance;
        bool isRemoveLiquidity;
    }



    struct Project {
        address tokenContract;
        uint256 updatedAt;
        Token token;
        Pair wethPair;
        Pair usdtPair;
    }

    struct ProjectQuery {
        address tokenContract;
        address msgCaller;
    }

    struct SimulationState {
        uint256 deadAllowance;
        uint256 zeroAllowance;
        uint256 wethPairAllowance;
        uint256 usdtPairAllowance;
        uint256 callerBalance;
    }

    struct ProjectWithSimulationState {
        Project project;
        SimulationState simulationState;
    }

    address public immutable DEAD_ADDRESS = 0x000000000000000000000000000000000000dEaD;
    address public immutable ZERO_ADDRESS = address(0);


    address public immutable factoryContract;
    address public immutable wethContract;
    uint8 public immutable wethDecimals;
    address public immutable usdtContract;
    uint8 public immutable usdtDecimals;
    bytes32 public immutable initCodePairHash;
    address public immutable v2pairFeeToAddress;

    constructor(uint256 chainId) {
        if (chainId == 1) {
            factoryContract = 0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f;
            wethContract = 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2;
            wethDecimals = 18;
            usdtContract = 0xdAC17F958D2ee523a2206206994597C13D831ec7;
            usdtDecimals = 6;
            v2pairFeeToAddress = 0xf38521f130fcCF29dB1961597bc5d2B60F995f85;
        } else if (chainId == 56) {
            factoryContract = 0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73;
            wethContract = 0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c;
            wethDecimals = 18;
            usdtContract = 0x55d398326f99059fF775485246999027B3197955;
            usdtDecimals = 18;
            v2pairFeeToAddress = 0x0ED943Ce24BaEBf257488771759F9BF482C39706;
        } else {
            revert("Invalid chain id");
        }
        initCodePairHash = IPancakeFactoryView(factoryContract).INIT_CODE_PAIR_HASH();
    }

    function PairFor(address tokenA, address tokenB) external view returns (address pair) {
        return _pairFor(tokenA, tokenB);
    }

    function PairForWithInitCodeHash(address tokenA, address tokenB) public view returns (address pair) {
        (address token0, address token1) = _sortTokens(tokenA, tokenB);
        bytes32 salt = keccak256(abi.encodePacked(token0, token1));

        pair = address(uint160(uint256(keccak256(abi.encodePacked(
            hex"ff",
            factoryContract,
            salt,
            initCodePairHash
        )))));
    }

    function Get(address tokenContract, address[] calldata lockers) external view returns (Project memory) {
        return _get(tokenContract, lockers);
    }

    function List(address[] calldata tokenContracts, address[] calldata lockers) external view returns (Project[] memory projects) {
        projects = new Project[](tokenContracts.length);
        for (uint256 i = 0; i < tokenContracts.length;) {
            projects[i] = _get(tokenContracts[i], lockers);
            unchecked {
                i++;
            }
        }
    }

    function GetWithSimulationState(ProjectQuery calldata query, address[] calldata lockers)
        external
        view
        returns (ProjectWithSimulationState memory)
    {
        return _getWithSimulationState(query, lockers);
    }

    function ListWithSimulationState(ProjectQuery[] calldata queries, address[] calldata lockers)
        external
        view
        returns (ProjectWithSimulationState[] memory projects)
    {
        projects = new ProjectWithSimulationState[](queries.length);
        for (uint256 i = 0; i < queries.length;) {
            projects[i] = _getWithSimulationState(queries[i], lockers);
            unchecked {
                i++;
            }
        }
    }

    function _getWithSimulationState(ProjectQuery calldata query, address[] calldata lockers)
        private
        view
        returns (ProjectWithSimulationState memory result)
    {
        result.project = _get(query.tokenContract, lockers);
        if (!result.project.token.isValidERC20) {
            return result;
        }

        (, result.simulationState.deadAllowance) = _safeAllowance(query.tokenContract, DEAD_ADDRESS, query.msgCaller);
        (, result.simulationState.zeroAllowance) = _safeAllowance(query.tokenContract, address(0), query.msgCaller);
        if (result.project.wethPair.isCreated) {
            (, result.simulationState.wethPairAllowance) =
                _safeAllowance(query.tokenContract, result.project.wethPair.contractAddress, query.msgCaller);
        }
        if (result.project.usdtPair.isCreated) {
            (, result.simulationState.usdtPairAllowance) =
                _safeAllowance(query.tokenContract, result.project.usdtPair.contractAddress, query.msgCaller);
        }
        (, result.simulationState.callerBalance) = _safeBalanceOf(query.tokenContract, query.msgCaller);
    }

    function _get(address tokenContract, address[] calldata lockers) private view returns (Project memory) {
        Project memory project;

        project.tokenContract = tokenContract;
        project.updatedAt = block.timestamp;

        bool nameOk;
        bool symbolOk;
        bool decimalsOk;
        bool totalSupplyOk;
        bool balanceOfZeroOk;
        bool allowanceZeroOk;

        (nameOk, project.token.name) = _safeString(tokenContract, IERC20View.name.selector);
        (symbolOk, project.token.symbol) = _safeString(tokenContract, IERC20View.symbol.selector);
        (decimalsOk, project.token.decimals) = _safeUint8(tokenContract, IERC20View.decimals.selector);
        (totalSupplyOk, project.token.totalSupply) = _safeUint256(tokenContract, IERC20View.totalSupply.selector);
        (balanceOfZeroOk,) = _safeBalanceOf(tokenContract, address(0));
        (allowanceZeroOk,) = _safeAllowance(tokenContract, address(0), address(0));

        project.token.isValidERC20 = totalSupplyOk
            && project.token.totalSupply > 0
            && balanceOfZeroOk
            && allowanceZeroOk
            && decimalsOk
            && project.token.decimals > 0
            && nameOk
            && bytes(project.token.name).length > 0
            && symbolOk
            && bytes(project.token.symbol).length > 0;

        if (!project.token.isValidERC20 || tokenContract == wethContract) {
            return project;
        }

        project.wethPair = _getPair(tokenContract, wethContract, lockers);
        project.usdtPair = _getPair(tokenContract, usdtContract, lockers);

        if (project.usdtPair.isCreated) {
            project.usdtPair.quoteUsdtValue = project.usdtPair.quoteBalance;
        }
        if (project.wethPair.isCreated) {
            project.wethPair.quoteUsdtValue = _quoteToUsdtValue(project.wethPair.quoteBalance, wethContract);
        }

        return project;
    }

    function _getPair(address baseTokenContract, address quoteTokenContract, address[] calldata lockers)
        private
        view
        returns (Pair memory pair)
    {
        pair.contractAddress = _pairFor(baseTokenContract, quoteTokenContract);
        pair.isCreated = pair.contractAddress.code.length > 0;
        if (!pair.isCreated) {
            return pair;
        }

        pair.token0 = _safeAddress(pair.contractAddress, IUniswapV2PairView.token0.selector);
        pair.token1 = _safeAddress(pair.contractAddress, IUniswapV2PairView.token1.selector);
        (, pair.totalSupply) = _safeUint256(pair.contractAddress, IUniswapV2PairView.totalSupply.selector);
        (pair.reserve0, pair.reserve1, pair.blockTimestampLast) = _safeReserves(pair.contractAddress);

        (, pair.baseBalance) = _safeBalanceOf(baseTokenContract, pair.contractAddress);
        (, pair.quoteBalance) = _safeBalanceOf(quoteTokenContract, pair.contractAddress);
        pair.lockedLiquidity = _lockedLiquidity(pair.contractAddress, lockers);

        (, pair.feeAddressHoldLiquidityBalance) = _safeBalanceOf(pair.contractAddress, v2pairFeeToAddress);
        if (pair.totalSupply > 0) {
            pair.isRemoveLiquidity = pair.feeAddressHoldLiquidityBalance * 100 >= pair.totalSupply * 90;
        }
    }

    function _quoteToUsdtValue(uint256 quoteAmount, address quoteTokenContract) private view returns (uint256) {
        if (quoteAmount == 0 || quoteTokenContract == usdtContract) {
            return quoteAmount;
        }

        address quoteUsdtPair = _pairFor(quoteTokenContract, usdtContract);
        if (quoteUsdtPair.code.length == 0) {
            return 0;
        }

        address pairToken0 = _safeAddress(quoteUsdtPair, IUniswapV2PairView.token0.selector);
        (uint256 reserve0, uint256 reserve1,) = _safeReserves(quoteUsdtPair);
        if (reserve0 == 0 || reserve1 == 0) {
            return 0;
        }

        if (pairToken0 == quoteTokenContract) {
            return quoteAmount * reserve1 / reserve0;
        }
        return quoteAmount * reserve0 / reserve1;
    }

    function _lockedLiquidity(address pairContract, address[] calldata lockers) private view returns (uint256 lockedLiquidity) {
        for (uint256 i = 0; i < lockers.length; i++) {
            (, uint256 balance) = _safeBalanceOf(pairContract, lockers[i]);
            lockedLiquidity += balance;
        }
    }

    function _pairFor(address tokenA, address tokenB) private view returns (address pair) {
        return PairForWithInitCodeHash(tokenA, tokenB);
    }

    function _sortTokens(address tokenA, address tokenB) private pure returns (address token0, address token1) {
        require(tokenA != tokenB, "Pancake: IDENTICAL_ADDRESSES");
        (token0, token1) = tokenA < tokenB ? (tokenA, tokenB) : (tokenB, tokenA);
        require(token0 != address(0), "Pancake: ZERO_ADDRESS");
    }

    function _safeString(address target, bytes4 selector) private view returns (bool ok, string memory value) {
        (bool callOk, bytes memory data) = target.staticcall(abi.encodeWithSelector(selector));
        if (!callOk || data.length < 64) {
            return (false, "");
        }
        return (true, abi.decode(data, (string)));
    }

    function _safeUint8(address target, bytes4 selector) private view returns (bool ok, uint8 value) {
        (bool callOk, bytes memory data) = target.staticcall(abi.encodeWithSelector(selector));
        if (!callOk || data.length < 32) {
            return (false, 0);
        }
        return (true, abi.decode(data, (uint8)));
    }

    function _safeUint256(address target, bytes4 selector) private view returns (bool ok, uint256 value) {
        (bool callOk, bytes memory data) = target.staticcall(abi.encodeWithSelector(selector));
        if (!callOk || data.length < 32) {
            return (false, 0);
        }
        return (true, abi.decode(data, (uint256)));
    }

    function _safeAddress(address target, bytes4 selector) private view returns (address) {
        (bool ok, bytes memory data) = target.staticcall(abi.encodeWithSelector(selector));
        if (!ok || data.length < 32) {
            return address(0);
        }
        return abi.decode(data, (address));
    }

    function _safeBalanceOf(address token, address account) private view returns (bool ok, uint256 value) {
        (bool callOk, bytes memory data) = token.staticcall(abi.encodeWithSelector(IERC20View.balanceOf.selector, account));
        if (!callOk || data.length < 32) {
            return (false, 0);
        }
        return (true, abi.decode(data, (uint256)));
    }

    function _safeAllowance(address token, address owner, address spender) private view returns (bool ok, uint256 value) {
        (bool callOk, bytes memory data) = token.staticcall(abi.encodeWithSelector(IERC20View.allowance.selector, owner, spender));
        if (!callOk || data.length < 32) {
            return (false, 0);
        }
        return (true, abi.decode(data, (uint256)));
    }

    function _safeReserves(address pair) private view returns (uint112 reserve0, uint112 reserve1, uint32 blockTimestampLast) {
        (bool ok, bytes memory data) = pair.staticcall(abi.encodeWithSelector(IUniswapV2PairView.getReserves.selector));
        if (!ok || data.length < 96) {
            return (0, 0, 0);
        }
        return abi.decode(data, (uint112, uint112, uint32));
    }
}
