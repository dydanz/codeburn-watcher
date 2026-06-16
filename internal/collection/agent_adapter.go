package collection

import "context"

// AgentAdapter is the inbound port for each agent's log source.
// Each of the six supported agents implements this interface.
// Adapters return an empty CollectResult (not an error) when their
// log source does not exist on this machine.
type AgentAdapter interface {
	Collect(ctx context.Context) (events []UsageEvent, result CollectResult, err error)
}
