import { describe, expect, it } from 'vitest';
import { AxiosError, AxiosHeaders, type AxiosResponse } from 'axios';
import { bpToPercent, errorMessage, formatPercent, formatRub, probabilityColor, rublesToKopecks } from './format';

// The server keeps money in kopecks and percentages in basis points. These
// tests make sure the conversions at the edge of the UI are exact.

describe('money', () => {
  it('turns typed rubles into whole kopecks', () => {
    expect(rublesToKopecks('150000')).toBe(15_000_000);
    expect(rublesToKopecks('12500.5')).toBe(1_250_050);
    // 0.1 + 0.2 style float errors must not leak into the database
    expect(rublesToKopecks('19.99')).toBe(1999);
    expect(rublesToKopecks(0.29)).toBe(29);
  });

  it('shows kopecks as rubles', () => {
    expect(formatRub(1_250_000)).toBe('₽12,500');
    expect(formatRub(746_250)).toBe('₽7,462.5');
    expect(formatRub(0)).toBe('₽0');
  });
});

describe('percentages', () => {
  it('turns basis points into percent', () => {
    expect(bpToPercent(3740)).toBe(37.4);
    expect(formatPercent(3740)).toBe('37.4%');
    expect(formatPercent(10000)).toBe('100%');
    expect(formatPercent(-1270)).toBe('-12.7%');
  });

  it('colours the probability by its level', () => {
    expect(probabilityColor(7000)).toBe('var(--success)'); // green from 70%
    expect(probabilityColor(4000)).toBe('var(--warning)'); // amber from 40%
    expect(probabilityColor(3999)).toBe('var(--danger)'); // red below
  });
});

describe('errorMessage', () => {
  // Builds the error axios throws when the server answers with a status.
  const serverError = (status: number, data: unknown) => {
    const response = { status, data, statusText: '', headers: {}, config: { headers: new AxiosHeaders() } } as AxiosResponse;
    return new AxiosError('Request failed', 'ERR_BAD_RESPONSE', undefined, undefined, response);
  };

  it('shows the message the Go server sends', () => {
    const err = serverError(409, { error: { code: 409, message: 'email is already registered' } });
    expect(errorMessage(err)).toBe('email is already registered');
  });

  it('explains when the server cannot be reached', () => {
    const err = new AxiosError('Network Error', 'ERR_NETWORK');
    expect(errorMessage(err)).toBe('The server is not reachable. Is it running?');
  });

  it('falls back to the given text for anything else', () => {
    expect(errorMessage(new Error('boom'), 'Login failed')).toBe('Login failed');
    expect(errorMessage(serverError(500, 'not json'))).toBe('Something went wrong');
  });
});