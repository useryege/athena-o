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
        address contractAddress;
        address token0;
        address token1;
        uint256 totalSupply;
        uint112 reserve0;
        uint112 reserve1;
        uint32 blockTimestampLast;
        uint256 tokenReserveBalance;
        uint256 wethReserveBalance;
        uint256 lockedLiquidity;
        bool isCreated;
    }

    struct Project {
        address tokenContract;
        uint256 updatedAt;
        Token token;
        Pair pair;
    }

    address public immutable factoryContract;
    address public immutable wethContract;
    bytes32 public immutable initCodePairHash;

    constructor(address factory, address weth) {
        factoryContract = factory;
        wethContract = weth;
        initCodePairHash = IPancakeFactoryView(factory).INIT_CODE_PAIR_HASH();
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

        address pairContract = _pairFor(tokenContract, wethContract);
        project.pair.contractAddress = pairContract;

        project.pair.isCreated = pairContract.code.length > 0;
        if (!project.pair.isCreated) {
            return project;
        }

        project.pair.token0 = _safeAddress(pairContract, IUniswapV2PairView.token0.selector);
        project.pair.token1 = _safeAddress(pairContract, IUniswapV2PairView.token1.selector);
        (, project.pair.totalSupply) = _safeUint256(pairContract, IUniswapV2PairView.totalSupply.selector);

        (
            project.pair.reserve0,
            project.pair.reserve1,
            project.pair.blockTimestampLast
        ) = _safeReserves(pairContract);

        (, project.pair.tokenReserveBalance) = _safeBalanceOf(tokenContract, pairContract);

        (, project.pair.wethReserveBalance) = _safeBalanceOf(wethContract, pairContract);

        project.pair.lockedLiquidity = _lockedLiquidity(pairContract, lockers);

        return project;
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