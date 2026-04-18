import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { FormRenderer } from '../src/FormRenderer';

const schema = {
  title: 'Test',
  type: 'object',
  properties: { name: { type: 'string', title: 'Name' } },
  required: ['name'],
};

describe('FormRenderer', () => {
  it('renders input for string property', () => {
    render(<FormRenderer schema={schema} formData={{}} onChange={() => {}} />);
    expect(screen.getByLabelText(/Name/)).toBeTruthy();
  });
});
