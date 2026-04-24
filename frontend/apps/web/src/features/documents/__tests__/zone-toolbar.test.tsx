import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ZoneToolbar } from '../zone-toolbar';

const allAllowed = {
  allowTables: true,
  allowImages: true,
  allowHeadings: true,
  allowLists: true,
};

const noneAllowed = {
  allowTables: false,
  allowImages: false,
  allowHeadings: false,
  allowLists: false,
};

describe('ZoneToolbar', () => {
  it('renders all buttons when all content types allowed', () => {
    render(<ZoneToolbar zoneId="z1" contentPolicy={allAllowed} />);
    expect(screen.getByTestId('zone-btn-table')).toBeInTheDocument();
    expect(screen.getByTestId('zone-btn-image')).toBeInTheDocument();
    expect(screen.getByTestId('zone-btn-heading')).toBeInTheDocument();
    expect(screen.getByTestId('zone-btn-list')).toBeInTheDocument();
  });

  it('hides table button when allowTables=false', () => {
    render(<ZoneToolbar zoneId="z1" contentPolicy={{ ...allAllowed, allowTables: false }} />);
    expect(screen.queryByTestId('zone-btn-table')).not.toBeInTheDocument();
    expect(screen.getByTestId('zone-btn-image')).toBeInTheDocument();
  });

  it('hides all buttons when none allowed', () => {
    render(<ZoneToolbar zoneId="z1" contentPolicy={noneAllowed} />);
    expect(screen.queryByTestId('zone-btn-table')).not.toBeInTheDocument();
    expect(screen.queryByTestId('zone-btn-image')).not.toBeInTheDocument();
    expect(screen.queryByTestId('zone-btn-heading')).not.toBeInTheDocument();
    expect(screen.queryByTestId('zone-btn-list')).not.toBeInTheDocument();
  });

  it('renders toolbar with correct testid', () => {
    render(<ZoneToolbar zoneId="my-zone" contentPolicy={allAllowed} />);
    expect(screen.getByTestId('zone-toolbar-my-zone')).toBeInTheDocument();
  });
});
