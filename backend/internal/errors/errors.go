// Package errors defines application error codes independently of HTTP.
package errors

import "fmt"

type Code string

const (
	CodeNone                 Code = ""
	CodeInvalidRequest       Code = "INVALID_REQUEST"
	CodeUnauthorized         Code = "UNAUTHORIZED"
	CodeForbidden            Code = "FORBIDDEN"
	CodeNotFound             Code = "NOT_FOUND"
	CodeRateLimited          Code = "RATE_LIMITED"
	CodeServiceUnavailable   Code = "SERVICE_UNAVAILABLE"
	CodeInternal             Code = "INTERNAL_ERROR"
	CodeConfigurationInvalid Code = "CONFIGURATION_INVALID"
	CodeDatabaseUnavailable  Code = "DATABASE_UNAVAILABLE"
	CodeInvalidDestination   Code = "INVALID_DESTINATION"
	CodeInvalidMessage       Code = "INVALID_MESSAGE"
	CodeBatchUnsupported     Code = "BATCH_UNSUPPORTED"
	CodeScheduleUnsupported  Code = "SCHEDULE_UNSUPPORTED"
	CodeProviderUnauthorized Code = "PROVIDER_UNAUTHORIZED"
	CodeInsufficientBalance  Code = "INSUFFICIENT_BALANCE"
	CodeProviderRejected     Code = "PROVIDER_REJECTED"
	CodeUnknownOutcome       Code = "UNKNOWN_OUTCOME"
	CodeConflict             Code = "CONFLICT"
	CodeLastAdmin            Code = "LAST_ADMIN_REQUIRED"
	CodeCSRFInvalid          Code = "CSRF_INVALID"
)

func (c Code) Message() string {
	switch c {
	case CodeInvalidRequest:
		return "A requisição é inválida."
	case CodeUnauthorized:
		return "Autenticação necessária."
	case CodeForbidden:
		return "Acesso não permitido."
	case CodeNotFound:
		return "Recurso não encontrado."
	case CodeRateLimited:
		return "Limite de requisições excedido."
	case CodeServiceUnavailable, CodeDatabaseUnavailable:
		return "Serviço temporariamente indisponível."
	case CodeInvalidDestination:
		return "Número de destino inválido."
	case CodeInvalidMessage:
		return "Mensagem inválida."
	case CodeBatchUnsupported:
		return "Esta etapa aceita apenas uma mensagem por requisição."
	case CodeScheduleUnsupported:
		return "Agendamento não disponível nesta etapa."
	case CodeProviderUnauthorized:
		return "Falha de autenticação com o fornecedor."
	case CodeInsufficientBalance:
		return "Saldo insuficiente para o envio."
	case CodeProviderRejected:
		return "O fornecedor rejeitou o envio."
	case CodeUnknownOutcome:
		return "Não foi possível confirmar o resultado do envio. Não reenviar automaticamente."
	case CodeConflict:
		return "Já existe um registro com esses dados, ou a operação conflita com o estado atual."
	case CodeLastAdmin:
		return "Mantenha pelo menos um administrador ativo."
	case CodeCSRFInvalid:
		return "A sessão de segurança expirou. Atualize a página e tente novamente."
	case CodeConfigurationInvalid:
		return "Configuração da aplicação inválida."
	default:
		return "Ocorreu um erro interno."
	}
}

type Error struct {
	Code  Code
	cause error
}

func Wrap(code Code, cause error) *Error { return &Error{Code: code, cause: cause} }
func (e *Error) Error() string           { return fmt.Sprintf("%s: %s", e.Code, e.Code.Message()) }
func (e *Error) Unwrap() error           { return e.cause }
