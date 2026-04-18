/**
 * @vitest-environment jsdom
 */
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { PageFooters } from '../PageFooters';

describe('PageFooters', () => {
  it('returns null when portal target is null', () => {
    const { container } = render(<PageFooters pages={3} portalTarget={null} />);
    expect(container.textContent).toBe('');
  });

  it('renders N footers via portal into target wrapper', () => {
    const target = document.createElement('div');
    document.body.appendChild(target);
    render(<PageFooters pages={3} portalTarget={target} />);
    const footers = target.querySelectorAll('[data-mddm-page-footer]');
    expect(footers).toHaveLength(3);
    expect(footers[0].textContent).toBe('Page 1');
    expect(footers[2].textContent).toBe('Page 3');
  });

  it('positions each footer at correct y', () => {
    const target = document.createElement('div');
    document.body.appendChild(target);
    render(<PageFooters pages={2} portalTarget={target} />);
    const footers = target.querySelectorAll<HTMLElement>('[data-mddm-page-footer]');
    const MM = (mm: number) => (mm / 25.4) * 96;
    const STRIDE = MM(297) + 32;
    expect(parseFloat(footers[0].style.top)).toBeCloseTo(MM(287), 2);
    expect(parseFloat(footers[1].style.top)).toBeCloseTo(STRIDE + MM(287), 2);
  });
});
