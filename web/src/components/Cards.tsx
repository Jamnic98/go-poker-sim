import { SUIT_NAMES, SUIT_SYMBOLS, cardParts, isRedSuit } from '../format'

export function CardView({ token }: { token: string }) {
  const { rank, suit } = cardParts(token)
  const symbol = SUIT_SYMBOLS[suit] ?? suit
  const red = isRedSuit(suit)
  const name = SUIT_NAMES[suit] ?? suit
  return (
    <span
      className={`card${red ? ' red' : ''}`}
      title={`${rank} of ${name}`}
      data-testid={`card-${token}`}
    >
      <span className="rank">{rank}</span>
      <span className="suit">{symbol}</span>
    </span>
  )
}

export function Cards({ tokens }: { tokens: string[] }) {
  return (
    <span className="cards">
      {tokens.map((t) => (
        <CardView key={t} token={t} />
      ))}
    </span>
  )
}
