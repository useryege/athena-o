import type {Activity, MarketRef} from '../../trader-sync-models';
import {Typography} from 'antd';
export const MarketFacts = ({market}: {market: MarketRef}) => (
    <div>
        {market.evidence.availability === 'available' ? (
            <>
                <strong>{market.title || 'Market title unavailable'}</strong>
                <p>Outcome: {market.outcome || 'Unavailable'}</p>
                {market.url.trim() && (
                    <a href={market.url} target='_blank' rel='noreferrer' onClick={event => event.stopPropagation()}>
                        View market
                    </a>
                )}
            </>
        ) : (
            <p>Metadata unavailable: {market.evidence.reasonCode || 'not provided'}</p>
        )}
        {market.id && <p>Market ID: {market.id}</p>}
    </div>
);
export const ComboConditions = ({metadata}: {metadata: Activity['metadata']}) => {
    if (metadata.legsEvidence.reasonCode === 'not_combo') return null;
    const known = metadata.legsEvidence.availability === 'available';
    return (
        <div className='trader-sync-combo'>
            <h3>Combo conditions</h3>
            <p>
                {metadata.relationship === 'NOT(AND(legs))'
                    ? 'Not all conditions met'
                    : metadata.relationship === 'AND(legs)'
                      ? 'All conditions met'
                      : 'Overall condition relationship unavailable'}
            </p>
            {metadata.relationship === 'NOT(AND(legs))' && <p>NO is the complement of all listed conditions being met together. It does not mean every condition is false.</p>}
            <details>
                <summary>{known ? `${metadata.legs.length} conditions` : 'Conditions unavailable; count unknown'}</summary>
                {!known && <p>Condition list incomplete: {metadata.legsEvidence.reasonCode || 'not provided'}</p>}
                <ol>
                    {metadata.legs.map((leg, index) => (
                        <li key={`${index}-${leg.positionId}`}>
                            <MarketFacts market={leg.market} />
                            <p>
                                Position ID: <Typography.Text copyable={{text: leg.positionId}}>{leg.positionId}</Typography.Text>
                            </p>
                        </li>
                    ))}
                </ol>
            </details>
        </div>
    );
};
