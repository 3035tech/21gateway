// Package pontal contains an isolated SMPP PDU translator based on the existing
// Node integration. It does not bind, submit, acknowledge or enable a runtime
// route. A future implementation must satisfy providers/sms.Adapter and feed
// confirmed deliver_sm events into the existing DLR/MO domains, not the HTTP
// webhook.Adapter. Supplier-specific settings and status mapping remain open.
package pontal
