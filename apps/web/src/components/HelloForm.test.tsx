import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { fireEvent } from '@testing-library/react';
import { HelloForm } from './HelloForm';

vi.mock('../services/helloClient', () => ({
  helloClient: {
    sayHello: vi.fn(),
  },
}));

vi.mock('@cuckoo/api-gen/hellopb/hello', () => ({
  HelloRequest: vi.fn(),
}));

const { helloClient } = await import('../services/helloClient');

describe('HelloForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders form elements', () => {
    render(<HelloForm />);
    
    expect(screen.getByText('Hello Service')).toBeDefined();
    expect(screen.getByPlaceholderText('Enter your name')).toBeDefined();
    expect(screen.getByRole('button', { name: /say hello/i })).toBeDefined();
  });

  it('shows greeting message on successful submission', async () => {
    vi.mocked(helloClient.sayHello).mockResolvedValue({ message: 'Hello, World!' });

    render(<HelloForm />);
    
    fireEvent.change(screen.getByPlaceholderText('Enter your name'), { target: { value: 'World' } });
    fireEvent.click(screen.getByRole('button', { name: /say hello/i }));

    await waitFor(() => {
      expect(screen.getByText('Hello, World!')).toBeDefined();
    });
  });

  it('shows error message on failed submission', async () => {
    vi.mocked(helloClient.sayHello).mockRejectedValue(new Error('Network error'));

    render(<HelloForm />);
    
    fireEvent.change(screen.getByPlaceholderText('Enter your name'), { target: { value: 'Test' } });
    fireEvent.click(screen.getByRole('button', { name: /say hello/i }));

    await waitFor(() => {
      expect(screen.getByText(/failed to get greeting/i)).toBeDefined();
    });
  });

  it('shows loading state while submitting', async () => {
    vi.mocked(helloClient.sayHello).mockImplementation(
      () => new Promise((resolve) => setTimeout(() => resolve({ message: 'Hi' }), 100))
    );

    render(<HelloForm />);
    
    fireEvent.change(screen.getByPlaceholderText('Enter your name'), { target: { value: 'Test' } });
    fireEvent.click(screen.getByRole('button', { name: /say hello/i }));

    expect(screen.getByRole('button', { name: /loading/i })).toBeDefined();
  });
});