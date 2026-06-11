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
    struct TokenValidation {
        bool isValidERC20;
        string name;
        string symbol;
        uint8 decimals;
        uint256 totalSupply;
        address wethPair;
        address usdtPair;
    }

    struct Pair {
        // Address of the pair contract
        address pairContract;
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
        // V2FeeToAddress hold balance of the pair _safeBalanceOf(pair.pairContract, v2pairFeeToAddress)
        uint256 feeAddressHoldLiquidityBalance;
        uint256 feeAddressHoldLiquidityRatio;
        bool isRemoveLiquidity;
    }


    struct ProjectState {
        address tokenContract;
        uint256 updatedAt;
        TokenValidation token;
        Pair wethPair;
        Pair usdtPair;
    }

    struct WalletAssetState {
        address wallet;
        WalletBalanceState assetState;
    }

    struct WalletBalanceState {
        uint256 wethBalance;
        uint256 usdtBalance;
        uint256 nativeBalance;
        uint256 usdtValue;
    }

    struct WalletSimulationStateQuery {
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

    address private constant DEAD_ADDRESS = 0x000000000000000000000000000000000000dEaD;
    address private constant ZERO_ADDRESS = address(0);

    address private immutable factoryContract;
    address private immutable wethContract;
    address private immutable usdtContract;
    bytes32 private immutable initCodePairHash;
    address private immutable v2pairFeeToAddress;

    uint256 private constant TOKEN_PROBE_GAS = 30000;
    uint256 private constant STRING_PROBE_GAS = 50000;
    uint256 private constant STRING_RETURN_DATA_MAX_BYTES = 4096;

    constructor(uint256 chainId) {
        if (chainId == 1) {
            factoryContract = 0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f;
            wethContract = 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2;
            usdtContract = 0xdAC17F958D2ee523a2206206994597C13D831ec7;
            v2pairFeeToAddress = 0xf38521f130fcCF29dB1961597bc5d2B60F995f85;
        } else if (chainId == 56) {
            factoryContract = 0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73;
            wethContract = 0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c;
            usdtContract = 0x55d398326f99059fF775485246999027B3197955;
            v2pairFeeToAddress = 0x0ED943Ce24BaEBf257488771759F9BF482C39706;
        } else {
            revert("Invalid chain id");
        }
        initCodePairHash = IPancakeFactoryView(factoryContract).INIT_CODE_PAIR_HASH();
    }

    function ValidateERC20(address[] calldata tokenContracts) external view returns (TokenValidation[] memory results) {
        results = new TokenValidation[](tokenContracts.length);
        for (uint256 i = 0; i < tokenContracts.length;) {
            results[i] = _getToken(tokenContracts[i]);
            unchecked {
                i++;
            }
        }
    }

    function ListProjectStates(address[] calldata tokenContracts)
        external
        view
        returns (ProjectState[] memory states)
    {
        states = new ProjectState[](tokenContracts.length);
        for (uint256 i = 0; i < tokenContracts.length;) {
            states[i] = _getProjectState(tokenContracts[i]);
            unchecked {
                i++;
            }
        }
    }

    function ListWalletAssetStates(address[] calldata wallets)
        external
        view
        returns (WalletAssetState[] memory states)
    {
        states = new WalletAssetState[](wallets.length);
        for (uint256 i = 0; i < wallets.length;) {
            states[i].wallet = wallets[i];
            states[i].assetState = _getWalletBalanceState(wallets[i]);
            unchecked {
                i++;
            }
        }
    }

    function ListWalletSimulationStates(WalletSimulationStateQuery[] calldata queries)
        external
        view
        returns (SimulationState[] memory states)
    {
        states = new SimulationState[](queries.length);
        for (uint256 i = 0; i < queries.length;) {
            states[i] = _getWalletSimulationState(queries[i].tokenContract, queries[i].msgCaller);
            unchecked {
                i++;
            }
        }
    }

    function _getProjectState(address tokenContract) private view returns (ProjectState memory state) {
        state.tokenContract = tokenContract;
        state.updatedAt = block.timestamp;
        state.token = _getToken(tokenContract);

        if (!state.token.isValidERC20 || tokenContract == wethContract) {
            return state;
        }

        state.wethPair = _getPairWithDefaultLockers(tokenContract, wethContract);
        if (tokenContract != usdtContract) {
            state.usdtPair = _getPairWithDefaultLockers(tokenContract, usdtContract);
        }

        if (state.usdtPair.isCreated) {
            state.usdtPair.quoteUsdtValue = state.usdtPair.quoteBalance;
        }
        if (state.wethPair.isCreated) {
            state.wethPair.quoteUsdtValue = _quoteToUsdtValue(state.wethPair.quoteBalance, wethContract);
        }
    }

    function _getWalletSimulationState(address tokenContract, address msgCaller)
        private
        view
        returns (SimulationState memory state)
    {
        TokenValidation memory token = _getToken(tokenContract);
        if (!token.isValidERC20) {
            return state;
        }

        (, state.deadAllowance) = _safeAllowance(tokenContract, DEAD_ADDRESS, msgCaller);
        (, state.zeroAllowance) = _safeAllowance(tokenContract, ZERO_ADDRESS, msgCaller);
        if (token.wethPair.code.length > 0) {
            (, state.wethPairAllowance) = _safeAllowance(tokenContract, token.wethPair, msgCaller);
        }
        if (token.usdtPair.code.length > 0) {
            (, state.usdtPairAllowance) = _safeAllowance(tokenContract, token.usdtPair, msgCaller);
        }
        (, state.callerBalance) = _safeBalanceOf(tokenContract, msgCaller);
    }

    function _getToken(address tokenContract) private view returns (TokenValidation memory token) {
        bool nameOk;
        bool symbolOk;
        bool decimalsOk;
        bool totalSupplyOk;
        bool balanceOfZeroOk;
        bool allowanceZeroOk;

        (nameOk, token.name) = _safeString(tokenContract, IERC20View.name.selector);
        (symbolOk, token.symbol) = _safeString(tokenContract, IERC20View.symbol.selector);
        (decimalsOk, token.decimals) = _safeUint8(tokenContract, IERC20View.decimals.selector);
        (totalSupplyOk, token.totalSupply) = _safeUint256(tokenContract, IERC20View.totalSupply.selector);
        (balanceOfZeroOk,) = _safeBalanceOf(tokenContract, address(0));
        (allowanceZeroOk,) = _safeAllowance(tokenContract, address(0), address(0));

        token.isValidERC20 = totalSupplyOk
            && token.totalSupply > 0
            && balanceOfZeroOk
            && allowanceZeroOk
            && decimalsOk
            && token.decimals > 0
            && nameOk
            && bytes(token.name).length > 0
            && symbolOk
            && bytes(token.symbol).length > 0;

        if (token.isValidERC20) {
            if (tokenContract != wethContract) {
                token.wethPair = _pairFor(tokenContract, wethContract);
            }
            if (tokenContract != usdtContract) {
                token.usdtPair = _pairFor(tokenContract, usdtContract);
            }
        }
    }

    function _getWalletBalanceState(address wallet) private view returns (WalletBalanceState memory state) {
        if (wallet == ZERO_ADDRESS) {
            return state;
        }

        (, state.wethBalance) = _safeBalanceOf(wethContract, wallet);
        (, state.usdtBalance) = _safeBalanceOf(usdtContract, wallet);
        state.nativeBalance = wallet.balance;

        uint256 wethUsdtValue = _quoteToUsdtValue(state.wethBalance, wethContract);
        uint256 nativeUsdtValue = _quoteToUsdtValue(state.nativeBalance, wethContract);
        state.usdtValue = wethUsdtValue + state.usdtBalance + nativeUsdtValue;
    }

    function _getPairWithDefaultLockers(address baseTokenContract, address quoteTokenContract)
        private
        view
        returns (Pair memory pair)
    {
        pair.pairContract = _pairFor(baseTokenContract, quoteTokenContract);
        pair.isCreated = pair.pairContract.code.length > 0;
        if (!pair.isCreated) {
            return pair;
        }

        pair.token0 = _safeAddress(pair.pairContract, IUniswapV2PairView.token0.selector);
        pair.token1 = _safeAddress(pair.pairContract, IUniswapV2PairView.token1.selector);
        (, pair.totalSupply) = _safeUint256(pair.pairContract, IUniswapV2PairView.totalSupply.selector);
        (pair.reserve0, pair.reserve1, pair.blockTimestampLast) = _safeReserves(pair.pairContract);

        (, pair.baseBalance) = _safeBalanceOf(baseTokenContract, pair.pairContract);
        (, pair.quoteBalance) = _safeBalanceOf(quoteTokenContract, pair.pairContract);
        pair.lockedLiquidity = _lockedLiquidityWithDefaultLockers(pair.pairContract);

        (, pair.feeAddressHoldLiquidityBalance) = _safeBalanceOf(pair.pairContract, v2pairFeeToAddress);
        if (pair.totalSupply > 0) {
            pair.isRemoveLiquidity = pair.feeAddressHoldLiquidityBalance * 100 >= pair.totalSupply * 90;
            pair.feeAddressHoldLiquidityRatio = pair.feeAddressHoldLiquidityBalance * 100 / pair.totalSupply;
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

    function _lockedLiquidityWithDefaultLockers(address pairContract)
        private
        view
        returns (uint256 lockedLiquidity)
    {
        (, uint256 zeroBalance) = _safeBalanceOf(pairContract, ZERO_ADDRESS);
        (, uint256 deadBalance) = _safeBalanceOf(pairContract, DEAD_ADDRESS);
        lockedLiquidity = zeroBalance + deadBalance;
    }

    function _pairFor(address tokenA, address tokenB) private view returns (address pair) {
        return _pairForWithInitCodeHash(tokenA, tokenB);
    }

    function _pairForWithInitCodeHash(address tokenA, address tokenB) private view returns (address pair) {
        (address token0, address token1) = _sortTokens(tokenA, tokenB);
        bytes32 salt = keccak256(abi.encodePacked(token0, token1));

        pair = address(uint160(uint256(keccak256(abi.encodePacked(
            hex"ff",
            factoryContract,
            salt,
            initCodePairHash
        )))));
    }

    function _sortTokens(address tokenA, address tokenB) private pure returns (address token0, address token1) {
        require(tokenA != tokenB, "Pancake: IDENTICAL_ADDRESSES");
        (token0, token1) = tokenA < tokenB ? (tokenA, tokenB) : (tokenB, tokenA);
        require(token0 != address(0), "Pancake: ZERO_ADDRESS");
    }

    function _safeString(address target, bytes4 selector) private view returns (bool ok, string memory value) {
        (bool callOk, bytes memory data) = target.staticcall{gas: STRING_PROBE_GAS}(abi.encodeWithSelector(selector));
        if (!callOk || data.length < 64 || data.length > STRING_RETURN_DATA_MAX_BYTES) {
            return (false, "");
        }
        uint256 offset;
        uint256 length;
        assembly {
            offset := mload(add(data, 32))
            length := mload(add(data, 64))
        }
        uint256 paddedLength = (length + 31) & ~uint256(31);
        if (offset != 32 || length > STRING_RETURN_DATA_MAX_BYTES || data.length < 64 + paddedLength) {
            return (false, "");
        }
        return (true, abi.decode(data, (string)));
    }

    function _safeUint8(address target, bytes4 selector) private view returns (bool ok, uint8 value) {
        (bool callOk, bytes memory data) = target.staticcall{gas: TOKEN_PROBE_GAS}(abi.encodeWithSelector(selector));
        if (!callOk || data.length < 32) {
            return (false, 0);
        }
        return (true, abi.decode(data, (uint8)));
    }

    function _safeUint256(address target, bytes4 selector) private view returns (bool ok, uint256 value) {
        (bool callOk, bytes memory data) = target.staticcall{gas: TOKEN_PROBE_GAS}(abi.encodeWithSelector(selector));
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
        (bool callOk, bytes memory data) = token.staticcall{gas: TOKEN_PROBE_GAS}(abi.encodeWithSelector(IERC20View.balanceOf.selector, account));
        if (!callOk || data.length < 32) {
            return (false, 0);
        }
        return (true, abi.decode(data, (uint256)));
    }

    function _safeAllowance(address token, address owner, address spender) private view returns (bool ok, uint256 value) {
        (bool callOk, bytes memory data) = token.staticcall{gas: TOKEN_PROBE_GAS}(abi.encodeWithSelector(IERC20View.allowance.selector, owner, spender));
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
