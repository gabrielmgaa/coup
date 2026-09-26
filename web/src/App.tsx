import { useState, type FormEvent } from 'react'
import { useTable } from './connection'
import { Lobby } from './Lobby'
import { Table } from './Table'

export default function App() {
  const table = useTable()

  if (table.game) return <Table table={table} game={table.game} />
  if (table.lobby) return <Lobby lobby={table.lobby} send={table.send} refusal={table.refusal} />
  return <JoinForm table={table} />
}

function JoinForm({ table }: { table: ReturnType<typeof useTable> }) {
  const [name, setName] = useState('')
  const [code, setCode] = useState(new URLSearchParams(location.search).get('room') ?? '')
  const [independent, setIndependent] = useState(false)

  function enter(submitted: FormEvent) {
    submitted.preventDefault()
    const player = name.trim()
    const room = code.trim().toUpperCase()
    table.enter(
      room
        ? { type: 'join', room, name: player }
        : { type: 'create_room', name: player, options: { independent_reactions: independent } },
    )
  }

  return (
    <main className="join-screen">
      <h1>Coup</h1>
      {table.reconnecting && <p className="waiting">reconectando à mesa…</p>}
      <form onSubmit={enter}>
        <input
          value={name}
          onChange={(typed) => setName(typed.target.value)}
          placeholder="seu nome"
          maxLength={16}
          autoFocus
        />
        <input
          value={code}
          onChange={(typed) => setCode(typed.target.value.toUpperCase())}
          placeholder="código da mesa"
          maxLength={4}
        />
        <button disabled={name.trim().length < 2 || (code !== '' && code.trim().length !== 4)}>
          {code === '' ? 'abrir mesa' : 'entrar na mesa'}
        </button>
      </form>
      <p className="hint">deixe o código vazio para abrir uma mesa nova</p>
      {code === '' && (
        <label className="hint">
          <input type="checkbox" checked={independent} onChange={(ticked) => setIndependent(ticked.target.checked)} />{' '}
          regra da casa: contestar e ainda bloquear a mesma ação
        </label>
      )}
      {table.refusal && <p className="refusal">{table.refusal}</p>}
    </main>
  )
}
