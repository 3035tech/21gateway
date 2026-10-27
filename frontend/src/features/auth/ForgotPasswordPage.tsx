import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { KeyRound, Mail } from 'lucide-react'
import { requestPasswordRecovery } from './api'
import { errorMessage } from '../../lib/admin'

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const request = useMutation({
    mutationFn: () => requestPasswordRecovery(email),
  })
  return (
    <main className="flex min-h-dvh items-center justify-center bg-slate-50 px-5 py-10">
      <section className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
        <div className="flex items-center gap-3 text-brand-purple">
          <KeyRound size={28} aria-hidden="true" />
          <span className="text-xs font-bold uppercase tracking-[0.18em]">
            Segurança da conta
          </span>
        </div>
        <h1 className="mt-5 text-3xl font-bold">Recuperar senha</h1>
        <p className="mt-3 text-sm leading-6 text-slate-600">
          Informe o e-mail da sua conta interna. Se ela existir, você receberá
          um link válido por 15 minutos.
        </p>
        {request.isSuccess ? (
          <p
            role="status"
            className="mt-6 rounded-xl bg-emerald-50 p-4 text-sm text-emerald-800"
          >
            Se o e-mail estiver cadastrado, enviaremos as instruções de
            recuperação.
          </p>
        ) : (
          <form
            className="mt-6 space-y-4"
            onSubmit={(event) => {
              event.preventDefault()
              request.mutate()
            }}
          >
            <label className="field">
              E-mail
              <input
                className="form-control"
                type="email"
                autoComplete="email"
                required
                maxLength={254}
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </label>
            {request.isError ? (
              <p role="alert" className="text-sm text-rose-700">
                {errorMessage(request.error)}
              </p>
            ) : null}
            <button
              className="button-primary w-full"
              disabled={request.isPending}
            >
              <Mail size={17} aria-hidden="true" />
              {request.isPending ? 'Enviando…' : 'Enviar link'}
            </button>
          </form>
        )}
        <Link
          to="/login"
          className="mt-6 block text-sm font-semibold text-brand-purple hover:underline"
        >
          Voltar ao login
        </Link>
      </section>
    </main>
  )
}
