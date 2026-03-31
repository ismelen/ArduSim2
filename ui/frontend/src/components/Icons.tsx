import React from 'react';

export const TerminalIcon: React.FC = () => (
  <svg viewBox="0 0 24 24" fill="none"
    stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <rect x="2" y="3" width="20" height="14" rx="2" />
    <path d="M8 21h8M12 17v4" />
    <polyline points="6 9 9 12 6 15" />
    <line x1="13" y1="15" x2="17" y2="15" />
  </svg>
);

export const NetworkIcon: React.FC = () => (
  <svg viewBox="0 0 24 24" fill="none"
    stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="5"  r="2" />
    <circle cx="5"  cy="19" r="2" />
    <circle cx="19" cy="19" r="2" />
    <line x1="12" y1="7"  x2="5"  y2="17" />
    <line x1="12" y1="7"  x2="19" y2="17" />
    <line x1="5"  y1="19" x2="19" y2="19" />
  </svg>
);

export const CopyIcon: React.FC = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none"
    stroke="currentColor" strokeWidth="1.5" strokeLinecap="square" strokeLinejoin="miter">
    <rect x="9" y="9" width="13" height="13" rx="1" />
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
  </svg>
);

export const CheckIcon: React.FC = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none"
    stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="20 6 9 17 4 12" />
  </svg>
);
