import {Alert, Button, Card, Empty, Space, Tag} from 'antd';
import {Link} from 'react-router-dom';
import {KeyValueGrid, Section, StatusTag} from '../../components';
import {formatBeijingDateTime, formatBeijingUnixSeconds, formatBlockNumber} from '../../shared/format';
import {TokenProject, TokenProjectProfile, TokenProjectProfilePair, TokenProjectProfileWalletSimulation} from '../../shared/services/token-service';
import {ProjectJSONDrawerValue} from './project-json-drawer';
import {pairRiskSignalCopy, RiskSignalTag} from './token-shared';

const present = (value: unknown) => value !== undefined && value !== null && value !== '';
const exact = (value?: string | number) => present(value) ? String(value) : '-';
const currency = (value?: string) => present(value) ? `$${value}` : '-';
const percentage = (value?: string | number) => present(value) ? `${value}%` : '-';
const simulationSuccessCount = (simulation?: TokenProjectProfileWalletSimulation) => simulation
    ? Object.values(simulation).filter(value => value === true).length
    : undefined;

const PairCard = (props: {title: string; pair?: TokenProjectProfilePair}) => {
    const pair = props.pair;
    const chainState = pair?.chainState;
    const market = pair?.market;
    return (
        <Card
            size='small'
            title={props.title}
            extra={chainState
                ? <StatusTag value={chainState.isCreated ? 'Created' : 'Not created'} positive={chainState.isCreated} />
                : pair ? <StatusTag value='Chain state unavailable' /> : undefined}
        >
            {pair ? <>
                <KeyValueGrid columns={2} items={[
                    {label: 'Kind', value: exact(pair.kind)},
                    {label: 'Address', value: exact(pair.address)}
                ]} />
                {chainState ? <KeyValueGrid columns={2} items={[
                    {label: 'Base balance', value: exact(chainState.baseBalance)},
                    {label: 'Quote balance', value: exact(chainState.quoteBalance)},
                    {label: 'Quote value in USDT', value: exact(chainState.quoteUsdtValueInt || chainState.quoteUsdtValue)},
                    {label: 'Reserves updated', value: formatBeijingUnixSeconds(chainState.reserveUpdatedAt) || '-'},
                    {label: 'LP total supply', value: exact(chainState.liquidity?.totalSupply)},
                    {label: 'Locked liquidity', value: exact(chainState.liquidity?.lockedLiquidity)},
                    {label: 'LP balance held by fixed fee address', value: exact(chainState.liquidity?.fixedFeeAddressBalance)},
                    {label: 'Fixed fee address share of LP supply', value: percentage(chainState.liquidity?.fixedFeeAddressShare)},
                    {label: pairRiskSignalCopy.pairTokenBalanceExceedsTotalSupply.label, value: <RiskSignalTag value={chainState.signals?.pairTokenBalanceExceedsTotalSupply} applicable={chainState.isCreated !== false} signalLabel={pairRiskSignalCopy.pairTokenBalanceExceedsTotalSupply.label} />},
                    {label: pairRiskSignalCopy.lpMinimumSupplyOnly.label, value: <RiskSignalTag value={chainState.signals?.lpMinimumSupplyOnly} applicable={chainState.isCreated !== false} signalLabel={pairRiskSignalCopy.lpMinimumSupplyOnly.label} />},
                    {label: pairRiskSignalCopy.fixedFeeAddressLpShareGte90Percent.label, value: <RiskSignalTag value={chainState.signals?.fixedFeeAddressLpShareGte90Percent} applicable={chainState.isCreated !== false} signalLabel={pairRiskSignalCopy.fixedFeeAddressLpShareGte90Percent.label} />}
                ]} /> : <Alert type='warning' showIcon={true} title='Chain-state evidence unavailable' description='On-chain balances, liquidity, and risk signals are unknown for this pair.' />}
                {market && <KeyValueGrid columns={2} items={[
                    {label: 'Ave AMM', value: exact(market.amm)},
                    {label: 'Ave reports pair as fake', value: <RiskSignalTag value={market.isFake} signalLabel='Ave reports pair as fake' />},
                    {label: 'Market reserves', value: `${exact(market.reserve0)} / ${exact(market.reserve1)}`},
                    {label: 'Market volume in USD', value: exact(market.volumeUSD)},
                    {label: 'Market cap in USD', value: exact(market.marketCapUSD)},
                    {label: 'Market FDV in USD', value: exact(market.fdvUSD)},
                    {label: 'Market created', value: formatBeijingDateTime(market.createdAt) || '-'},
                    {label: 'Market updated', value: formatBeijingDateTime(market.updatedAt) || '-'}
                ]} />}
            </> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Pair evidence is unavailable in this profile' />}
        </Card>
    );
};

export const ProjectProfileTab = (props: {project?: TokenProject; profile?: TokenProjectProfile; profileState?: string; openJSON: (content: ProjectJSONDrawerValue) => void}) => {
    const profile = props.profile;
    if (!profile) {
        return (
            <Empty
                className='project-profile-empty'
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={props.profileState === 'failed' ? 'Profile construction failed after three attempts' : 'The profile will appear after all six collection tasks reach a terminal state'}
            />
        );
    }
    const market = profile.market;
    const wallet = profile.walletSummary;
    const transactions = profile.transactions;
    const source = profile.contractSource;
    const aveRisk = market?.aveRisk;
    return (
        <div className='project-detail-tab project-profile-tab'>
            {profile.completenessStatus === 'incomplete' && <Alert type='warning' showIcon={true} title='Incomplete project profile' description={`Final failures: ${profile.failedDataTypes.join(', ') || 'unknown source'}`} />}
            <Section title='Profile overview' extra={<Button disabled={!profile.profileJSON} onClick={() => props.openJSON({title: 'Project profile JSON', value: profile.profileJSON || ''})}>View profile JSON</Button>}>
                <KeyValueGrid columns={3} items={[
                    {label: 'Completeness', value: <StatusTag value={profile.completenessStatus || 'unknown'} positive={profile.completenessStatus === 'complete'} negative={profile.completenessStatus === 'incomplete'} />},
                    {label: 'Schema version', value: profile.schemaVersion || '-'},
                    {label: 'Built', value: formatBeijingDateTime(profile.builtAt) || '-'},
                    {label: 'Failed data types', value: profile.failedDataTypes.length ? <Space wrap={true}>{profile.failedDataTypes.map(type => <Tag color='red' key={type}>{type}</Tag>)}</Space> : 'None'},
                    {label: 'Content hash', value: exact(profile.contentHash)},
                    {label: 'Created', value: formatBeijingDateTime(profile.createdAt) || '-'}
                ]} />
            </Section>
            <Section title='Market and Ave risk'>
                {market ? <>
                    <KeyValueGrid columns={3} items={[
                        {label: 'Price USD', value: currency(market.currentPriceUSD)},
                        {label: 'Price ETH', value: exact(market.currentPriceETH)},
                        {label: 'Market cap', value: currency(market.marketCapUSD)},
                        {label: 'FDV', value: currency(market.fdvUSD)},
                        {label: 'TVL', value: currency(market.tvlUSD)},
                        {label: 'Main pair TVL', value: currency(market.mainPairTVLUSD)},
                        {label: 'Holders', value: market.holders ?? '-'},
                        {label: 'Launch time', value: formatBeijingDateTime(market.launchAt) || '-'},
                        {label: 'Provider updated', value: formatBeijingDateTime(market.providerUpdatedAt) || '-'}
                    ]} />
                    {aveRisk && <KeyValueGrid columns={3} items={[
                        {label: 'Ave risk level', value: aveRisk.riskLevel ?? '-'},
                        {label: 'Ave risk score', value: exact(aveRisk.riskScore)},
                        {label: 'Ave audited', value: <StatusTag value={aveRisk.audited ? 'Audited' : 'Not audited'} positive={aveRisk.audited} negative={!aveRisk.audited} />},
                        {label: 'Ave reports token as mintable', value: <RiskSignalTag value={aveRisk.mintable} signalLabel='Ave reports token as mintable' />},
                        {label: 'Ave reports token as a honeypot', value: <RiskSignalTag value={aveRisk.honeypot} signalLabel='Ave reports token as a honeypot' />},
                        {label: 'Ave reports token as blacklisted', value: <RiskSignalTag value={aveRisk.inBlacklist} signalLabel='Ave reports token as blacklisted' />},
                        {label: 'Ave risk information', value: exact(aveRisk.riskInfo)}
                    ]} />}
                    <Button disabled={!aveRisk} onClick={() => props.openJSON({title: 'Ave risk evidence', value: aveRisk ? JSON.stringify(aveRisk, null, 2) : ''})}>View Ave risk evidence</Button>
                </> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Ave market evidence is unavailable' />}
            </Section>
            <Section title='Canonical pair profiles'>
                <div className='project-profile-pair-grid'>
                    <PairCard title='WETH / WBNB' pair={profile.wrappedNativePair} />
                    <PairCard title='USDT' pair={profile.usdtPair} />
                </div>
            </Section>
            <Section title='Wallet profile'>
                {wallet ? <KeyValueGrid columns={3} items={[
                    {label: 'Related wallets', value: wallet.walletCount ?? '-'},
                    {label: 'Native balance', value: exact(wallet.nativeBalanceTotal)},
                    {label: 'Wrapped native balance', value: exact(wallet.wrappedNativeBalanceTotal)},
                    {label: 'USDT balance', value: exact(wallet.usdtBalanceTotal)},
                    {label: 'Tracked asset value in USDT', value: exact(wallet.trackedAssetUsdtValueTotal)},
                    {label: 'Initial recipients sampled', value: wallet.initialRecipientCount ?? '-'},
                    {label: 'Deployment-time recipient allocation estimate', value: wallet.initialRecipientAllocationBPS === undefined ? '-' : `${(wallet.initialRecipientAllocationBPS / 100).toFixed(2)}%`},
                    {label: 'Wallets with at least one successful simulation call', value: wallet.walletsWithSimulationSignals ?? '-'},
                    {label: 'Roles', value: wallet.roleCounts.length ? <Space wrap={true}>{wallet.roleCounts.map(item => <Tag key={item.role}>{item.role || 'unknown'}: {item.count ?? 0}</Tag>)}</Space> : 'None'}
                ]} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Wallet evidence is unavailable' />}
                {profile.wallets.length > 0 && <div className='project-profile-evidence-grid'>
                    {profile.wallets.map(item => <Card size='small' key={item.address} title={exact(item.address)} extra={item.transactionSampleCapped ? <Tag color='gold'>300-record cap</Tag> : undefined}>
                        <KeyValueGrid columns={1} items={[
                            {label: 'Roles', value: item.roles.length ? <Space wrap={true}>{item.roles.map(role => <Tag key={role}>{role}</Tag>)}</Space> : 'None'},
                            {label: 'Deployment-time received estimate', value: item.initialRecipient?.ratioBPS === undefined ? '-' : `${(item.initialRecipient.ratioBPS / 100).toFixed(2)}%`},
                            {label: 'Recipient rank', value: item.initialRecipient?.rank ?? '-'},
                            {label: 'Native balance', value: exact(item.assets?.nativeBalance)},
                            {label: 'Wrapped native balance', value: exact(item.assets?.wrappedNativeBalance)},
                            {label: 'USDT balance', value: exact(item.assets?.usdtBalance)},
                            {label: 'Tracked asset value in USDT', value: exact(item.assets?.trackedAssetUsdtValue)},
                            {label: 'Successful simulation calls', value: item.simulation ? `${simulationSuccessCount(item.simulation)}/6` : 'Unknown'}
                        ]} />
                    </Card>)}
                </div>}
            </Section>
            <Section title='Sampled pre-deployment transactions'>
                {transactions ? <KeyValueGrid columns={3} items={[
                    {label: 'Wallets sampled', value: transactions.walletCount ?? '-'},
                    {label: 'Wallet-to-transaction associations', value: transactions.transactionAssociationCount ?? '-'},
                    {label: 'Unique transaction hashes', value: transactions.uniqueTransactionCount ?? '-'},
                    {label: 'Successful transactions', value: transactions.succeededTransactionCount ?? '-'},
                    {label: 'Failed transactions', value: transactions.failedTransactionCount ?? '-'},
                    {label: 'Native value inflow', value: exact(transactions.totalInflowNativeValue)},
                    {label: 'Native value outflow', value: exact(transactions.totalOutflowNativeValue)},
                    {label: 'Wallets at 300-record cap', value: transactions.cappedWallets.length ? transactions.cappedWallets.join(', ') : 'None'},
                    {label: 'Common methods', value: transactions.topMethods.length ? <Space wrap={true}>{transactions.topMethods.map((item, index) => <Tag key={`${item.methodID}-${item.functionName}-${index}`}>{item.functionName || item.methodID || 'unknown'}: {item.count ?? 0}</Tag>)}</Space> : 'None'},
                    {label: 'Top counterparties', value: transactions.topCounterparties.length ? <Space wrap={true}>{transactions.topCounterparties.map(item => <Tag key={item.address}>{item.address || 'unknown'}: {item.count ?? 0}</Tag>)}</Space> : 'None'}
                ]} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Transaction evidence is unavailable' />}
            </Section>
            <Section title='Contract source'>
                {source ? <KeyValueGrid columns={2} items={[
                    {label: 'Verification', value: <StatusTag value={source.verificationStatus || 'unknown'} positive={source.verificationStatus === 'verified'} />},
                    {label: 'Code hash', value: source.codeHash ? <Link to={`/token/contract-codes/${encodeURIComponent(source.codeHash)}`}>{source.codeHash}</Link> : '-'},
                    {label: 'Artifact reference', value: exact(source.artifactReference)},
                    {label: 'Deployment block', value: formatBlockNumber(props.project?.blockNumber)}
                ]} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Contract source evidence is unavailable' />}
            </Section>
            <Section title='Evidence references'>
                <div className='project-profile-evidence-grid'>
                    {profile.evidence.map(item => <Card size='small' key={item.taskID || item.dataType} title={item.dataType || 'Unknown source'} extra={<StatusTag value={item.status} positive={item.status === 'succeeded'} negative={item.status === 'failed'} />}>
                        <KeyValueGrid columns={1} items={[
                            {label: 'Task', value: item.taskID || '-'},
                            {label: 'Failures', value: `${item.failureCount || 0}/3`},
                            {label: 'Block', value: formatBlockNumber(item.blockNumber)},
                            {label: 'Collected', value: formatBeijingDateTime(item.collectedAt) || '-'},
                            {label: 'Result hash', value: exact(item.resultContentHash)},
                            {label: 'Last error', value: exact(item.lastError)}
                        ]} />
                    </Card>)}
                </div>
            </Section>
        </div>
    );
};
