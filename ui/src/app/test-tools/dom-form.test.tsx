import React, {useState} from 'react';
import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

function EmailForm() {
    const [email, setEmail] = useState('');
    const [submitted, setSubmitted] = useState('');
    const invalid = email !== '' && !email.includes('@');

    return (
        <form
            onSubmit={event => {
                event.preventDefault();
                if (!invalid && email) {
                    setSubmitted(email);
                }
            }}
        >
            <label htmlFor="email">Email address</label>
            <input id="email" value={email} onChange={event => setEmail(event.target.value)} />
            {invalid && <p role="alert">Enter a valid email address.</p>}
            <button type="submit">Submit</button>
            {submitted && <p role="status">Submitted {submitted}</p>}
        </form>
    );
}

test('validates a labelled form while typing', async () => {
    const user = userEvent.setup();
    render(<EmailForm />);

    const email = screen.getByLabelText('Email address');
    await user.type(email, 'trader');
    expect(screen.getByRole('alert').textContent).toBe('Enter a valid email address.');
});

test('submits a labelled form by clicking the submit button', async () => {
    const user = userEvent.setup();
    render(<EmailForm />);

    const email = screen.getByLabelText('Email address');
    await user.type(email, 'trader@example.test');
    await user.click(screen.getByRole('button', {name: 'Submit'}));
    expect(screen.getByRole('status').textContent).toBe('Submitted trader@example.test');
});

test('submits a labelled form with the Enter key', async () => {
    const user = userEvent.setup();
    render(<EmailForm />);

    const email = screen.getByLabelText('Email address');
    await user.type(email, 'trader@example.test');
    await user.keyboard('{Enter}');
    expect(screen.getByRole('status').textContent).toBe('Submitted trader@example.test');
});

test('automatically cleans up the previous rendered form', () => {
    render(<EmailForm />);

    expect(screen.queryAllByLabelText('Email address')).toHaveLength(1);
    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.queryByRole('status')).toBeNull();
});
