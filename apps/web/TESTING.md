# Testing Guide for Web Frontend

This document provides comprehensive testing guidelines for the web frontend application.

## Test Organization

### Test Types

1. **Unit Tests** (`*.test.tsx`, `*.test.ts`)
   - Test individual components and functions
   - Fast execution (< 1 second)
   - Use React Testing Library
   - Run by default with `npm test`

2. **Component Tests**
   - Test React components in isolation
   - Test user interactions
   - Test component rendering
   - Use React Testing Library + Vitest

3. **Integration Tests**
   - Test complete user workflows
   - Test API integration
   - Test state management
   - Use React Testing Library + MSW (Mock Service Worker)

## Running Tests

### Quick Test

```bash
# Run all tests
npm test

# Run tests in watch mode
npm test -- --watch

# Run tests with coverage
npm test -- --coverage

# Run specific test file
npm test -- HelloForm.test.tsx
```

### Coverage Report

```bash
# Generate coverage report
npm test -- --coverage

# View HTML coverage report
open coverage/index.html
```

### UI Mode

Vitest provides a UI for running and debugging tests:

```bash
# Run tests in UI mode
npm test -- --ui
```

## Writing Unit Tests

### Basic Component Test

```tsx
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { HelloForm } from './HelloForm';

describe('HelloForm', () => {
  it('renders input field', () => {
    render(<HelloForm />);
    
    const input = screen.getByPlaceholderText(/enter your name/i);
    expect(input).toBeInTheDocument();
  });
  
  it('renders submit button', () => {
    render(<HelloForm />);
    
    const button = screen.getByRole('button', { name: /say hello/i });
    expect(button).toBeInTheDocument();
  });
});
```

### Testing User Interactions

```tsx
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { HelloForm } from './HelloForm';

describe('HelloForm - User Interactions', () => {
  it('calls onSubmit when form is submitted', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    
    render(<HelloForm onSubmit={onSubmit} />);
    
    const input = screen.getByPlaceholderText(/enter your name/i);
    const button = screen.getByRole('button', { name: /say hello/i });
    
    // Type in input
    await user.type(input, 'Alice');
    
    // Click button
    await user.click(button);
    
    // Verify callback was called
    expect(onSubmit).toHaveBeenCalledWith('Alice');
  });
  
  it('updates input value when typing', async () => {
    const user = userEvent.setup();
    
    render(<HelloForm />);
    
    const input = screen.getByPlaceholderText(/enter your name/i) as HTMLInputElement;
    
    await user.type(input, 'Bob');
    
    expect(input.value).toBe('Bob');
  });
});
```

### Testing Async Operations

```tsx
import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TodoList } from './TodoList';

describe('TodoList - Async Operations', () => {
  it('displays loading state', () => {
    const queryClient = new QueryClient();
    
    render(
      <QueryClientProvider client={queryClient}>
        <TodoList />
      </QueryClientProvider>
    );
    
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });
  
  it('displays todos after loading', async () => {
    const queryClient = new QueryClient();
    
    render(
      <QueryClientProvider client={queryClient}>
        <TodoList />
      </QueryClientProvider>
    );
    
    // Wait for todos to load
    await waitFor(() => {
      expect(screen.getByText(/buy groceries/i)).toBeInTheDocument();
    });
  });
  
  it('displays error message on failure', async () => {
    const queryClient = new QueryClient();
    
    // Mock API to return error
    vi.mock('./services/todoClient', () => ({
      listTodos: vi.fn().mockRejectedValue(new Error('API Error')),
    }));
    
    render(
      <QueryClientProvider client={queryClient}>
        <TodoList />
      </QueryClientProvider>
    );
    
    await waitFor(() => {
      expect(screen.getByText(/error/i)).toBeInTheDocument();
    });
  });
});
```

### Testing Custom Hooks

```tsx
import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useTodos } from './useTodos';

describe('useTodos Hook', () => {
  it('fetches todos successfully', async () => {
    const queryClient = new QueryClient();
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
    
    const { result } = renderHook(() => useTodos(), { wrapper });
    
    // Initially loading
    expect(result.current.isLoading).toBe(true);
    
    // Wait for data
    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
    
    // Verify data
    expect(result.current.data).toBeDefined();
    expect(result.current.data?.length).toBeGreaterThan(0);
  });
  
  it('handles create todo mutation', async () => {
    const queryClient = new QueryClient();
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
    
    const { result } = renderHook(() => useTodos(), { wrapper });
    
    // Create todo
    result.current.createTodo.mutate({
      title: 'New TODO',
      description: 'Test description',
    });
    
    // Wait for mutation
    await waitFor(() => {
      expect(result.current.createTodo.isSuccess).toBe(true);
    });
  });
});
```

## Mocking API Calls

### Using MSW (Mock Service Worker)

