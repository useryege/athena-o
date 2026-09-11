export const CursorNavigation = ({
    canPrevious,
    nextCursor,
    onPrevious,
    onNext,
    ariaLabel = 'Activity pages'
}: {
    ariaLabel?: string;
    canPrevious: boolean;
    nextCursor?: string;
    onPrevious: () => void;
    onNext: () => void;
}) => (
    <nav className='trader-sync-cursors' aria-label={ariaLabel}>
        <button type='button' disabled={!canPrevious} onClick={onPrevious}>
            Previous
        </button>
        <button type='button' disabled={!nextCursor} onClick={onNext}>
            Next
        </button>
    </nav>
);
