# `internal/protocol` — o que viaja no fio

A casca. Pega o que o cliente mandou e transforma em jogada do motor; pega o que o motor
devolveu e embrulha no envelope que sai pelo WebSocket. **Não tem regra de jogo dentro.**

Um arquivo só, `protocol.go`, e ele é pequeno de propósito.

## Os três tipos

| | |
|---|---|
| `FromClient` | tudo que chega: `type`, e os campos opcionais `name`, `action`, `target`, `card` |
| `Update` | `{"type":"update","state":…,"events":[…]}` — o snapshot e a narração, **juntos, sempre** |
| `RefusalMessage` | `{"type":"error", …}` — o `engine.Refusal` com um `type` na frente |

`ToMove` é a tradução de entrada: `"play"` vira `engine.Act`, `"lose_influence"` vira
`engine.LoseInfluence`. Nome de ação e de carta chegam em inglês e são resolvidos pelo motor
(`ActionByName`, `CharacterByName`); nome que não existe vira um refusal com a lista do que existe.

## O que não pode quebrar

- **O snapshot não é remodelado aqui.** `Update.State` é o `engine.View` inteiro, como o motor
  produziu. Se este pacote começar a escolher campos, a informação oculta passa a depender de
  duas decisões em vez de uma.
- **`events` nunca sai `null`.** `NewUpdate` troca `nil` por `[]`, porque o cliente faz `.map`
  em cima sem perguntar.
- **Decidir o que é legal não é trabalho daqui.** Este pacote recusa mensagem que não entende;
  quem recusa jogada ilegal é o motor.

## Por que existe separado

Porque o dia em que a arena de bots exigir protocolo público, `internal/protocol` vira
`protocol/` — que é um rename. O caminho contrário seria breaking change.
