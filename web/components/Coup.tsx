import { useTable } from '@/lib/connection'
import { Ending } from './Ending'
import { JoinForm } from './JoinForm'
import { Lobby } from './Lobby'
import { Table } from './Table'

export function Coup() {
  const table = useTable()

  if (table.game) return <Table table={table} game={table.game} />
  if (table.ended) return <Ending game={table.ended} onBack={table.dismissEnding} />
  if (table.lobby) return <Lobby lobby={table.lobby} send={table.send} refusal={table.refusal} />
  return <JoinForm table={table} />
}
