import * as React from 'react';
import {Typography} from 'antd';
import type {Activity} from '../../trader-sync-models';
import {SubscriptionIdentity, subscriptionTime} from './subscription-state';
import {formatFillPrice, formatRaw} from './precision';
import {ComboConditions, MarketFacts} from './combo-conditions';
import {selectionIntersects} from './activity-feed';
export const TradeFacts = ({activity}: {activity: Activity}) => {
    const node = React.useRef<HTMLDivElement>(null);
    const latest = React.useRef(activity.metadata);
    latest.current = activity.metadata;
    const [metadata, setMetadata] = React.useState(activity.metadata);
    React.useLayoutEffect(() => {
        if (!selectionIntersects(node.current, window.getSelection())) setMetadata(activity.metadata);
    }, [activity.metadata]);
    React.useEffect(() => {
        const apply = () => {
            if (!selectionIntersects(node.current, window.getSelection())) setMetadata(latest.current);
        };
        document.addEventListener('selectionchange', apply);
        return () => document.removeEventListener('selectionchange', apply);
    }, []);
    const price = activity.priceEvidence.availability === 'available' ? formatFillPrice(activity.priceNumerator, activity.priceDenominator) : undefined;
    const copy = (value: string) => <Typography.Text copyable={{text: value}}>{value}</Typography.Text>;
    return (
        <div className='trader-sync-trade-facts'>
            <SubscriptionIdentity wallet={activity.wallet} note={activity.noteSnapshot} display={activity.targetDisplaySnapshot} />
            <p>
                <strong>{activity.side}</strong> · Settled at {subscriptionTime(activity.settledAt)}
            </p>
            <div ref={node}>
                <MarketFacts market={metadata.market} />
                <ComboConditions metadata={metadata} />
            </div>
            <dl className='trader-sync-fact-values'>
                <div>
                    <dt>Trade value</dt>
                    <dd>
                        {copy(formatRaw(activity.collateralRaw, activity.collateralDecimals))} {activity.collateralSymbol}
                    </dd>
                </div>
                <div>
                    <dt>Shares</dt>
                    <dd>{copy(formatRaw(activity.sharesRaw, activity.sharesDecimals))}</dd>
                </div>
                <div>
                    <dt>Fee</dt>
                    <dd>
                        {copy(formatRaw(activity.feeRaw, activity.collateralDecimals))} {activity.collateralSymbol}
                    </dd>
                </div>
                <div>
                    <dt>Fill price (value ÷ shares)</dt>
                    <dd>
                        {price ? (
                            <>
                                {price.approximate ? '≈ ' : ''}
                                {copy(price.text)} {activity.collateralSymbol} per share
                            </>
                        ) : (
                            `Unavailable: ${activity.priceEvidence.reasonCode || 'denominator unavailable'}`
                        )}
                    </dd>
                </div>
            </dl>
            <p>{activity.side === 'BUY' ? 'Fee is paid in addition to the trade value.' : 'Fee is deducted from the proceeds.'}</p>
            {price?.approximate && <p>Rounded to 6 decimal places, half away from zero. The exact numerator and denominator are retained below.</p>}
            {price && !price.approximate && price.text.includes('/') && <p>Below the decimal display unit; shown as an exact fraction.</p>}
            {activity.finalityAnomaly && (
                <p className='trader-sync-anomaly'>
                    <strong>Finality anomaly</strong> · Later chain evidence conflicts with the published trade. Original trade facts and notification results are retained.
                </p>
            )}
            <details>
                <summary>Original amounts and source evidence</summary>
                <dl>
                    {Object.entries({
                        'Activity ID': activity.id,
                        'Position ID': activity.positionId,
                        'Collateral raw': activity.collateralRaw,
                        'Shares raw': activity.sharesRaw,
                        'Fee raw': activity.feeRaw,
                        'Collateral decimals': String(activity.collateralDecimals),
                        'Shares decimals': String(activity.sharesDecimals),
                        'Price numerator': activity.priceNumerator,
                        'Price denominator': activity.priceDenominator,
                        'Source record ID': activity.sourceRecordId,
                        'Source version': activity.sourceVersion,
                        'Chain ID': activity.sourceLocation.chainId,
                        'Exchange address': activity.sourceLocation.exchangeAddress,
                        'Transaction hash': activity.sourceLocation.transactionHash,
                        'Block hash': activity.sourceLocation.blockHash,
                        'Block number': activity.sourceLocation.blockNumber,
                        'Log index': activity.sourceLocation.logIndex
                    }).map(([label, value]) => (
                        <div key={label}>
                            <dt>{label}</dt>
                            <dd>{copy(value)}</dd>
                        </div>
                    ))}
                </dl>
                <p>Received at: {subscriptionTime(activity.receivedAt)}</p>
                <p>Recorded at: {subscriptionTime(activity.recordedAt)}</p>
                <p>Public time unavailable: {activity.publicTimeEvidence.reasonCode || 'no confirmed public timestamp'}. Public-to-recorded latency cannot be determined.</p>
                <p>
                    Metadata source: {metadata.market.evidence.source || 'Unknown'} · Queried at: {subscriptionTime(metadata.market.evidence.queriedAt)}
                </p>
                <p>
                    Price evidence: {activity.priceEvidence.availability} · {activity.priceEvidence.reasonCode} · {activity.priceEvidence.source}
                </p>
                {activity.finalityAnomaly && (
                    <div>
                        <h3>Finality anomaly evidence</h3>
                        <p>Reason: {activity.finalityAnomaly.reason}</p>
                        <p>Detected at: {subscriptionTime(activity.finalityAnomaly.detectedAt)}</p>
                        <p>Published block hash: {copy(activity.finalityAnomaly.publishedBlockHash)}</p>
                        <p>
                            {activity.finalityAnomaly.conflictingBlockHash ? (
                                <>Conflicting block hash: {copy(activity.finalityAnomaly.conflictingBlockHash)}</>
                            ) : (
                                'Conflicting block hash unknown'
                            )}
                        </p>
                    </div>
                )}
            </details>
        </div>
    );
};