```tsx
// src/test/mocks/handlers.ts
import { rest } from 'msw';

export const handlers = [
  rest.post('/api/hello/SayHello', (req, res, ctx) => {
    return res(
      ctx.json({
        message: 'Hello, Test User!',
      })
    );
  }),
  
  rest.post('/api/todo/ListTodos', (req, res, ctx) => {
    return res(
      ctx.json({
        todos: [
          { id: '1', title: 'Test TODO 1', completed: false },
          { id: '2', title: 'Test TODO 2', completed: true },
        ],
      })
    );
  }),
];

// src/test/mocks/server.ts
import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);

// src/test/setup.ts
import { beforeAll, afterEach, afterAll } from 'vitest';
import { server } from './mocks/server';

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
```

### Using Vitest Mocks

```tsx
import { vi } from 'vitest';

// Mock entire module
vi.mock('./services/helloClient', () => ({
  sayHello: vi.fn().mockResolvedValue({
    message: 'Hello, Mock!',
  }),
}));

// Mock specific function
const mockSayHello = vi.fn();
vi.mock('./services/helloClient', () => ({
  sayHello: mockSayHello,
}));

// In test
mockSayHello.mockResolvedValue({ message: 'Hello!' });
```

## Testing Best Practices

### DO

✅ Test user behavior, not implementation
✅ Use semantic queries (getByRole, getByLabelText)
✅ Test accessibility (ARIA roles, labels)
✅ Use userEvent for interactions (not fireEvent)
✅ Wait for async operations (waitFor, findBy)
✅ Mock external dependencies (API calls)
✅ Test error states and edge cases
✅ Keep tests simple and focused
✅ Use descriptive test names
✅ Clean up after tests (cleanup, resetHandlers)

### DON'T

❌ Don't test implementation details
❌ Don't use getByTestId unless necessary
❌ Don't test third-party library code
❌ Don't write flaky tests
❌ Don't use arbitrary timeouts
❌ Don't test CSS styles
❌ Don't mock everything
❌ Don't write tests that depend on each other

## Coverage Requirements

### Targets

- **Overall coverage**: 70% minimum
- **Component coverage**: 80% minimum
- **Utility functions**: 90% minimum

### Checking Coverage

```bash
# Generate coverage report
npm test -- --coverage

# View coverage summary
npm test -- --coverage --reporter=text

# View HTML report
open coverage/index.html
```

## Testing Utilities

### Custom Render Function

```tsx
// src/test/utils.tsx
import { render, RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

export function renderWithProviders(
  ui: React.ReactElement,
  options?: RenderOptions
) {
  const queryClient = createTestQueryClient();
  
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>,
    options
  );
}

// Usage in tests
import { renderWithProviders } from './test/utils';

it('renders with providers', () => {
  renderWithProviders(<MyComponent />);
});
```

### Test Data Factories

```tsx
// src/test/factories.ts
export const createMockTodo = (overrides = {}) => ({
  id: '1',
  title: 'Test TODO',
  description: 'Test description',
  completed: false,
  createdAt: Date.now(),
  ...overrides,
});

export const createMockTodoList = (count = 3) =>
  Array.from({ length: count }, (_, i) =>
    createMockTodo({
      id: `${i + 1}`,
      title: `TODO ${i + 1}`,
    })
  );

// Usage in tests
const todo = createMockTodo({ completed: true });
const todos = createMockTodoList(5);
```

## Debugging Tests

### Using Vitest UI

```bash
npm test -- --ui
```

### Using screen.debug()

```tsx
import { render, screen } from '@testing-library/react';

it('debugs component', () => {
  render(<MyComponent />);
  
  // Print entire document
  screen.debug();
  
  // Print specific element
  screen.debug(screen.getByRole('button'));
});
```

### Using logRoles()

```tsx
import { render, logRoles } from '@testing-library/react';

it('logs available roles', () => {
  const { container } = render(<MyComponent />);
  
  logRoles(container);
});
```

## Continuous Integration

Tests run automatically in CI on:
- Every pull request
- Every commit to main branch

### CI Test Commands

```bash
# Run tests
npm test

# Run tests with coverage
npm test -- --coverage

# Verify coverage thresholds
npm test -- --coverage --reporter=json
```

## Troubleshooting

### Tests Are Slow

- Check for unnecessary waitFor calls
- Use findBy queries instead of waitFor + getBy
- Mock expensive operations
- Profile tests: `npm test -- --reporter=verbose`

### Flaky Tests

- Avoid arbitrary timeouts
- Use waitFor for async operations
- Check for race conditions
- Use deterministic data

### Low Coverage

- Run `npm test -- --coverage` to see uncovered code
- Add tests for error paths
- Add tests for edge cases
- Test user interactions

## Resources

- [React Testing Library](https://testing-library.com/react)
- [Vitest Documentation](https://vitest.dev/)
- [Testing Library Queries](https://testing-library.com/docs/queries/about)
- [Common Mistakes](https://kentcdodds.com/blog/common-mistakes-with-react-testing-library)
- [MSW Documentation](https://mswjs.io/)
